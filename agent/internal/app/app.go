package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/config"
	"github.com/ekkywi/sailguard/agent/internal/controlplane"
	"github.com/ekkywi/sailguard/agent/internal/credstore"
	"github.com/ekkywi/sailguard/agent/internal/host"
	"github.com/ekkywi/sailguard/agent/internal/policycache"
	"github.com/ekkywi/sailguard/pkgs/agentcontract"
	policypkg "github.com/ekkywi/sailguard/pkgs/policy"
)

func Run(ctx context.Context, cfg config.Config, h host.Host) error {
	hostname, err := h.Hostname()
	if err != nil {
		return fmt.Errorf("hostname: %w", err)
	}
	machineID, err := h.MachineID()
	if err != nil {
		return fmt.Errorf("machine id: %w", err)
	}
	osVersion, err := h.OSVersion()
	if err != nil {
		return fmt.Errorf("os version: %w", err)
	}

	dataDir := credstore.ResolveDir(cfg.DataDir)
	log.Printf(
		"identity hostname=%s machine_id=%s os=%s/%s server=%s data_dir=%s",
		hostname, machineID, h.OSFamily(), osVersion, cfg.ServerURL, dataDir,
	)

	cred, err := ensureEnrolled(ctx, cfg, h, dataDir, hostname, machineID, osVersion)
	if err != nil {
		return err
	}
	log.Printf("enrolled as device_id=%s", cred.DeviceID)

	serverURL := cfg.ServerURL
	if cred.ServerURL != "" {
		serverURL = cred.ServerURL
	}
	client := controlplane.NewClient(serverURL)

	return h.Run(ctx, func(ctx context.Context) error {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		syncPolicy(ctx, client, cred, dataDir)
		scanAndReport(ctx, client, cred, h, dataDir)
		sendHeartbeat(ctx, client, cred, cfg, hostname, osVersion)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				syncPolicy(ctx, client, cred, dataDir)
				scanAndReport(ctx, client, cred, h, dataDir)
				sendHeartbeat(ctx, client, cred, cfg, hostname, osVersion)
			}
		}
	})
}

func ensureEnrolled(
	ctx context.Context,
	cfg config.Config,
	h host.Host,
	dataDir, hostname, machineID, osVersion string,
) (*credstore.Credential, error) {
	cred, err := credstore.Load(dataDir)
	if err == nil {
		log.Printf("loaded existing credential from %s", credstore.Path(dataDir))
		return cred, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("load credential: %w", err)
	}

	token := cfg.EnrollmentToken
	if token == "" {
		return nil, errors.New(
			"no local credential; set SG_ENROLLMENT_TOKEN to enroll once " +
				"(or point SG_DATA_DIR at a directory that already has device.json)",
		)
	}

	client := controlplane.NewClient(cfg.ServerURL)
	res, err := client.Enroll(ctx, agentcontract.EnrollRequest{
		SchemaVersion:   1,
		EnrollmentToken: token,
		Hostname:        hostname,
		MachineGUID:     machineID,
		OSFamily:        h.OSFamily(),
		OSVersion:       osVersion,
		AgentVersion:    cfg.AgentVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("enroll: %w", err)
	}

	cred = &credstore.Credential{
		DeviceID:    res.DeviceID,
		DeviceToken: res.DeviceToken,
		ServerURL:   cfg.ServerURL,
	}
	if err := credstore.Save(dataDir, *cred); err != nil {
		return nil, fmt.Errorf("save credential: %w", err)
	}
	log.Printf("enroll ok: credential saved to %s", credstore.Path(dataDir))
	return cred, nil
}

func sendHeartbeat(
	ctx context.Context,
	client *controlplane.Client,
	cred *credstore.Credential,
	cfg config.Config,
	hostname, osVersion string,
) {
	now := time.Now().UTC().Format(time.RFC3339)
	req := agentcontract.EventBatchRequest{
		SchemaVersion: 1,
		DeviceID:      cred.DeviceID,
		SentAt:        now,
		Events: []agentcontract.Event{
			{
				ClientEventID: fmt.Sprintf("hb-%d", time.Now().UnixNano()),
				EventType:     agentcontract.EventHeartbeat,
				OccurredAt:    now,
				Hostname:      hostname,
				OSVersion:     osVersion,
				AgentVersion:  cfg.AgentVersion,
			},
		},
	}

	res, err := client.PostEvents(ctx, cred.DeviceToken, req)
	if err != nil {
		log.Printf("heartbeat failed: device_id=%s err=%v", cred.DeviceID, err)
		return
	}
	log.Printf(
		"heartbeat ok: device_id=%s accepted=%d rejected=%d",
		cred.DeviceID, res.Accepted, len(res.Rejected),
	)
}

func syncPolicy(
	ctx context.Context,
	client *controlplane.Client,
	cred *credstore.Credential,
	dataDir string,
) {
	currentHash := policycache.Hash(dataDir)
	data, err := client.GetPolicy(ctx, cred.DeviceToken, currentHash)
	if err != nil {
		log.Printf("policy sync failed: device_id=%s err=%v", cred.DeviceID, err)
		return
	}

	if data.Unchanged {
		log.Printf("policy unchanged: device_id=%s hash=%s", cred.DeviceID, currentHash)
		return
	}

	ep := policypkg.EffectivePolicy{
		SchemaVersion: data.SchemaVersion,
		Version:       data.Version,
		ComputedAt:    data.ComputedAt,
		Mode:          data.Mode,
		Hash:          data.Hash,
		Rules:         data.Rules,
	}

	if err := policycache.Save(dataDir, ep); err != nil {
		log.Printf("policy cache save failed: %v", err)
		return
	}
	log.Printf(
		"policy updated: device_id=%s version=%d mode=%s rules=%d hash=%s path=%s",
		cred.DeviceID, ep.Version, ep.Mode, len(ep.Rules), ep.Hash, policycache.Path(dataDir),
	)
}

func scanAndReport(
	ctx context.Context,
	client *controlplane.Client,
	cred *credstore.Credential,
	h host.Host,
	dataDir string,
) {
	ep, err := policycache.Load(dataDir)
	if err != nil {
		log.Printf("policy load failed: %v", err)
		return
	}
	if ep == nil || len(ep.Rules) == 0 {
		log.Printf("audit scan skip: no cached policy rules")
		return
	}

	procs, err := h.List(ctx)
	if err != nil {
		log.Printf("process list: %v", err)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	events := make([]agentcontract.Event, 0)

	for _, p := range procs {
		if policypkg.IsCriticalProcess(p.Name) {
			continue
		}

		attrs := policypkg.ProcessAttrs{
			Name:        p.Name,
			Path:        p.Path,
			HashSHA256:  p.HashSHA256,
			Publisher:   p.Publisher,
			ProductName: p.ProductName,
		}

		matched, allow := policypkg.Evaluate(*ep, attrs)
		if matched == nil || allow {
			continue
		}

		events = append(events, agentcontract.Event{
			ClientEventID: fmt.Sprintf("vd-%d-%d", p.PID, time.Now().UnixNano()),
			EventType:     agentcontract.EventViolationDetected,
			OccurredAt:    now,
			DedupeKey:     fmt.Sprintf("%s:%d:%s", ep.Hash, p.PID, matched.ID),
			Process: &agentcontract.ProcessInfo{
				PID:         p.PID,
				Name:        p.Name,
				Path:        p.Path,
				HashSHA256:  p.HashSHA256,
				Publisher:   p.Publisher,
				ProductName: p.ProductName,
				UserName:    p.UserName,
			},
			Match: &agentcontract.MatchInfo{
				RuleID:       matched.ID,
				Action:       string(matched.Action),
				RuleType:     string(matched.RuleType),
				Value:        matched.Value,
				CategoryCode: matched.CategoryCode,
			},
			Policy: &agentcontract.PolicyRef{
				Version: ep.Version,
				Mode:    string(ep.Mode),
			},
		})
	}

	if len(events) == 0 {
		log.Printf("audit scan: processes=%d violations=0", len(procs))
		return
	}

	res, err := client.PostEvents(ctx, cred.DeviceToken, agentcontract.EventBatchRequest{
		SchemaVersion: 1,
		DeviceID:      cred.DeviceID,
		SentAt:        now,
		Events:        events,
	})
	if err != nil {
		log.Printf("violation flush failed: %v", err)
		return
	}
	log.Printf(
		"audit scan: processes=%d violations=%d accepted=%d rejected=%d",
		len(procs), len(events), res.Accepted, len(res.Rejected),
	)
}
