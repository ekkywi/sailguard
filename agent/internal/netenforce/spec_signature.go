package netenforce

import (
	"fmt"
	"sort"
	"strings"
)

func specSignature(spec Spec) string {
	safety := strings.Join(ruleKeys(spec.SafetyAllow), "\n")
	rules := strings.Join(ruleKeys(spec.Rules), "\n")
	return safety + "\n---\n" + rules
}

func ruleKeys(rules []Rule) []string {
	keys := make([]string, 0, len(rules))
	for _, r := range rules {
		port := "-"
		if r.Port != nil {
			port = fmt.Sprintf("%d", *r.Port)
		}
		keys = append(keys, fmt.Sprintf("%s|%s|%s|%s", r.Action, r.DestType, r.DestValue, port))
	}
	sort.Strings(keys)
	return keys
}