//go:build darwin

package darwin

import (
	"context"
	"fmt"
	"os"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

type Host struct{}

func New() host.Host { return &Host{} }

func (h *Host) OSFamily() string { return "darwin" }

func (h *Host) Hostname() (string, error) { return os.Hostname() }

func (h *Host) MachineID() (string, error) {
	return "darwin-stub", nil
}

func (h *Host) OSVersion() (string, error) {
	return "macOS (stub)", nil
}

func (h *Host) List(ctx context.Context) ([]host.Process, error) {
	_ = ctx
	if procs := host.LabFakeProcesses(); procs != nil {
		return procs, nil
	}
	return nil, fmt.Errorf("darwin process monitor: stub not implemented (set SG_LAB_FAKE_PROCESS for lab)")
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	return fmt.Errorf("darwin enforce: stub not implemented (pid=%d)", pid)
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
