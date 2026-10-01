package policy

import "strings"

// CriticalProcessNames must never be terminated by the agent, even if policy says block.
// Extensible list for Windows-first MVP; other OS names can be added later.
var CriticalProcessNames = []string{
	"system",
	"smss.exe",
	"csrss.exe",
	"wininit.exe",
	"winlogon.exe",
	"services.exe",
	"lsass.exe",
	"svchost.exe",
	"fontdrvhost.exe",
	"dwm.exe",
	"explorer.exe", // controversial but prevents mass desktop breakage; revisit later
	"sailguard-agent.exe",
}

// IsCriticalProcess reports whether terminating this process is forbidden by safety rails.
func IsCriticalProcess(name string) bool {
	n := NormalizeValue(RuleTypeProcessName, name)
	for _, c := range CriticalProcessNames {
		if n == strings.ToLower(c) {
			return true
		}
	}
	return false
}
