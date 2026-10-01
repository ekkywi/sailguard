package app

import (
	"context"
	"log"
	"time"

	"github.com/ekkywi/sailguard/agent/internal/config"
	"github.com/ekkywi/sailguard/agent/internal/host"
)

// Run is the cross-OS agent loop skeleton (enroll/policy/enforce arrive in later milestones).
func Run(ctx context.Context, cfg config.Config, h host.Host) error {
	hostname, _ := h.Hostname()
	machineID, _ := h.MachineID()
	osVersion, _ := h.OSVersion()

	log.Printf("identity hostname=%s machine_id=%s os=%s server=%s", hostname, machineID, osVersion, cfg.ServerURL)

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
				log.Printf("heartbeat stub: processes_visible=%d (policy/enforce not wired yet)", len(procs))
			}
		}
	})
}
