//go:build windows

package netenforce

import (
	"fmt"
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

func clearProviderFilters(h engineHandle) error {
	var enumHandle windows.Handle
	r1, _, _ := procFwpmFilterCreateEnumHandle0.Call(
		uintptr(h),
		0,
		uintptr(unsafe.Pointer(&enumHandle)),
	)
	if r1 != 0 {
		return fmt.Errorf("FwpmFilterCreateEnumHandle0: 0x%X", r1)
	}
	defer procFwpmFilterDestroyEnumHandle0.Call(uintptr(h), uintptr(enumHandle))

	var toDelete []windows.GUID
	for {
		var entries uintptr
		var num uint32
		r1, _, _ = procFwpmFilterEnum0.Call(
			uintptr(h),
			uintptr(enumHandle),
			100,
			uintptr(unsafe.Pointer(&entries)),
			uintptr(unsafe.Pointer(&num)),
		)
		if r1 != 0 {
			return fmt.Errorf("FwpmFilterEnum0: 0x%X", r1)
		}
		if num == 0 {
			break
		}

		ptrs := unsafe.Slice((*uintptr)(unsafe.Pointer(entries)), num)
		for _, p := range ptrs {
			if p == 0 {
				continue
			}
			f := (*fwpmFilter0Prefix)(unsafe.Pointer(p))
			if filterOwnedByUs(f) {
				toDelete = append(toDelete, f.FilterKey)
			}
		}
		procFwpmFreeMemory0.Call(entries)
	}

	var deleted int
	for i := range toDelete {
		key := toDelete[i]
		r1, _, _ = procFwpmFilterDeleteByKey0.Call(
			uintptr(h),
			uintptr(unsafe.Pointer(&key)),
		)
		if r1 != 0 {
			log.Printf("network enforce: delete filter 0x%X", r1)
			continue
		}
		deleted++
	}
	log.Printf("network enforce: cleared %d SailGuard filter(s)", deleted)
	return nil
}
