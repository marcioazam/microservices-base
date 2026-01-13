// Package config provides configuration loading and validation.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all service configuration.
type Config struct {
	Server  ServerConfig
	Redis   RedisConfig
	Broker  BrokerConfig
	Auth    AuthConfig
	Cache   CacheConfig
	Metrics MetricsConfig
	Logging LoggingConfig
	Tracing TracingConfig
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	GRPCPort        int
	HTTPPort        int
	GracefulTimeout time.Duration
}

// RedisConfig holds Redis-related configuration.
type RedisConfig struct {
	Addresses    []string
	Password     string
	DB           int
	PoolSize     int
	ClusterMode  bool
	TLSEnabled   bool
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// BrokerConfig holds message broker configuration.
type BrokerConfig struct {
	Type    string // "rabbitmq" or "kafka"
	URL     string
	Topic   string
	GroupID string
}

// AuthConfig holds authentication configuration.
type AuthConfig struct {
	JWTSecret     string
	JWTIssuer     string
	EncryptionKey string
}

// CacheConfig holds cache behavior configuration.
type CacheConfig struct {
	DefaultTTL        time.Duration
	MaxMemoryMB       int
	EvictionPolicy    string
	LocalCacheEnabled bool
	LocalCacheSize    int
}

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// LoggingConfig holds logging-service client configuration.
type LoggingConfig struct {
	ServiceAddress string
	BatchSize      int
	FlushInterval  time.Duration
	BufferSize     int
	Enabled        bool
}

// TracingConfig holds tracing configuration.
type TracingConfig struct {
	Enabled  bool
	Endpoint string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			GRPCPort:        getEnvInt("SERVER_GRPC_PORT", 50051),
			HTTPPort:        getEnvInt("SERVER_HTTP_PORT", 8080),
			GracefulTimeout: getEnvDuration("SERVER_GRACEFUL_TIMEOUT", 30*time.Second),
		},
		Redis: RedisConfig{
			Addresses:    getEnvStringSlice("REDIS_ADDRESSES", []string{"localhost:6379"}),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getEnvInt("REDIS_DB", 0),
			PoolSize:     getEnvInt("REDIS_POOL_SIZE", 10),
			ClusterMode:  getEnvBool("REDIS_CLUSTER_MODE", false),
			TLSEnabled:   getEnvBool("REDIS_TLS_ENABLED", false),
			DialTimeout:  getEnvDuration("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getEnvDuration("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getEnvDuration("REDIS_WRITE_TIMEOUT", 3*time.Second),
		},
		Broker: BrokerConfig{
			Type:    getEnv("BROKER_TYPE", "rabbitmq"),
			URL:     getEnv("BROKER_URL", ""),
			Topic:   getEnv("BROKER_TOPIC", "cache-invalidation"),
			GroupID: getEnv("BROKER_GROUP_ID", "cache-service"),
		},
		Auth: AuthConfig{
			JWTSecret:     getEnv("AUTH_JWT_SECRET", ""),
			JWTIssuer:     getEnv("AUTH_JWT_ISSUER", "cache-service"),
			EncryptionKey: getEnv("AUTH_ENCRYPTION_KEY", ""),
		},
		Cache: CacheConfig{
			DefaultTTL:        getEnvDuration("CACHE_DEFAULT_TTL", time.Hour),
			MaxMemoryMB:       getEnvInt("CACHE_MAX_MEMORY_MB", 512),
			EvictionPolicy:    getEnv("CACHE_EVICTION_POLICY", "lru"),
			LocalCacheEnabled: getEnvBool("CACHE_LOCAL_CACHE_ENABLED", true),
			LocalCacheSize:    getEnvInt("CACHE_LOCAL_CACHE_SIZE", 10000),
		},
		Metrics: MetricsConfig{
			Enabled: getEnvBool("METRICS_ENABLED", true),
			Path:    getEnv("METRICS_PATH", "/metrics"),
		},
		Logging: LoggingConfig{
			ServiceAddress: getEnv("LOGGING_SERVICE_ADDRESS", "localhost:50052"),
			BatchSize:      getEnvInt("LOGGING_BATCH_SIZE", 100),
			FlushInterval:  getEnvDuration("LOGGING_FLUSH_INTERVAL", 5*time.Second),
			BufferSize:     getEnvInt("LOGGING_BUFFER_SIZE", 10000),
			Enabled:        getEnvBool("LOGGING_ENABLED", true),
		},
		Tracing: TracingConfig{
			Enabled:  getEnvBool("TRACING_ENABLED", false),
			Endpoint: getEnv("TRACING_ENDPOINT", ""),
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	var errs []string

	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		errs = append(errs, "SERVER_GRPC_PORT must be between 1 and 65535")
	}

	if c.Server.HTTPPort <= 0 || c.Server.HTTPPort > 65535 {
		errs = append(errs, "SERVER_HTTP_PORT must be between 1 and 65535")
	}

	if len(c.Redis.Addresses) == 0 {
		errs = append(errs, "REDIS_ADDRESSES is required")
	}

	if c.Redis.PoolSize <= 0 {
		errs = append(errs, "REDIS_POOL_SIZE must be positive")
	}

	if c.Cache.DefaultTTL <= 0 {
		errs = append(errs, "CACHE_DEFAULT_TTL must be positive")
	}

	if c.Cache.MaxMemoryMB <= 0 {
		errs = append(errs, "CACHE_MAX_MEMORY_MB must be positive")
	}

	policy := strings.ToLower(c.Cache.EvictionPolicy)
	if policy != "lru" && policy != "lfu" {
		errs = append(errs, "CACHE_EVICTION_POLICY must be 'lru' or 'lfu'")
	}

	if c.Cache.LocalCacheEnabled && c.Cache.LocalCacheSize <= 0 {
		errs = append(errs, "CACHE_LOCAL_CACHE_SIZE must be positive when local cache is enabled")
	}

	if c.Logging.Enabled && c.Logging.ServiceAddress == "" {
		errs = append(errs, "LOGGING_SERVICE_ADDRESS is required when logging is enabled")
	}

	if c.Logging.BatchSize <= 0 {
		errs = append(errs, "LOGGING_BATCH_SIZE must be positive")
	}

	if c.Logging.BufferSize <= 0 {
		errs = append(errs, "LOGGING_BUFFER_SIZE must be positive")
	}

	if len(errs) > 0 {
		return errors.New("configuration validation failed: " + strings.Join(errs, "; "))
	}

	return nil
}

// LogSafe returns a copy of config with sensitive values masked.
func (c *Config) LogSafe() map[string]interface{} {
	return map[string]interface{}{
		"server": map[string]interface{}{
			"grpc_port":        c.Server.GRPCPort,
			"http_port":        c.Server.HTTPPort,
			"graceful_timeout": c.Server.GracefulTimeout.String(),
		},
		"redis": map[string]interface{}{
			"addresses":     c.Redis.Addresses,
			"db":            c.Redis.DB,
			"pool_size":     c.Redis.PoolSize,
			"cluster_mode":  c.Redis.ClusterMode,
			"tls_enabled":   c.Redis.TLSEnabled,
			"dial_timeout":  c.Redis.DialTimeout.String(),
			"read_timeout":  c.Redis.ReadTimeout.String(),
			"write_timeout": c.Redis.WriteTimeout.String(),
			"password":      maskSecret(c.Redis.Password),
		},
		"broker": map[string]interface{}{
			"type":     c.Broker.Type,
			"url":      maskURL(c.Broker.URL),
			"topic":    c.Broker.Topic,
			"group_id": c.Broker.GroupID,
		},
		"auth": map[string]interface{}{
			"jwt_secret":     maskSecret(c.Auth.JWTSecret),
			"jwt_issuer":     c.Auth.JWTIssuer,
			"encryption_key": maskSecret(c.Auth.EncryptionKey),
		},
		"cache": map[string]interface{}{
			"default_ttl":         c.Cache.DefaultTTL.String(),
			"max_memory_mb":       c.Cache.MaxMemoryMB,
			"eviction_policy":     c.Cache.EvictionPolicy,
			"local_cache_enabled": c.Cache.LocalCacheEnabled,
			"local_cache_size":    c.Cache.LocalCacheSize,
		},
		"metrics": map[string]interface{}{
			"enabled": c.Metrics.Enabled,
			"path":    c.Metrics.Path,
		},
		"logging": map[string]interface{}{
			"service_address": c.Logging.ServiceAddress,
			"batch_size":      c.Logging.BatchSize,
			"flush_interval":  c.Logging.FlushInterval.String(),
			"buffer_size":     c.Logging.BufferSize,
			"enabled":         c.Logging.Enabled,
		},
		"tracing": map[string]interface{}{
			"enabled":  c.Tracing.Enabled,
			"endpoint": c.Tracing.Endpoint,
		},
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func getEnvStringSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		return strings.Split(value, ",")
	}
	return defaultValue
}

func maskSecret(s string) string {
	if s == "" {
		return "<not set>"
	}
	return fmt.Sprintf("<set, %d chars>", len(s))
}

func maskURL(s string) string {
	if s == "" {
		return "<not set>"
	}
	return "<set>"
}
