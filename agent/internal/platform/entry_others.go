//go:build !windows

package platform

import (
	"context"
	"os/signal"
	"syscall"
)

// RunAgent runs the agent under SIGINT/SIGTERM on non-Windows platforms.
func RunAgent(run func(ctx context.Context) error) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	return run(ctx)
}
