package netenforce

import "testing"

func TestCotrolPlaneSafetyRuleIP(t *testing.T) {
	t.Parallel()
	r, ok := controlPlaneSafetyRule("http://172.31.254.3:18080")
	if !ok {
		t.Fatal("expected ok")
	}
	if r.DestType != "ip" || r.DestValue != "172.31.254.3" {
		t.Fatalf("dest: %+v", r)
	}
	if r.Port == nil || *r.Port != 18080 {
		t.Fatalf("port: %v", *r.Port)
	}
	if r.ID != safetyControlPlaneID {
		t.Fatalf("id: %s", r.ID)
	}
}

func TestControlPlaneSafetyRuleHostDefaultHTTPS(t *testing.T) {
	t.Parallel()
	r, ok := controlPlaneSafetyRule("https://cp.example.local/path")
	if !ok {
		t.Fatal("expected ok")
	}
	if r.DestType != "host" || r.DestValue != "cp.example.local" {
		t.Fatalf("dest: %+v", r)
	}
	if r.Port == nil || *r.Port != 443 {
		t.Fatalf("port want 443 got %d", *r.Port)
	}
}

func TestEnsureControlPlaneSafetyDedup(t *testing.T) {
	t.Parallel()
	spec := Spec{
		SafetyAllow: []Rule{{ID: "safety-loopback", DestType: "cidr", DestValue: "127.0.0.1/8"}},
	}
	once := EnsureControlPlaneSafety(spec, "http://10.0.0.1:18080")
	twice := EnsureControlPlaneSafety(once, "http://10.0.0.1:18080")
	if len(twice.SafetyAllow) != 2 {
		t.Fatalf("want dedup 2, got %d", len(twice.SafetyAllow))
	}
	if twice.SafetyAllow[0].ID != safetyControlPlaneID {
		t.Fatalf("controlplane should be first: %+v", twice.SafetyAllow[0])
	}
}