//go:build windows

package netenforce

import (
	"log"
	"unsafe"
)

// clearInstalledLocked deletes only filters we added (tracked keys).
// Caller must hold e.mu. Does not enumerate the system-wide WFP table.
func (e *windowsEnforcer) clearInstalledLocked() error {
	if e.eng == 0 || len(e.installed) == 0 {
		e.installed = nil
		return nil
	}

	var deleted int
	for i := range e.installed {
		key := e.installed[i]
		r1, _, _ := procFwpmFilterDeleteByKey0.Call(
			uintptr(e.eng),
			uintptr(unsafe.Pointer(&key)),
		)
		if r1 != 0 {
			log.Printf("network enforce: delete filter 0x%X", r1)
			continue
		}
		deleted++
	}
	log.Printf("network enforce: cleared %d SailGuard filter(s)", deleted)
	e.installed = nil
	return nil
}
