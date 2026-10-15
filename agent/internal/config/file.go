package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// FileConfig is the optional config.json shipped next to the agent binary
// (download package) or pointed at by SG_CONFIG_FILE.
type FileConfig struct {
	ServerURL       string `json:"server_url"`
	EnrollmentToken string `json:"enrollment_token"`
	DataDir         string `json:"data_dir"`
	AgentVersion    string `json:"agent_version"`
}

func loadFileConfig() (FileConfig, string, error) {
	candidates := make([]string, 0, 4)
	if v := os.Getenv("SG_CONFIG_FILE"); v != "" {
		candidates = append(candidates, v)
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "config.json"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "config.json"))
	}

	var lastErr error
	for _, path := range candidates {
		b, err := os.ReadFile(path)
		if err != nil {
			lastErr = err
			continue
		}
		var fc FileConfig
		if err := json.Unmarshal(b, &fc); err != nil {
			return FileConfig{}, path, fmt.Errorf("parse %s: %w", path, err)
		}
		return fc, path, nil
	}
	if lastErr == nil {
		lastErr = os.ErrNotExist
	}
	return FileConfig{}, "", lastErr
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
