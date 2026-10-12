package policycache

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	policypkg "github.com/ekkywi/sailguard/pkgs/policy"
)

const fileName = "policy.json"

func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

func Load(dir string) (*policypkg.EffectivePolicy, error) {
	b, err := os.ReadFile(Path(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read policy file: %w", err)
	}
	var ep policypkg.EffectivePolicy
	if err := json.Unmarshal(b, &ep); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	return &ep, nil
}

func Save(dir string, ep policypkg.EffectivePolicy) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	b, err := json.MarshalIndent(ep, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	path := Path(dir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp policy: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename policy: %w", err)
	}
	return nil
}

func Hash(dir string) string {
	ep, err := Load(dir)
	if err != nil || ep == nil {
		return ""
	}
	return ep.Hash
}

func IsNotExist(err error) bool {
	return errors.Is(err, os.ErrNotExist) || os.IsNotExist(err)
}
