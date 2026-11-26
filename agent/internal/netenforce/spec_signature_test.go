package netenforce

import "testing"

func TestSpecSignatureIgnoresOrder(t *testing.T) {
	a := Spec{Rules: []Rule{
		{Action: "deny", DestType: "ip", DestValue: "1.1.1.1"},
		{Action: "deny", DestType: "cidr", DestValue: "10.0.0.0/8"},
	}}
	b := Spec{Rules: []Rule{a.Rules[1], a.Rules[0]}}
	if specSignature(a) != specSignature(b) {
		t.Fatal("same rules in different order should have same signature")
	}
}

func TestSpecSignatureChangesWithDest(t *testing.T) {
	a := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.1"}}}
	b := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.2"}}}
	if specSignature(a) == specSignature(b) {
		t.Fatal("different IP should change signature")
	}
}

func TestSpecSignatureChangesWithPort(t *testing.T) {
	p := 443
	a := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.1"}}}
	b := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.1", Port: &p}}}
	if specSignature(a) == specSignature(b) {
		t.Fatal("different port should change signature")
	}
}

func TestSpecSignatureIgnoresLabel(t *testing.T) {
	a := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.1", Label: "x"}}}
	b := Spec{Rules: []Rule{{Action: "deny", DestType: "ip", DestValue: "203.0.113.1", Label: "y"}}}
	if specSignature(a) != specSignature(b) {
		t.Fatal("label change should not change signature")
	}
}

func TestSpecSignatureSeparatesSafetyFromRules(t *testing.T) {
	r := Rule{Action: "deny", DestType: "ip", DestValue: "203.0.113.1"}
	a := Spec{SafetyAllow: []Rule{r}}
	b := Spec{Rules: []Rule{r}}
	if specSignature(a) == specSignature(b) {
		t.Fatal("same rule in safety vs rules should differ")
	}
}