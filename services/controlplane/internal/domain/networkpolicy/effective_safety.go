package networkpolicy

func defaultSafetyAllow() []EffectiveRule {
	return []EffectiveRule{
		{
			ID:        "safety-loopback",
			DestType:  "cidr",
			DestValue: "127.0.0.0/8",
			Label:     "loopback",
			Source:    "safety",
		},
		{
			ID:        "safety-api",
			DestType:  "ip",
			DestValue: "127.0.0.1",
			Port:      intPtr(18080),
			Label:     "sailguard-api",
			Source:    "safety",
		},
	}
}

func intPtr(v int) *int { return &v }
