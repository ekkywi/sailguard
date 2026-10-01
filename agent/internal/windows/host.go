//go:build windows

package windows

import (
	"context"
	"fmt"
	"os"

	"github.com/ekkywi/sailguard/agent/internal/host"
)

type Host struct{}

func New() host.Host { return &Host{} }

func (h *Host) OSFamily() string { return "windows" }

func (h *Host) Hostname() (string, error) { return os.Hostname() }

func (h *Host) MachineID() (string, error) {
	// MVP placeholder: real MachineGuid from registry comes in M2.
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	return "win-pending:" + name, nil
}

func (h *Host) OSVersion() (string, error) {
	return "Windows (detail pending)", nil
}

func (h *Host) List(ctx context.Context) ([]host.Process, error) {
	_ = ctx
	// Real Toolhelp32 enumeration lands in M4.
	return nil, fmt.Errorf("windows process enumeration: not implemented yet")
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	return fmt.Errorf("windows terminate pid=%d: not implemented yet", pid)
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	// Windows Service wrapper lands later; foreground for now.
	return fn(ctx)
}
