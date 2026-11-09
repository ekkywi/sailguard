package networkpolicy

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

func HashEffective(ep EffectivePolicy) string {
	clone := struct {
		SchemaVersion int             `json:"schema_version"`
		Version       int64           `json:"version"`
		Mode          string          `json:"mode"`
		SafetyAllow   []EffectiveRule `json:"safety_allow"`
		Rules         []EffectiveRule `json:"rules"`
	}{
		SchemaVersion: ep.SchemaVersion,
		Version:       ep.Version,
		Mode:          ep.Mode,
		SafetyAllow:   ep.SafetyAllow,
		Rules:         ep.Rules,
	}
	b, _ := json.Marshal(clone)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
