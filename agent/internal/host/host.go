package host

import (
	"context"
	"errors"
	"os"
	"strings"
)

// Process is a running process snapshot used for policy matching.
type Process struct {
	PID         uint32
	Name        string
	Path        string
	HashSHA256  string
	Publisher   string
	ProductName string
	UserName    string
}

// LabFakeProcesses returns a single fake process when SG_LAB_FAKE_PROCESS is set
// (e.g. notepad.exe). Used for audit-match smoke on hosts without real enumeration.
func LabFakeProcesses() []Process {
	name := strings.TrimSpace(os.Getenv("SG_LAB_FAKE_PROCESS"))
	if name == "" {
		return nil
	}
	return []Process{{
		PID:      4242,
		Name:     name,
		Path:     "/lab/" + name,
		UserName: "lab\\fake-user",
	}}
}

type ProcessMonitor interface {
	List(ctx context.Context) ([]Process, error)
}

type ProcessWatcher interface {
	Watch(ctx context.Context, onCreate func(Process)) error
}

var ErrWatchUnsupported = errors.New("process watch unsupported")

type Enforcer interface {
	Terminate(ctx context.Context, pid uint32) error
}

type IdentityProvider interface {
	Hostname() (string, error)
	MachineID() (string, error)
	OSVersion() (string, error)
}

// HardwareInfo is relatively stable inventory for control-plane device detail.
type HardwareInfo struct {
	CPUModel       string
	CPUCores       int
	RAMMB          int64
	DiskGB         int64
	HWManufacturer string
	HWModel        string
}

type HardwareProvider interface {
	// Hardware returns best-effort static inventory. Partial results are OK;
	// callers should ignore empty / zero fields.
	Hardware() (HardwareInfo, error)
}

// ResourceSample is a last-sample CPU/RAM/disk/uptime snapshot (heartbeat only).
type ResourceSample struct {
	UptimeSec   int64
	CPUPercent  float64
	RAMUsedMB   int64
	RAMPercent  float64
	DiskPercent float64
}

type MetricsProvider interface {
	// Metrics samples current resource use. May briefly sleep for CPU delta.
	Metrics() (ResourceSample, error)
}

type ServiceManager interface {
	// Run runs fn as a background service when supported; otherwise foreground.
	Run(ctx context.Context, fn func(context.Context) error) error
}

// Host bundles OS-specific capabilities.
type Host interface {
	OSFamily() string // windows | linux | darwin
	IdentityProvider
	HardwareProvider
	MetricsProvider
	ProcessMonitor
	ProcessWatcher
	Enforcer
	ServiceManager
}
