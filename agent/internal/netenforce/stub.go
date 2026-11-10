//go:build !windows

package netenforce

import (
	"context"
	"log"
)

type stubEnforcer struct{}

func New() Enforcer { return stubEnforcer{} }

func (stubEnforcer) Apply(ctx context.Context, spec Spec) error {
	_ = ctx
	log.Printf(
		"network enforce stub: mode=%s version=%d safety=%d rules=%d hash=%s (no OS filters yet)",
		spec.Mode, spec.Version, len(spec.SafetyAllow), len(spec.Rules), spec.Hash,
	)
	return nil
}

func (stubEnforcer) Clear(ctx context.Context) error {
	_ = ctx
	log.Printf("network enforce stub: clear (no-op)")
	return nil
}
