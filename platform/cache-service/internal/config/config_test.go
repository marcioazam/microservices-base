package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	clearEnv()
	defer clearEnv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.GRPCPort != 50051 {
		t.Errorf("GRPCPort = %d, want 50051", cfg.Server.GRPCPort)
	}
	if cfg.Server.HTTPPort != 8080 {
		t.Errorf("HTTPPort = %d, want 8080", cfg.Server.HTTPPort)
	}
	if cfg.Server.GracefulTimeout != 30*time.Second {
		t.Errorf("GracefulTimeout = %v, want 30s", cfg.Server.GracefulTimeout)
	}
	if len(cfg.Redis.Addresses) != 1 || cfg.Redis.Addresses[0] != "localhost:6379" {
		t.Errorf("Redis.Addresses = %v, want [localhost:6379]", cfg.Redis.Addresses)
	}
	if cfg.Redis.PoolSize != 10 {
		t.Errorf("Redis.PoolSize = %d, want 10", cfg.Redis.PoolSize)
	}
	if cfg.Cache.DefaultTTL != time.Hour {
		t.Errorf("Cache.DefaultTTL = %v, want 1h", cfg.Cache.DefaultTTL)
	}
	if cfg.Cache.EvictionPolicy != "lru" {
		t.Errorf("Cache.EvictionPolicy = %s, want lru", cfg.Cache.EvictionPolicy)
	}
	if !cfg.Cache.LocalCacheEnabled {
		t.Error("Cache.LocalCacheEnabled = false, want true")
	}
	if !cfg.Metrics.Enabled {
		t.Error("Metrics.Enabled = false, want true")
	}
}

func TestLoad_CustomValues(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("SERVER_GRPC_PORT", "9090")
	os.Setenv("SERVER_HTTP_PORT", "8081")
	os.Setenv("SERVER_GRACEFUL_TIMEOUT", "60s")
	os.Setenv("REDIS_ADDRESSES", "redis1:6379,redis2:6379")
	os.Setenv("REDIS_PASSWORD", "secret")
	os.Setenv("REDIS_POOL_SIZE", "20")
	os.Setenv("REDIS_CLUSTER_MODE", "true")
	os.Setenv("CACHE_DEFAULT_TTL", "2h")
	os.Setenv("CACHE_EVICTION_POLICY", "lfu")
	os.Setenv("CACHE_LOCAL_CACHE_SIZE", "5000")
	os.Setenv("BROKER_TYPE", "kafka")
	os.Setenv("BROKER_URL", "kafka:9092")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.GRPCPort != 9090 {
		t.Errorf("GRPCPort = %d, want 9090", cfg.Server.GRPCPort)
	}
	if cfg.Server.HTTPPort != 8081 {
		t.Errorf("HTTPPort = %d, want 8081", cfg.Server.HTTPPort)
	}
	if cfg.Server.GracefulTimeout != 60*time.Second {
		t.Errorf("GracefulTimeout = %v, want 60s", cfg.Server.GracefulTimeout)
	}
	if len(cfg.Redis.Addresses) != 2 {
		t.Errorf("Redis.Addresses len = %d, want 2", len(cfg.Redis.Addresses))
	}
	if cfg.Redis.Password != "secret" {
		t.Errorf("Redis.Password = %s, want secret", cfg.Redis.Password)
	}
	if cfg.Redis.PoolSize != 20 {
		t.Errorf("Redis.PoolSize = %d, want 20", cfg.Redis.PoolSize)
	}
	if !cfg.Redis.ClusterMode {
		t.Error("Redis.ClusterMode = false, want true")
	}
	if cfg.Cache.DefaultTTL != 2*time.Hour {
		t.Errorf("Cache.DefaultTTL = %v, want 2h", cfg.Cache.DefaultTTL)
	}
	if cfg.Cache.EvictionPolicy != "lfu" {
		t.Errorf("Cache.EvictionPolicy = %s, want lfu", cfg.Cache.EvictionPolicy)
	}
	if cfg.Broker.Type != "kafka" {
		t.Errorf("Broker.Type = %s, want kafka", cfg.Broker.Type)
	}
}

func TestValidate_InvalidGRPCPort(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("SERVER_GRPC_PORT", "0")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid GRPC port")
	}
}

func TestValidate_InvalidHTTPPort(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("SERVER_HTTP_PORT", "70000")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid HTTP port")
	}
}

func TestValidate_InvalidPoolSize(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("REDIS_POOL_SIZE", "0")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid pool size")
	}
}

func TestValidate_InvalidEvictionPolicy(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("CACHE_EVICTION_POLICY", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid eviction policy")
	}
}

func TestValidate_InvalidLocalCacheSize(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("CACHE_LOCAL_CACHE_ENABLED", "true")
	os.Setenv("CACHE_LOCAL_CACHE_SIZE", "0")

	_, err := Load()
	if err == nil {
		t.Error("Load() expected error for invalid local cache size")
	}
}

func TestLogSafe_MasksSecrets(t *testing.T) {
	clearEnv()
	defer clearEnv()

	os.Setenv("REDIS_PASSWORD", "supersecret")
	os.Setenv("AUTH_JWT_SECRET", "jwtsecret")
	os.Setenv("AUTH_ENCRYPTION_KEY", "enckey")
	os.Setenv("BROKER_URL", "amqp://user:pass@host")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	safe := cfg.LogSafe()

	redis := safe["redis"].(map[string]interface{})
	if redis["password"] == "supersecret" {
		t.Error("LogSafe() did not mask redis password")
	}

	auth := safe["auth"].(map[string]interface{})
	if auth["jwt_secret"] == "jwtsecret" {
		t.Error("LogSafe() did not mask jwt_secret")
	}
	if auth["encryption_key"] == "enckey" {
		t.Error("LogSafe() did not mask encryption_key")
	}

	broker := safe["broker"].(map[string]interface{})
	if broker["url"] == "amqp://user:pass@host" {
		t.Error("LogSafe() did not mask broker url")
	}
}

func clearEnv() {
	envVars := []string{
		"SERVER_GRPC_PORT", "SERVER_HTTP_PORT", "SERVER_GRACEFUL_TIMEOUT",
		"REDIS_ADDRESSES", "REDIS_PASSWORD", "REDIS_DB", "REDIS_POOL_SIZE",
		"REDIS_CLUSTER_MODE", "REDIS_TLS_ENABLED", "REDIS_DIAL_TIMEOUT",
		"REDIS_READ_TIMEOUT", "REDIS_WRITE_TIMEOUT",
		"BROKER_TYPE", "BROKER_URL", "BROKER_TOPIC", "BROKER_GROUP_ID",
		"AUTH_JWT_SECRET", "AUTH_JWT_ISSUER", "AUTH_ENCRYPTION_KEY",
		"CACHE_DEFAULT_TTL", "CACHE_MAX_MEMORY_MB", "CACHE_EVICTION_POLICY",
		"CACHE_LOCAL_CACHE_ENABLED", "CACHE_LOCAL_CACHE_SIZE",
		"METRICS_ENABLED", "METRICS_PATH",
	}
	for _, v := range envVars {
		os.Unsetenv(v)
	}
}
