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
	return readMachineID()
}

func (h *Host) OSVersion() (string, error) {
	return readOSVersion(), nil
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

	var comm string
	if b, err := os.ReadFile(filepath.Join(base, "comm")); err == nil {
		comm = strings.TrimSpace(string(b))
	}

	var argv0 string
	if b, err := os.ReadFile(filepath.Join(base, "cmdline")); err == nil && len(b) > 0 {
		parts := strings.Split(string(b), "\x00")
		if len(parts) > 0 {
			argv0 = parts[0]
		}
	}

	return identityFromCommAndCmdline(comm, argv0)
}

// linuxMaxCommLen is TASK_COMM_LEN-1: /proc/<pid>/comm truncates names longer than this.
const linuxMaxCommLen = 15

func identityFromCommAndCmdline(comm, argv0 string) (name, path string) {
	name = strings.TrimSpace(comm)
	path = argv0
	if path == "" {
		return name, path
	}
	baseName := filepath.Base(path)
	if preferCmdlineBaseName(name, baseName) {
		name = baseName
	}
	if name == "" {
		return "", path
	}
	return name, path
}

// preferCmdlineBaseName upgrades a truncated /proc/comm value (e.g. "gnome-calculato")
// to the full executable basename from cmdline ("gnome-calculator").
func preferCmdlineBaseName(comm, baseName string) bool {
	if baseName == "" {
		return false
	}
	if comm == "" {
		return true
	}
	// Only replace when comm looks truncated and is a prefix of the real basename.
	// Avoid rewriting short names like "sh" → "shutdown".
	return len(comm) == linuxMaxCommLen &&
		len(baseName) > len(comm) &&
		strings.HasPrefix(baseName, comm)
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	if pid == 0 || pid == 1 {
		return fmt.Errorf("linux terminate: refusing pid %d", pid)
	}
	proc, err := os.FindProcess(int(pid))
	if err != nil {
		return err
	}
	if err := proc.Signal(os.Kill); err != nil { //SIGKILL
		return fmt.Errorf("kill(%d): %w", pid, err)
	}
	return nil
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (h *Host) Watch(ctx context.Context, onCreate func(host.Process)) error {
	_ = ctx
	_ = onCreate
	return host.ErrWatchUnsupported
}
