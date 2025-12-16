package config

import (
	"os"
	"strconv"
)

type Config struct {
	Host            string
	Port            int
	PolicyPath      string
	CacheTTLSeconds int
	AuditLogEnabled bool
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("PORT", "50054"))
	cacheTTL, _ := strconv.Atoi(getEnv("CACHE_TTL_SECONDS", "300"))

	return &Config{
		Host:            getEnv("HOST", "0.0.0.0"),
		Port:            port,
		PolicyPath:      getEnv("POLICY_PATH", "./policies"),
		CacheTTLSeconds: cacheTTL,
		AuditLogEnabled: getEnv("AUDIT_LOG_ENABLED", "true") == "true",
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
