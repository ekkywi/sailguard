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
	"github.com/ekkywi/sailguard/pkgs/agentcontract"
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

	return h.Run(ctx, func(ctx context.Context) error {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
				procs, err := h.List(ctx)
				if err != nil {
					log.Printf("process list: %v", err)
					continue
				}
				log.Printf(
					"heartbeat stub: device_id=%s processes_visible=%d (server heartbeat not wired yet)",
					cred.DeviceID, len(procs),
				)
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
			"no local crendential; set SG_ENROLLMENT_TOKEN to enroll once " +
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
	log.Printf("enroll ok: crendential saved to %s", credstore.Path(dataDir))
	return cred, nil
}
