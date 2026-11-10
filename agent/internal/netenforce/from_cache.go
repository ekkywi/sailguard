package netenforce

import (
	"github.com/ekkywi/sailguard/agent/internal/networkpolicycache"
	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

func SpecFromDocument(doc networkpolicycache.Document) Spec {
	return Spec{
		Mode:        doc.Mode,
		Hash:        doc.Hash,
		Version:     doc.Version,
		SafetyAllow: mapRules(doc.SafetyAllow),
		Rules:       mapRules(doc.Rules),
	}
}

func mapRules(in []agentcontract.NetworkEffectiveRule) []Rule {
	if len(in) == 0 {
		return []Rule{}
	}
	out := make([]Rule, 0, len(in))
	for _, r := range in {
		out = append(out, Rule{
			ID:        r.ID,
			Action:    r.Action,
			DestType:  r.DestType,
			DestValue: r.DestValue,
			Port:      r.Port,
			Label:     r.Label,
			Source:    r.Source,
		})
	}
	return out
}
