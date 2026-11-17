//go:build windows

package netenforce

import (
	"context"
	"log"
	"sync"

	"golang.org/x/sys/windows"
)

// Singleton so Apply keeps one DYNAMIC WFP session open for the process lifetime.
// Closing the engine after each Apply would delete all filters immediately.
var globalWindowsEnforcer = &windowsEnforcer{}

type windowsEnforcer struct {
	mu        sync.Mutex
	eng       engineHandle
	installed []windows.GUID
}

func New() Enforcer { return globalWindowsEnforcer }

func (e *windowsEnforcer) ensureEngine() error {
	if e.eng != 0 {
		return nil
	}
	h, err := openEngine()
	if err != nil {
		return err
	}
	if err := ensureProvider(h); err != nil {
		closeEngine(h)
		return err
	}
	if err := ensureSublayer(h); err != nil {
		closeEngine(h)
		return err
	}
	e.eng = h
	log.Printf("network enforce: WFP engine opened (dynamic session held)")
	return nil
}

func (e *windowsEnforcer) Apply(ctx context.Context, spec Spec) error {
	_ = ctx
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.ensureEngine(); err != nil {
		return err
	}

	if err := e.clearInstalledLocked(); err != nil {
		return err
	}

	if spec.Mode != "enforce" {
		log.Printf("network enforce: mode=%s — no block filters installed", spec.Mode)
		return nil
	}

	spec = expandHosts(spec)
	if err := e.applySpecFiltersLocked(spec); err != nil {
		return err
	}
	log.Printf(
		"network enforce: applied mode=enforce version=%d safety=%d rules=%d installed=%d hash=%s",
		spec.Version, len(spec.SafetyAllow), len(spec.Rules), len(e.installed), spec.Hash,
	)
	return nil
}

func (e *windowsEnforcer) Clear(ctx context.Context) error {
	_ = ctx
	e.mu.Lock()
	defer e.mu.Unlock()

	log.Printf("network enforce: clear")
	if e.eng == 0 {
		e.installed = nil
		return nil
	}
	_ = e.clearInstalledLocked()
	closeEngine(e.eng)
	e.eng = 0
	log.Printf("network enforce: WFP engine closed")
	return nil
}
