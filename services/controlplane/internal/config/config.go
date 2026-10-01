package config

import "os"

type Config struct {
	HTTPAddr  string
	DBURL     string
	RedisURL  string
	JWTSecret string
	Env       string
}

func Load() Config {
	return Config{
		HTTPAddr:  getEnv("SG_HTTP_ADDR", "127.0.0.1:18080"),
		DBURL:     getEnv("SG_DB_URL", "postgres://sailguard:sailguard@127.0.0.1:15433/sailguard?sslmode=disable"),
		RedisURL:  getEnv("SG_REDIS_URL", "redis://127.0.0.1:16380/0"),
		JWTSecret: getEnv("SG_JWT_SECRET", "dev-only-change-me"),
		Env:       getEnv("SG_ENV", "development"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
