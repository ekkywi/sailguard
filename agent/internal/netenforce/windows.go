//go:build windows

package netenforce

import (
	"context"
	"log"
)

type windowsEnforcer struct{}

func New() Enforcer { return windowsEnforcer{} }

func (windowsEnforcer) Apply(ctx context.Context, spec Spec) error {
	_ = ctx
	eng, err := openEngine()
	if err != nil {
		return err
	}
	defer closeEngine(eng)

	if err := ensureProvider(eng); err != nil {
		return err
	}
	if err := ensureSublayer(eng); err != nil {
		return err
	}

	if spec.Mode != "enforce" {
		log.Printf("network enforce: mode=%s - provider ready, no block filters", spec.Mode)
		return nil
	}

	log.Printf(
		"network enforce: WFP engine ready mode=enforce version=%d rules=%d hash=%s",
		spec.Version, len(spec.SafetyAllow), len(spec.Rules), spec.Hash,
	)
	return nil
}

func (windowsEnforcer) Clear(ctx context.Context) error {
	_ = ctx
	eng, err := openEngine()
	if err != nil {
		return err
	}
	defer closeEngine(eng)
	log.Printf("network enforce: clear")
	return nil
}