package policy

import (
	"path/filepath"
	"strings"
)

// NormalizeValue prepares rule values / process attrs for stable match keys.
func NormalizeValue(ruleType RuleType, value string) string {
	v := strings.TrimSpace(value)
	switch ruleType {
	case RuleTypeProcessName:
		v = strings.ToLower(v)
		v = filepath.Base(strings.ReplaceAll(v, "\\", "/"))
		return v
	case RuleTypePath:
		v = strings.ToLower(v)
		v = strings.ReplaceAll(v, "/", "\\")
		v = strings.TrimPrefix(v, `\\?\`)
		return v
	case RuleTypeHashSHA256:
		return strings.ToLower(strings.TrimSpace(v))
	case RuleTypePublisher, RuleTypeProductName:
		return strings.Join(strings.Fields(v), " ")
	default:
		return v
	}
}

// MatchKey builds the conflict key used by Merge.
func MatchKey(ruleType RuleType, operator Operator, value string) string {
	return string(ruleType) + "|" + string(operator) + "|" + NormalizeValue(ruleType, value)
}

// AttrFor returns the process attribute corresponding to a rule type.
func AttrFor(ruleType RuleType, p ProcessAttrs) string {
	switch ruleType {
	case RuleTypeProcessName:
		return p.Name
	case RuleTypePath:
		return p.Path
	case RuleTypeHashSHA256:
		return p.HashSHA256
	case RuleTypePublisher:
		return p.Publisher
	case RuleTypeProductName:
		return p.ProductName
	default:
		return ""
	}
}
