package networkpolicycache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ekkywi/sailguard/pkgs/agentcontract"
)

const fileName = "network_policy.json"

type Document struct {
	SchemaVersion int                                  `json:"schema_version"`
	Version       int64                                `json:"version"`
	ComputedAt    string                               `json:"computed_at,omitempty"`
	Mode          string                               `json:"mode"`
	Hash          string                               `json:"hash"`
	SafetyAllow   []agentcontract.NetworkEffectiveRule `json:"safety_allow"`
	Rules         []agentcontract.NetworkEffectiveRule `json:"rules"`
}

func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

func Load(dir string) (*Document, error) {
	b, err := os.ReadFile(Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read network policy file: %w", err)
	}
	var doc Document
	if err := json.Unmarshal(b, &doc); err != nil {
		return nil, fmt.Errorf("unmarshal network policy: %w", err)
	}
	return &doc, nil
}

func Save(dir string, doc Document) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	if doc.SafetyAllow == nil {
		doc.SafetyAllow = []agentcontract.NetworkEffectiveRule{}
	}
	if doc.Rules == nil {
		doc.Rules = []agentcontract.NetworkEffectiveRule{}
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	path := Path(dir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp network policy: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename network policy: %w", err)
	}
	return nil
}

func Hash(dir string) string {
	doc, err := Load(dir)
	if err != nil || doc == nil {
		return ""
	}
	return doc.Hash
}

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || os.IsNotExist(err)
}

func FromSync(data agentcontract.NetworkPolicySyncData) Document {
	return Document{
		SchemaVersion: data.SchemaVersion,
		Version:       data.Version,
		ComputedAt:    data.ComputedAt,
		Mode:          data.Mode,
		Hash:          data.Hash,
		SafetyAllow:   data.SafetyAllow,
		Rules:         data.Rules,
	}
}
