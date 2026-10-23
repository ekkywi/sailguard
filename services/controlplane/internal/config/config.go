package config

import (
	"os"
	"time"
)

type Config struct {
	HTTPAddr         string
	DBURL            string
	RedisURL         string
	JWTSecret        string
	Env              string
	AgentWin7Path    string
	AgentWindowsPath string
	AgentSetupPath   string
	PublicAgentURL   string
	AdminTokenTTL    time.Duration
}

func Load() Config {
	return Config{
		HTTPAddr:         getEnv("SG_HTTP_ADDR", "127.0.0.1:18080"),
		DBURL:            getEnv("SG_DB_URL", "postgres://sailguard:sailguard@127.0.0.1:15433/sailguard?sslmode=disable"),
		RedisURL:         getEnv("SG_REDIS_URL", "redis://127.0.0.1:16380/0"),
		JWTSecret:        getEnv("SG_JWT_SECRET", "dev-only-change-me"),
		Env:              getEnv("SG_ENV", "development"),
		AgentWin7Path:    getEnv("SG_AGENT_WIN7_PATH", "bin/sailguard-agent-win7.exe"),
		AgentWindowsPath: getEnv("SG_AGENT_WINDOWS_PATH", "bin/sailguard-agent-windows.exe"),
		AgentSetupPath:   getEnv("SG_AGENT_SETUP_PATH", "bin/sailguard-agent-setup.exe"),
		PublicAgentURL:   getEnv("SG_PUBLIC_AGENT_URL", ""),
		AdminTokenTTL:    getEnvDuration("SG_ADMIN_TOKEN_TTL", 8*time.Hour),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
