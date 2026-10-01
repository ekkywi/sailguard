package policy

import "testing"

func TestNormalizeProcessName(t *testing.T) {
	got := NormalizeValue(RuleTypeProcessName, `C:\Apps\Telegram.EXE`)
	if got != "telegram.exe" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizePath(t *testing.T) {
	got := NormalizeValue(RuleTypePath, `C:/Games/Foo`)
	if got != `c:\games\foo` {
		t.Fatalf("got %q", got)
	}
}

func TestMatchEqualsProcess(t *testing.T) {
	rule := EffectiveRule{
		Action:   ActionBlock,
		RuleType: RuleTypeProcessName,
		Operator: OperatorEquals,
		Value:    "telegram.exe",
	}
	if !Match(rule, ProcessAttrs{Name: "Telegram.exe"}) {
		t.Fatal("expected match")
	}
}

func TestMergeAllowBeatsBlockSameGroupWeight(t *testing.T) {
	ep := Merge(MergeInput{
		Version: 1,
		NowUnix: 1_700_000_000,
		Candidates: []Candidate{
			{
				ID: "finance-block", PolicyID: "p1", Action: ActionBlock,
				RuleType: RuleTypeProcessName, Operator: OperatorEquals, Value: "telegram.exe",
				Scope: ScopeGroup, ScopeWeight: WeightGroup, PolicyMode: ModeEnforce, Source: "policy",
			},
			{
				ID: "it-allow", PolicyID: "p2", Action: ActionAllow,
				RuleType: RuleTypeProcessName, Operator: OperatorEquals, Value: "telegram.exe",
				Scope: ScopeGroup, ScopeWeight: WeightGroup, PolicyMode: ModeEnforce, Source: "policy",
			},
		},
	})
	if len(ep.Rules) != 1 || ep.Rules[0].Action != ActionAllow {
		t.Fatalf("expected allow winner, got %+v", ep.Rules)
	}
	if ep.Mode != ModeAudit {
		t.Fatalf("allow-only winners should yield audit mode, got %s", ep.Mode)
	}
}

func TestMergeDeviceOverrideBeatsGroup(t *testing.T) {
	ep := Merge(MergeInput{
		Version: 2,
		NowUnix: 1_700_000_000,
		Candidates: []Candidate{
			{
				ID: "g-block", Action: ActionBlock, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "teamviewer.exe",
				Scope: ScopeGroup, ScopeWeight: WeightGroup, PolicyMode: ModeEnforce, Source: "policy",
			},
			{
				ID: "ov-allow", Action: ActionAllow, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "teamviewer.exe",
				Scope: ScopeOverride, ScopeWeight: WeightOverride, PolicyMode: ModeEnforce,
				Source: "override", ExpiresUnix: 1_800_000_000, Reason: "ticket-1",
			},
		},
	})
	if len(ep.Rules) != 1 || ep.Rules[0].Action != ActionAllow {
		t.Fatalf("expected override allow, got %+v", ep.Rules)
	}
}

func TestMergeExpiredOverrideIgnored(t *testing.T) {
	ep := Merge(MergeInput{
		Version: 3,
		NowUnix: 1_700_000_000,
		Candidates: []Candidate{
			{
				ID: "g-block", Action: ActionBlock, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "teamviewer.exe",
				Scope: ScopeGroup, ScopeWeight: WeightGroup, PolicyMode: ModeEnforce, Source: "policy",
			},
			{
				ID: "ov-allow", Action: ActionAllow, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "teamviewer.exe",
				Scope: ScopeOverride, ScopeWeight: WeightOverride,
				Source: "override", ExpiresUnix: 1_600_000_000,
			},
		},
	})
	if len(ep.Rules) != 1 || ep.Rules[0].Action != ActionBlock {
		t.Fatalf("expected block after expired override, got %+v", ep.Rules)
	}
	if ep.Mode != ModeEnforce {
		t.Fatalf("expected enforce mode, got %s", ep.Mode)
	}
}

func TestMergeOrgBlockDeviceAllow(t *testing.T) {
	ep := Merge(MergeInput{
		Version: 4,
		NowUnix: 1_700_000_000,
		Candidates: []Candidate{
			{
				ID: "org", Action: ActionBlock, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "utorrent.exe",
				Scope: ScopeOrg, PolicyMode: ModeEnforce, Source: "policy",
			},
			{
				ID: "dev", Action: ActionAllow, RuleType: RuleTypeProcessName,
				Operator: OperatorEquals, Value: "utorrent.exe",
				Scope: ScopeDevice, PolicyMode: ModeEnforce, Source: "policy",
			},
		},
	})
	if ep.Rules[0].Action != ActionAllow {
		t.Fatalf("device should win: %+v", ep.Rules)
	}
}

func TestEvaluateAllowFirst(t *testing.T) {
	ep := EffectivePolicy{
		Mode: ModeEnforce,
		Rules: []EffectiveRule{
			{Action: ActionBlock, RuleType: RuleTypeProcessName, Operator: OperatorEquals, Value: "telegram.exe"},
			{Action: ActionAllow, RuleType: RuleTypeProcessName, Operator: OperatorEquals, Value: "telegram.exe"},
		},
	}
	r, allow := Evaluate(ep, ProcessAttrs{Name: "telegram.exe"})
	if r == nil || !allow {
		t.Fatal("expected allow")
	}
}

func TestIsCriticalProcess(t *testing.T) {
	if !IsCriticalProcess("lsass.exe") {
		t.Fatal("lsass should be critical")
	}
	if IsCriticalProcess("telegram.exe") {
		t.Fatal("telegram should not be critical")
	}
}
