package host

import "context"

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

type ProcessMonitor interface {
	List(ctx context.Context) ([]Process, error)
}

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
	Enforcer
	ServiceManager
}
