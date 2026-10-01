package config

import "os"

type Config struct {
	ServerURL    string
	DataDir      string
	AgentVersion string
}

func Load() Config {
	return Config{
		ServerURL:    getEnv("SG_SERVER_URL", "http://127.0.0.1:18080"),
		DataDir:      getEnv("SG_DATA_DIR", ""), // empty → platform default
		AgentVersion: getEnv("SG_AGENT_VERSION", "0.0.0-dev"),
	}
}

func getEnv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
