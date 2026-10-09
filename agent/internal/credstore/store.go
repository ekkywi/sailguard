package credstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const fileName = "device.json"

type Credential struct {
	DeviceID    string `json:"device_id"`
	DeviceToken string `json:"device_token"`
	ServerURL   string `json:"server_url,omitempty"`
}

func DefaultDir() string {
	switch runtime.GOOS {
	case "windows":
		base := os.Getenv("PROGRAMDATA")
		if base == "" {
			base = `C:\ProgramData`
		}
		return filepath.Join(base, "SailGuard", "agent")
	default:
		if home, err := os.UserHomeDir(); err == nil && home != "" {
			return filepath.Join(home, ".sailguard", "agent")
		}
		return filepath.Join(".", ".sailguard", "agent")
	}
}

func ResolveDir(configured string) string {
	if configured != "" {
		return configured
	}
	return DefaultDir()
}

func Path(dir string) string {
	return filepath.Join(dir, fileName)
}

func Load(dir string) (*Credential, error) {
	path := Path(dir)
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err
		}
		return nil, fmt.Errorf("read credential: %w", err)
	}

	var c Credential
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse credential: %w", err)
	}
	if c.DeviceID == "" || c.DeviceToken == "" {
		return nil, fmt.Errorf("credential file incomplete %s", path)
	}
	return &c, nil
}

func Save(dir string, c Credential) error {
	if c.DeviceID == "" || c.DeviceToken == "" {
		return errors.New("device_id and device_token are required")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir data dir: %w", err)
	}

	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')

	path := Path(dir)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return fmt.Errorf("write temp credential: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename credential: %w", err)
	}
	return nil
}

func Exists(dir string) bool {
	_, err := os.Stat(Path(dir))
	return err == nil
}