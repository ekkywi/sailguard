package config

import "os"

type Config struct {
	HTTPAddr         string
	DBURL            string
	RedisURL         string
	JWTSecret        string
	Env              string
	AgentWin7Path    string // legacy Win7 SP1+ / 8.1 (Go 1.20 artefact)
	AgentWindowsPath string // current Win10/11+ (modern Go artefact)
	AgentSetupPath   string // Windows setup exe (Go 1.20; installs to Program Files)
	PublicAgentURL   string // optional hint shown in download UI (LAN API base)
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
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
