package netenforce

import (
	"net"
	"testing"
)

func TestExpandHostsKeepsIPAndCIDR(t *testing.T) {
	t.Parallel()
	spec := Spec{
		Rules: []Rule{
			{ID: "1", Action: "deny", DestType: "ip", DestValue: "1.1.1.1"},
			{ID: "2", Action: "deny", DestType: "cidr", DestValue: "10.0.0.0/8"},
		},
	}
	out := expandHosts(spec)
	if len(out.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(out.Rules))
	}
	if out.Rules[0].DestType != "ip" || out.Rules[1].DestType != "cidr" {
		t.Fatalf("ip/cidr should be unchanged: %+v", out.Rules)
	}
}

func TestExpandHostsResolvesLocalhost(t *testing.T) {
	t.Parallel()
	spec := Spec{
		Rules: []Rule{
			{ID: "h1", Action: "deny", DestType: "host", DestValue: "localhost", Label: "loop"},
		},
	}
	out := expandHosts(spec)
	if len(out.Rules) == 0 {
		t.Fatal("expected at least one IPv4 from localhost")
	}
	for _, r := range out.Rules {
		if r.DestType != "ip" {
			t.Fatalf("expected ip after expand, got %s", r.DestType)
		}
		if r.Action != "deny" {
			t.Fatalf("action changed: %s", r.Action)
		}
	}
}

func TestExpandHostsUsesLookup(t *testing.T) {
	old := lookupIP
	lookupIP = func(string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("203.0.113.10"), net.ParseIP("203.0.113.11")}, nil
	}
	defer func() { lookupIP = old }()

	spec := Spec{Rules: []Rule{
		{ID: "h1", Action: "deny", DestType: "host", DestValue: "example.test"},
	}}
	out := expandHosts(spec)

	if len(out.Rules) != 2 {
		t.Fatalf("want 2 rules, got %d", len(out.Rules))
	}
	if out.Rules[0].DestValue != "203.0.113.10" || out.Rules[1].DestValue != "203.0.113.11" {
		t.Fatalf("unexpected IPs: %+v", out.Rules)
	}
}
