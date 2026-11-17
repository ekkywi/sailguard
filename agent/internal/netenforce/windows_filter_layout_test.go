//go:build windows && (amd64 || arm64)

package netenforce

import (
	"testing"
	"unsafe"
)

// Lock FWPM_* amd64/arm64 ABI against WireGuard / Windows SDK offsets.
func TestWFPFilterLayout(t *testing.T) {
	if got, want := unsafe.Sizeof(fwpAction0{}), uintptr(20); got != want {
		t.Fatalf("fwpAction0 size=%d want %d", got, want)
	}
	if got, want := unsafe.Offsetof(fwpAction0{}.FilterType), uintptr(4); got != want {
		t.Fatalf("fwpAction0.FilterType offset=%d want %d", got, want)
	}

	if got, want := unsafe.Sizeof(fwpValue0{}), uintptr(16); got != want {
		t.Fatalf("fwpValue0 size=%d want %d", got, want)
	}
	if got, want := unsafe.Offsetof(fwpValue0{}.Value), uintptr(8); got != want {
		t.Fatalf("fwpValue0.Value offset=%d want %d", got, want)
	}

	if got, want := unsafe.Sizeof(fwpmFilterCondition0{}), uintptr(40); got != want {
		t.Fatalf("fwpmFilterCondition0 size=%d want %d", got, want)
	}
	if got, want := unsafe.Offsetof(fwpmFilterCondition0{}.ConditionValue), uintptr(24); got != want {
		t.Fatalf("fwpmFilterCondition0.ConditionValue offset=%d want %d", got, want)
	}

	if got, want := unsafe.Sizeof(fwpmFilter0{}), uintptr(200); got != want {
		t.Fatalf("fwpmFilter0 size=%d want %d", got, want)
	}
	checks := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"Action", unsafe.Offsetof(fwpmFilter0{}.Action), 128},
		{"ProviderContextKey", unsafe.Offsetof(fwpmFilter0{}.ProviderContextKey), 152},
		{"Reserved", unsafe.Offsetof(fwpmFilter0{}.Reserved), 168},
		{"FilterID", unsafe.Offsetof(fwpmFilter0{}.FilterID), 176},
		{"EffectiveWeight", unsafe.Offsetof(fwpmFilter0{}.EffectiveWeight), 184},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("fwpmFilter0.%s offset=%d want %d", c.name, c.got, c.want)
		}
	}
}
