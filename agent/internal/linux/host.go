//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
	if procs := host.LabFakeProcesses(); procs != nil {
		return procs, nil
	}

	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	out := make([]host.Process, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		pid64, err := strconv.ParseUint(e.Name(), 10, 32)
		if err != nil {
			continue
		}

		name, path := readProcIdentity(e.Name())
		if name == "" {
			continue
		}

		out = append(out, host.Process{
			PID:  uint32(pid64),
			Name: name,
			Path: path,
		})
	}
	return out, nil
}

func readProcIdentity(pid string) (name, path string) {
	base := filepath.Join("/proc", pid)

	if b, err := os.ReadFile(filepath.Join(base, "comm")); err == nil {
		name = strings.TrimSpace(string(b))
	}

	if b, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil && len(b) > 0 {
		parts := strings.Split(string(b), "\x00")
		if len(parts) > 0 && parts[0] != "" {
			path = parts[0]
			if name == "" {
				name = filepath.Base(parts[0])
			}
		}
	}

	if name == "" {
		return "", ""
	}
	return name, path
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	return fmt.Errorf("linux enforce: stub not implemented (pid=%d)", pid)
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
