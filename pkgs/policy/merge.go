package policy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

// MergeInput is the set of expanded candidates for one device.
type MergeInput struct {
	Candidates []Candidate
	NowUnix    int64 // 0 = use time.Now
	Version    int64
}

// Merge resolves conflicts per match_key and returns a flattened effective policy.
// Spec summary (see docs/policy-merge.md):
//  1. Higher scope_weight wins
//  2. Same weight: allow beats block
//  3. Then policy priority, UpdatedUnix, ID
//  4. Mode = enforce if any winning block came from an enforce policy
func Merge(in MergeInput) EffectivePolicy {
	now := in.NowUnix
	if now == 0 {
		now = time.Now().Unix()
	}

	byKey := map[string][]Candidate{}
	for _, c := range in.Candidates {
		if c.ExpiresUnix > 0 && c.ExpiresUnix < now {
			continue
		}
		if c.RuleType == RuleTypeCategory {
			continue // must be expanded before merge
		}
		c.Value = NormalizeValue(c.RuleType, c.Value)
		if c.ScopeWeight == 0 {
			c.ScopeWeight = weightForScope(c.Scope)
		}
		key := MatchKey(c.RuleType, c.Operator, c.Value)
		byKey[key] = append(byKey[key], c)
	}

	winners := make([]Candidate, 0, len(byKey))
	for _, group := range byKey {
		winners = append(winners, pickWinner(group))
	}

	sort.Slice(winners, func(i, j int) bool {
		a, b := winners[i], winners[j]
		if a.RuleType != b.RuleType {
			return a.RuleType < b.RuleType
		}
		if a.Value != b.Value {
			return a.Value < b.Value
		}
		return a.Action < b.Action
	})

	mode := ModeAudit
	rules := make([]EffectiveRule, 0, len(winners))
	for _, w := range winners {
		if w.Action == ActionBlock && w.PolicyMode == ModeEnforce {
			mode = ModeEnforce
		}
		meta := map[string]any{
			"source":    w.Source,
			"policy_id": w.PolicyID,
			"scope":     string(w.Scope),
		}
		if w.CategoryCode != "" {
			meta["category_code"] = w.CategoryCode
		}
		if w.Reason != "" {
			meta["reason"] = w.Reason
		}
		if w.ExpiresUnix > 0 {
			meta["expires_at"] = time.Unix(w.ExpiresUnix, 0).UTC().Format(time.RFC3339)
		}
		rules = append(rules, EffectiveRule{
			ID:           w.ID,
			Action:       w.Action,
			RuleType:     w.RuleType,
			Operator:     w.Operator,
			Value:        w.Value,
			ValueMeta:    meta,
			CategoryCode: w.CategoryCode,
		})
	}

	ep := EffectivePolicy{
		SchemaVersion: SchemaVersion,
		Version:       in.Version,
		ComputedAt:    time.Unix(now, 0).UTC().Format(time.RFC3339),
		Mode:          mode,
		Rules:         rules,
	}
	ep.Hash = HashEffective(ep)
	return ep
}

func weightForScope(s Scope) int {
	switch s {
	case ScopeOrg:
		return WeightOrg
	case ScopeGroup:
		return WeightGroup
	case ScopeDevice:
		return WeightDevice
	case ScopeOverride:
		return WeightOverride
	default:
		return WeightOrg
	}
}

func pickWinner(group []Candidate) Candidate {
	best := group[0]
	for _, c := range group[1:] {
		if better(c, best) {
			best = c
		}
	}
	return best
}

func better(a, b Candidate) bool {
	if a.ScopeWeight != b.ScopeWeight {
		return a.ScopeWeight > b.ScopeWeight
	}
	if a.Action != b.Action {
		// allow beats block at same weight
		return a.Action == ActionAllow && b.Action == ActionBlock
	}
	if a.PolicyPrio != b.PolicyPrio {
		return a.PolicyPrio > b.PolicyPrio
	}
	if a.UpdatedUnix != b.UpdatedUnix {
		return a.UpdatedUnix > b.UpdatedUnix
	}
	return a.ID > b.ID
}

// HashEffective computes a stable content hash (excludes computed_at).
func HashEffective(ep EffectivePolicy) string {
	clone := struct {
		SchemaVersion int             `json:"schema_version"`
		Version       int64           `json:"version"`
		Mode          Mode            `json:"mode"`
		Rules         []EffectiveRule `json:"rules"`
	}{
		SchemaVersion: ep.SchemaVersion,
		Version:       ep.Version,
		Mode:          ep.Mode,
		Rules:         ep.Rules,
	}
	b, _ := json.Marshal(clone)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
