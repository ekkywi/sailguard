package host

import (
	"context"
	"os"
	"strings"
	"errors"
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
		PID:  4242,
		Name: name,
		Path: "/lab/" + name,
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

type ServiceManager interface {
	// Run runs fn as a background service when supported; otherwise foreground.
	Run(ctx context.Context, fn func(context.Context) error) error
}

// Host bundles OS-specific capabilities.
type Host interface {
	OSFamily() string // windows | linux | darwin
	IdentityProvider
	ProcessMonitor
	ProcessWatcher
	Enforcer
	ServiceManager
}
