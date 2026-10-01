package policy

import (
	"strings"
	"time"
)

// Match reports whether process attributes match a single effective rule.
// Case handling follows NormalizeValue for both sides.
func Match(rule EffectiveRule, p ProcessAttrs) bool {
	if rule.RuleType == RuleTypeCategory {
		return false // categories must be expanded server-side
	}
	if exp, ok := rule.ValueMeta["expires_at"].(string); ok && exp != "" {
		if t, err := time.Parse(time.RFC3339, exp); err == nil && time.Now().UTC().After(t) {
			return false
		}
	}

	left := NormalizeValue(rule.RuleType, AttrFor(rule.RuleType, p))
	right := NormalizeValue(rule.RuleType, rule.Value)
	if left == "" || right == "" {
		return false
	}

	switch rule.Operator {
	case OperatorEquals:
		if rule.RuleType == RuleTypePublisher || rule.RuleType == RuleTypeProductName {
			return strings.EqualFold(left, right)
		}
		return left == right
	case OperatorPrefix:
		return strings.HasPrefix(left, right)
	case OperatorContains:
		return strings.Contains(left, right)
	default:
		return false
	}
}

// Evaluate returns the first matching allow rule, else the first matching block rule.
// Allows are checked first (aligned with merge semantics).
func Evaluate(policy EffectivePolicy, p ProcessAttrs) (matched *EffectiveRule, allow bool) {
	for i := range policy.Rules {
		r := &policy.Rules[i]
		if r.Action == ActionAllow && Match(*r, p) {
			return r, true
		}
	}
	for i := range policy.Rules {
		r := &policy.Rules[i]
		if r.Action == ActionBlock && Match(*r, p) {
			return r, false
		}
	}
	return nil, false
}
