//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

type Host struct{}

func New() host.Host { return &Host{} }

func (h *Host) OSFamily() string { return "linux" }

func (h *Host) Hostname() (string, error) { return os.Hostname() }

func (h *Host) MachineID() (string, error) {
	return "linux-stub", nil
}

func (h *Host) OSVersion() (string, error) {
	return "Linux (stub)", nil
}

func (h *Host) List(ctx context.Context) ([]host.Process, error) {
	_ = ctx
	return nil, fmt.Errorf("linux process monitor: stub not implemented")
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	return fmt.Errorf("linux enforce: stub not implemented (pid=%d)", pid)
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
