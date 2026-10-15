package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
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
		scanEvery := cfg.ScanInterval
		if scanEvery <= 0 {
			scanEvery = 3 * time.Second
		}
		hbEvery := cfg.HeartbeatInterval
		if hbEvery <= 0 {
			hbEvery = 30 * time.Second
		}

		scanTicker := time.NewTicker(scanEvery)
		defer scanTicker.Stop()
		hbTicker := time.NewTicker(hbEvery)
		defer hbTicker.Stop()

		log.Printf("timers scan %s heartbeat %s", scanEvery, hbEvery)

		recent := &recentSet{}

		syncPolicy(ctx, client, cred, dataDir)
		scanAndReport(ctx, client, cred, h, dataDir, recent)
		sendHeartbeat(ctx, client, cred, cfg, hostname, osVersion)

		go func() {
			err := h.Watch(ctx, func(p host.Process) {
				handleProcessEvent(ctx, client, cred, h, dataDir, p, recent, "watch")
			})
			if errors.Is(err, host.ErrWatchUnsupported) {
				log.Printf("process watch: unsupported — poll fallback only")
				return
			}
			if err != nil && ctx.Err() == nil {
				log.Printf("process watch stopped: %v (poll continues)", err)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-scanTicker.C:
				scanAndReport(ctx, client, cred, h, dataDir, recent)
			case <-hbTicker.C:
				syncPolicy(ctx, client, cred, dataDir)
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
		if strings.Contains(err.Error(), "device_exists") {
			return nil, fmt.Errorf(
				"enroll: device already registered on the server — reuse existing %s "+
					"(keep data_dir / ProgramData) or remove the device in the lab DB before re-enrolling: %w",
				credstore.Path(dataDir),
				err,
			)
		}
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
	recent *recentSet,
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
		ev, ok := evaluateAndReport(ctx, ep, p, h, now, recent)
		if !ok {
			continue
		}
		events = append(events, ev)
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

func evaluateAndReport(
	ctx context.Context,
	ep *policypkg.EffectivePolicy,
	p host.Process,
	h host.Host,
	now string,
	recent *recentSet,
) (agentcontract.Event, bool) {
	if policypkg.IsCriticalProcess(p.Name) {
		return agentcontract.Event{}, false
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
		return agentcontract.Event{}, false
	}

	eventType := agentcontract.EventViolationDetected
	clientPrefix := "vd"
	if ep.Mode == policypkg.ModeEnforce {
		eventType = agentcontract.EventViolationBlocked
		clientPrefix = "vb"
	}

	dedupeKey := fmt.Sprintf("%s:%d:%s:%s", ep.Hash, p.PID, matched.ID, eventType)
	// Shared with Watch so poll reconciliation does not double-terminate / double-report.
	if recent != nil && recent.seenRecently(dedupeKey, 45*time.Second) {
		return agentcontract.Event{}, false
	}

	var enf *agentcontract.Enforcement
	if ep.Mode == policypkg.ModeEnforce {
		errTerm := h.Terminate(ctx, p.PID)
		enf = &agentcontract.Enforcement{
			Result:   "terminated",
			Attempts: 1,
		}
		if errTerm != nil {
			msg := errTerm.Error()
			enf.Result = "terminate_failed"
			enf.Error = &msg
			log.Printf("enforce terminate pid=%d name=%s: %v", p.PID, p.Name, errTerm)
		}
	}

	ev := agentcontract.Event{
		ClientEventID: fmt.Sprintf("%s-%d-%d", clientPrefix, p.PID, time.Now().UnixNano()),
		EventType:     eventType,
		OccurredAt:    now,
		DedupeKey:     dedupeKey,
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
		Enforcement: enf,
	}
	return ev, true
}

type recentSet struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func (r *recentSet) seenRecently(key string, window time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.m == nil {
		r.m = make(map[string]time.Time)
	}
	now := time.Now()
	for k, t := range r.m {
		if now.Sub(t) > window {
			delete(r.m, k)
		}
	}
	if t, ok := r.m[key]; ok && now.Sub(t) <= window {
		return true
	}
	r.m[key] = now
	return false
}

func flushEvents(
	ctx context.Context,
	client *controlplane.Client,
	cred *credstore.Credential,
	events []agentcontract.Event,
	label string,
) {
	if len(events) == 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := client.PostEvents(ctx, cred.DeviceToken, agentcontract.EventBatchRequest{
		SchemaVersion: 1,
		DeviceID:      cred.DeviceID,
		SentAt:        now,
		Events:        events,
	})
	if err != nil {
		log.Printf("%s flush failed: %v", label, err)
		return
	}
	log.Printf("%s: violations=%d accepted=%d rejected=%d",
		label, len(events), res.Accepted, len(res.Rejected))
}

func handleProcessEvent(
	ctx context.Context,
	client *controlplane.Client,
	cred *credstore.Credential,
	h host.Host,
	dataDir string,
	p host.Process,
	recent *recentSet,
	label string,
) {
	ep, err := policycache.Load(dataDir)
	if err != nil || ep == nil || len(ep.Rules) == 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ev, ok := evaluateAndReport(ctx, ep, p, h, now, recent)
	if !ok {
		return
	}
	flushEvents(ctx, client, cred, []agentcontract.Event{ev}, label)
}
