package netenforce

import (
	"errors"
	"net"
	"testing"
	"time"
)

func fakeHostResolver(t *testing.T, now *time.Time) {
	t.Helper()
	oldDefault := defaultHostResolver
	oldLookup := lookupIP
	t.Cleanup(func() {
		defaultHostResolver = oldDefault
		lookupIP = oldLookup
	})
	defaultHostResolver = &hostResolver{
		mem: newHostMemory(),
		ttl: 10 * time.Minute,
		now: func() time.Time { return *now },
	}
}

func lookupReturns(ips ...string) func(string) ([]net.IP, error) {
	return func(string) ([]net.IP, error) {
		out := make([]net.IP, 0, len(ips))
		for _, s := range ips {
			out = append(out, net.ParseIP(s))
		}
		return out, nil
	}
}

func lookupFails(string) ([]net.IP, error) {
	return nil, errors.New("dns down")
}

func expandOne(host string) []string {
	out := expandHosts(Spec{Rules: []Rule{
		{ID: "h", Action: "deny", DestType: "host", DestValue: host},
	}})
	vals := make([]string, 0, len(out.Rules))
	for _, r := range out.Rules {
		vals = append(vals, r.DestValue)
	}
	return vals
}

func TestHostResolverRememberRotationIPs(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)

	lookupIP = lookupReturns("203.0.113.1")
	expandOne("cdn.test")

	now = now.Add(2 * time.Minute)
	lookupIP = lookupReturns("203.0.113.2")
	got := expandOne("cdn.test")

	if len(got) != 2 || got[0] != "203.0.113.1" || got[1] != "203.0.113.2" {
		t.Fatalf("want old and new IP, got %v", got)
	}
}

func TestHostResolverFallsBackWhenDNSFails(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)

	lookupIP = lookupReturns("203.0.113.1")
	expandOne("cdn.test")

	now = now.Add(time.Minute)
	lookupIP = lookupFails
	got := expandOne("cdn.test")

	if len(got) != 1 || got[0] != "203.0.113.1" {
		t.Fatalf("want remembered IP on DNS failure, got %v", got)
	}
}

func TestHostResolverDropsWhenNothingKnown(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)

	lookupIP = lookupFails
	got := expandOne("unknown.test")

	if len(got) != 0 {
		t.Fatalf("want rule skipped, got %v", got)
	}
}

func TestHostResolverExpiresAfterTTL(t *testing.T) {
	now := time.Date(2026, 11, 26, 9, 0, 0, 0, time.UTC)
	fakeHostResolver(t, &now)

	lookupIP = lookupReturns("203.0.113.1")
	expandOne("cdn.test")

	now = now.Add(11 * time.Minute)
	lookupIP = lookupReturns("203.0.113.2")
	got := expandOne("cdn.test")

	if len(got) != 1 || got[0] != "203.0.113.2" {
		t.Fatalf("want only fresh IP after TTl, got %v", got)
	}
}
