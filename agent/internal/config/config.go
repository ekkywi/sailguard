package config

import (
	"log"
	"os"
	"strings"
	"time"
)

type Config struct {
	ServerURL         string
	DataDir           string
	AgentVersion      string
	EnrollmentToken   string
	ScanInterval      time.Duration
	HeartbeatInterval time.Duration
	// NetworkEnforce installs OS filters (WFP on Windows). Default off:
	// set SG_NETWORK_ENFORCE=1 or config network_enforce=true when ready to lab-test.
	NetworkEnforce bool
}

func Load() Config {
	file, path, err := loadFileConfig()
	if err == nil && path != "" {
		log.Printf("loaded agent config file %s", path)
	}

	return Config{
		ServerURL: firstNonEmpty(
			os.Getenv("SG_SERVER_URL"),
			file.ServerURL,
			"http://127.0.0.1:18080",
		),
		DataDir: firstNonEmpty(
			os.Getenv("SG_DATA_DIR"),
			file.DataDir,
		),
		AgentVersion: firstNonEmpty(
			os.Getenv("SG_AGENT_VERSION"),
			file.AgentVersion,
			"0.0.0-dev",
		),
		EnrollmentToken: firstNonEmpty(
			os.Getenv("SG_ENROLLMENT_TOKEN"),
			file.EnrollmentToken,
		),
		ScanInterval:      getDurationEnv("SG_SCAN_INTERVAL", 3*time.Second),
		HeartbeatInterval: getDurationEnv("SG_HEARTBEAT_INTERVAL", 30*time.Second),
		NetworkEnforce:    loadNetworkEnforce(file.NetworkEnforce),
	}
}

// loadNetworkEnforce: env wins. Default false (filters off).
func loadNetworkEnforce(fileVal *bool) bool {
	if v, ok := os.LookupEnv("SG_NETWORK_ENFORCE"); ok {
		return parseBoolish(v)
	}
	if fileVal != nil {
		return *fileVal
	}
	return false
}

func parseBoolish(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func getDurationEnv(k string, fallback time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
