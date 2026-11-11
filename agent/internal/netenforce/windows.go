//go:build windows

package netenforce

import (
	"context"
	"log"

	"golang.org/x/sys/windows"
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
	if err := clearProviderFilters(eng); err != nil {
		return err
	}
	if spec.Mode != "enforce" {
		log.Printf("network enforce: mode=%s — cleared filters, no blocks", spec.Mode)
		return nil
	}
	if err := applySpecFilters(eng, spec); err != nil {
		return err
	}
	log.Printf(
		"network enforce: applied mode=enforce version=%d safety=%d rules=%d hash=%s",
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
	return clearProviderFilters(eng)
}

type fwpmFilter0Prefix struct {
	FilterKey   windows.GUID
	DisplayData fwpmDisplayData0
	Flags       uint32
	ProviderKey *windows.GUID
}

func filterOwnedByUs(f *fwpmFilter0Prefix) bool {
	if f == nil || f.ProviderKey == nil {
		return false
	}
	want := guidFromUUID(providerKey)
	return *f.ProviderKey == want
}
