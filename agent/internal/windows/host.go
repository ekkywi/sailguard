//go:build windows

package windows

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
	"unsafe"

	"github.com/ekkywi/sailguard/agent/internal/host"
	"golang.org/x/sys/windows"
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
	if procs := host.LabFakeProcesses(); procs != nil {
		return procs, nil
	}

	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))

	err = windows.Process32First(snap, &pe)
	if err != nil {
		return nil, fmt.Errorf("Process32First: %w", err)
	}

	out := make([]host.Process, 0, 128)
	for {
		name := windows.UTF16ToString(pe.ExeFile[:])
		if name != "" {
			out = append(out, host.Process{
				PID:  pe.ProcessID,
				Name: name,
			})
		}
		err = windows.Process32Next(snap, &pe)
		if err != nil {
			if err == windows.ERROR_NO_MORE_FILES {
				break
			}
			return nil, fmt.Errorf("Process32Next: %w", err)
		}
	}
	return out, nil
}

func (h *Host) Terminate(ctx context.Context, pid uint32) error {
	_ = ctx
	if pid == 0 {
		return fmt.Errorf("windows terminate: invalid pid 0")
	}
	handle, err := windows.OpenProcess(windows.PROCESS_TERMINATE, false, pid)
	if err != nil {
		return fmt.Errorf("OpenProcess(%d): %w", pid, err)
	}
	defer windows.CloseHandle(handle)
	if err := windows.TerminateProcess(handle, 1); err != nil {
		return fmt.Errorf("TerminateProcess(%d): %w", pid, err)
	}
	return nil
}

func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	// Windows Service wrapper lands later; foreground for now.
	return fn(ctx)
}

func (h *Host) Watch(ctx context.Context, onCreate func(host.Process)) error {
	if onCreate == nil {
		return fmt.Errorf("windows watch: onCreate is nil")
	}

	seen := make(map[uint32]struct{})
	if procs, err := h.List(ctx); err == nil {
		for _, p := range procs {
			seen[p.PID] = struct{}{}
		}
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			procs, err := h.List(ctx)
			if err != nil {
				log.Printf("windows watch list: %v", err)
				continue
			}
			live := make(map[uint32]struct{}, len(procs))
			for _, p := range procs {
				live[p.PID] = struct{}{}
				if _, ok := seen[p.PID]; ok {
					continue
				}
				seen[p.PID] = struct{}{}
				onCreate(p)
			}
			for pid := range seen {
				if _, ok := live[pid]; !ok {
					delete(seen, pid)
				}
			}
		}
	}
}
