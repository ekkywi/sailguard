//go:build windows

package windows

import (
	"context"
)

// Run executes fn in the current process. Windows Service SCM handoff happens
// earlier in platform.RunAgent so the service can report Running before enroll.
func (h *Host) Run(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
