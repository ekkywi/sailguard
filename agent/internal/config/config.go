package config

import (
	"os"
	"time"
)

type Config struct {
	ServerURL         string
	DataDir           string
	AgentVersion      string
	EnrollmentToken   string
	ScanInterval      time.Duration
	HeartbeatInterval time.Duration
}

func Load() Config {
	return Config{
		ServerURL:         getEnv("SG_SERVER_URL", "http://127.0.0.1:18080"),
		DataDir:           getEnv("SG_DATA_DIR", ""),
		AgentVersion:      getEnv("SG_AGENT_VERSION", "0.0.0-dev"),
		EnrollmentToken:   getEnv("SG_ENROLLMENT_TOKEN", ""),
		ScanInterval:      getDurationEnv("SG_SCAN_INTERVAL", 3*time.Second),
		HeartbeatInterval: getDurationEnv("SG_HEARTBEAT_INTERVAL", 30*time.Second),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
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
