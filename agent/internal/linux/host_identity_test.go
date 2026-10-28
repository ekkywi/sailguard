//go:build linux

package linux

import "testing"

func TestPreferCmdlineBaseName_TruncatedComm(t *testing.T) {
	if !preferCmdlineBaseName("gnome-calculato", "gnome-calculator") {
		t.Fatal("expected truncated gnome-calculator comm to prefer cmdline basename")
	}
}

func TestPreferCmdlineBaseName_ShortCommUnchanged(t *testing.T) {
	// "sh" must not become "shutdown" via HasPrefix alone.
	if preferCmdlineBaseName("sh", "shutdown") {
		t.Fatal("short comm must not be replaced by longer prefix basename")
	}
}

func TestIdentityFromCommAndCmdline(t *testing.T) {
	name, path := identityFromCommAndCmdline("gnome-calculato", "/usr/bin/gnome-calculator")
	if name != "gnome-calculator" {
		t.Fatalf("name=%q want gnome-calculator", name)
	}
	if path != "/usr/bin/gnome-calculator" {
		t.Fatalf("path=%q", path)
	}
}
