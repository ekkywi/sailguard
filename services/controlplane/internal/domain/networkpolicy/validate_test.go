package networkpolicy

import "testing"

func TestNormalizeRuleFieldsHost(t *testing.T) {
	t.Parallel()

	action, destType, destValue, _, _, err := normalizeRuleFields(
		"deny", "host", "Activation.Windows.com", "win-activation", nil,
	)
	if err != nil {
		t.Fatalf("expected host ok, got %v", err)
	}
	if action != "deny" || destType != "host" || destValue != "activation.windows.com" {
		t.Fatalf("got action=%q type=%q value=%q", action, destType, destValue)
	}

	_, _, _, _, _, err = normalizeRuleFields("deny", "host", "1.1.1.1", "bad", nil)
	if err == nil {
		t.Fatal("expected host+IP to be rejected")
	}

	_, _, _, _, _, err = normalizeRuleFields("deny", "ip", "1.1.1.1", "ip-ok", nil)
	if err != nil {
		t.Fatalf("expected ip ok, got %v", err)
	}

	_, _, _, _, _, err = normalizeRuleFields("deny", "host", "bad host.com", "x", nil)
	if err == nil {
		t.Fatal("expected space in host to be rejected")
	}
}
