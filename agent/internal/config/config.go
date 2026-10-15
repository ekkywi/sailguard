package config

import (
	"log"
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
