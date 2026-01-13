// Package config provides centralized configuration management for IAM Policy Service.
package config

import (
	"fmt"
	"time"

	libconfig "github.com/authcorp/libs/go/src/config"
)

// Config holds all service configuration.
type Config struct {
	Server  ServerConfig
	Policy  PolicyConfig
	Cache   CacheConfig
	Logging LoggingConfig
	CAEP    CAEPConfig
	Metrics MetricsConfig
	Tracing TracingConfig
	Crypto  *CryptoConfig

	// Convenience fields for backward compatibility
	Host            string
	Port            int
	HealthPort      int
	ShutdownTimeout time.Duration
	PolicyPath      string
}

// ServerConfig holds server-related configuration.
type ServerConfig struct {
	GRPCPort        int
	HealthPort      int
	MetricsPort     int
	ShutdownTimeout time.Duration
}

// PolicyConfig holds policy engine configuration.
type PolicyConfig struct {
	Path         string
	CacheTTL     time.Duration
	WatchEnabled bool
}

// CacheConfig holds cache client configuration.
type CacheConfig struct {
	Address        string
	Namespace      string
	LocalFallback  bool
	LocalCacheSize int
	Timeout        time.Duration
	TTL            time.Duration
}

// LoggingConfig holds logging client configuration.
type LoggingConfig struct {
	Address       string
	ServiceName   string
	MinLevel      string
	LocalFallback bool
	BufferSize    int
	FlushInterval time.Duration
}

// CAEPConfig holds CAEP emitter configuration.
type CAEPConfig struct {
	Enabled        bool
	Transmitter    string
	TransmitterURL string
	ServiceToken   string
	Issuer         string
}

// MetricsConfig holds metrics configuration.
type MetricsConfig struct {
	Enabled bool
	Path    string
}

// TracingConfig holds tracing configuration.
type TracingConfig struct {
	Enabled  bool
	Endpoint string
}

// CryptoConfig holds crypto client configuration.
type CryptoConfig struct {
	Enabled           bool
	Address           string
	Timeout           time.Duration
	CacheEncryption   bool
	DecisionSigning   bool
	EncryptionKeyID   string
	SigningKeyID      string
	KeyCacheTTL       time.Duration
}

// defaults returns default configuration values.
func defaults() map[string]any {
	return map[string]any{
		"server.grpc.port":        50054,
		"server.health.port":      8080,
		"server.metrics.port":     9090,
		"server.shutdown.timeout": "30s",

		"policy.path":          "./policies",
		"policy.cache.ttl":     "5m",
		"policy.watch.enabled": true,

		"cache.address":         "localhost:50051",
		"cache.namespace":       "iam-policy",
		"cache.local.fallback":  true,
		"cache.local.size":      10000,
		"cache.timeout":         "100ms",
		"cache.ttl":             "5m",

		"logging.address":        "localhost:50052",
		"logging.service.name":   "iam-policy-service",
		"logging.min.level":      "info",
		"logging.local.fallback": true,
		"logging.buffer.size":    1000,
		"logging.flush.interval": "5s",

		"caep.enabled":       false,
		"caep.transmitter":   "",
		"caep.service.token": "",
		"caep.issuer":        "iam-policy-service",

		"metrics.enabled": true,
		"metrics.path":    "/metrics",

		"tracing.enabled":  false,
		"tracing.endpoint": "",

		"crypto.enabled":            false,
		"crypto.address":            "localhost:50051",
		"crypto.timeout":            "5s",
		"crypto.cache.encryption":   false,
		"crypto.decision.signing":   false,
		"crypto.encryption.key.id":  "iam-policy/cache-encryption/1",
		"crypto.signing.key.id":     "iam-policy/decision-signing/1",
		"crypto.key.cache.ttl":      "5m",
	}
}

// Load loads configuration from environment variables and optional config file.
func Load() (*Config, error) {
	cfg := libconfig.New().WithDefaults(defaults()).LoadEnv("IAM_POLICY")

	return parseConfig(cfg)
}

// LoadFromFile loads configuration from a YAML file.
func LoadFromFile(path string) (*Config, error) {
	cfg := libconfig.New().WithDefaults(defaults())

	if err := cfg.LoadFile(path); err != nil {
		return nil, fmt.Errorf("failed to load config file: %w", err)
	}

	cfg.LoadEnv("IAM_POLICY")
	return parseConfig(cfg)
}

func parseConfig(cfg *libconfig.Config) (*Config, error) {
	shutdownTimeout, err := time.ParseDuration(cfg.GetString("server.shutdown.timeout"))
	if err != nil {
		shutdownTimeout = 30 * time.Second
	}

	cacheTTL, err := time.ParseDuration(cfg.GetString("policy.cache.ttl"))
	if err != nil {
		cacheTTL = 5 * time.Minute
	}

	cacheTimeout, err := time.ParseDuration(cfg.GetString("cache.timeout"))
	if err != nil {
		cacheTimeout = 100 * time.Millisecond
	}

	cacheTTLDuration, err := time.ParseDuration(cfg.GetString("cache.ttl"))
	if err != nil {
		cacheTTLDuration = 5 * time.Minute
	}

	flushInterval, err := time.ParseDuration(cfg.GetString("logging.flush.interval"))
	if err != nil {
		flushInterval = 5 * time.Second
	}

	cryptoTimeout, err := time.ParseDuration(cfg.GetString("crypto.timeout"))
	if err != nil {
		cryptoTimeout = 5 * time.Second
	}

	keyCacheTTL, err := time.ParseDuration(cfg.GetString("crypto.key.cache.ttl"))
	if err != nil {
		keyCacheTTL = 5 * time.Minute
	}

	transmitterURL := cfg.GetString("caep.transmitter")

	cryptoConfig := &CryptoConfig{
		Enabled:           cfg.GetBool("crypto.enabled"),
		Address:           cfg.GetString("crypto.address"),
		Timeout:           cryptoTimeout,
		CacheEncryption:   cfg.GetBool("crypto.cache.encryption"),
		DecisionSigning:   cfg.GetBool("crypto.decision.signing"),
		EncryptionKeyID:   cfg.GetString("crypto.encryption.key.id"),
		SigningKeyID:      cfg.GetString("crypto.signing.key.id"),
		KeyCacheTTL:       keyCacheTTL,
	}

	return &Config{
		Server: ServerConfig{
			GRPCPort:        cfg.GetInt("server.grpc.port"),
			HealthPort:      cfg.GetInt("server.health.port"),
			MetricsPort:     cfg.GetInt("server.metrics.port"),
			ShutdownTimeout: shutdownTimeout,
		},
		Policy: PolicyConfig{
			Path:         cfg.GetString("policy.path"),
			CacheTTL:     cacheTTL,
			WatchEnabled: cfg.GetBool("policy.watch.enabled"),
		},
		Cache: CacheConfig{
			Address:        cfg.GetString("cache.address"),
			Namespace:      cfg.GetString("cache.namespace"),
			LocalFallback:  cfg.GetBool("cache.local.fallback"),
			LocalCacheSize: cfg.GetInt("cache.local.size"),
			Timeout:        cacheTimeout,
			TTL:            cacheTTLDuration,
		},
		Logging: LoggingConfig{
			Address:       cfg.GetString("logging.address"),
			ServiceName:   cfg.GetString("logging.service.name"),
			MinLevel:      cfg.GetString("logging.min.level"),
			LocalFallback: cfg.GetBool("logging.local.fallback"),
			BufferSize:    cfg.GetInt("logging.buffer.size"),
			FlushInterval: flushInterval,
		},
		CAEP: CAEPConfig{
			Enabled:        cfg.GetBool("caep.enabled"),
			Transmitter:    transmitterURL,
			TransmitterURL: transmitterURL,
			ServiceToken:   cfg.GetString("caep.service.token"),
			Issuer:         cfg.GetString("caep.issuer"),
		},
		Metrics: MetricsConfig{
			Enabled: cfg.GetBool("metrics.enabled"),
			Path:    cfg.GetString("metrics.path"),
		},
		Tracing: TracingConfig{
			Enabled:  cfg.GetBool("tracing.enabled"),
			Endpoint: cfg.GetString("tracing.endpoint"),
		},
		Crypto: cryptoConfig,

		// Convenience fields
		Host:            "0.0.0.0",
		Port:            cfg.GetInt("server.grpc.port"),
		HealthPort:      cfg.GetInt("server.health.port"),
		ShutdownTimeout: shutdownTimeout,
		PolicyPath:      cfg.GetString("policy.path"),
	}, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Server.GRPCPort <= 0 || c.Server.GRPCPort > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", c.Server.GRPCPort)
	}
	if c.Policy.Path == "" {
		return fmt.Errorf("policy path is required")
	}
	if c.Cache.Namespace == "" {
		return fmt.Errorf("cache namespace is required")
	}
	if c.Logging.ServiceName == "" {
		return fmt.Errorf("logging service name is required")
	}
	if c.Crypto != nil {
		if err := c.Crypto.Validate(); err != nil {
			return fmt.Errorf("crypto config: %w", err)
		}
	}
	return nil
}

// Validate validates the crypto configuration.
func (c *CryptoConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Address == "" {
		return fmt.Errorf("crypto address is required when enabled")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("crypto timeout must be positive")
	}
	if c.CacheEncryption && c.EncryptionKeyID == "" {
		return fmt.Errorf("encryption key ID is required when cache encryption is enabled")
	}
	if c.DecisionSigning && c.SigningKeyID == "" {
		return fmt.Errorf("signing key ID is required when decision signing is enabled")
	}
	if c.EncryptionKeyID != "" {
		if err := ValidateKeyID(c.EncryptionKeyID); err != nil {
			return fmt.Errorf("invalid encryption key ID: %w", err)
		}
	}
	if c.SigningKeyID != "" {
		if err := ValidateKeyID(c.SigningKeyID); err != nil {
			return fmt.Errorf("invalid signing key ID: %w", err)
		}
	}
	return nil
}

// ValidateKeyID validates a key ID format (namespace/id/version).
func ValidateKeyID(keyID string) error {
	parts := splitKeyID(keyID)
	if len(parts) != 3 {
		return fmt.Errorf("key ID must be in format 'namespace/id/version', got: %s", keyID)
	}
	if parts[0] == "" {
		return fmt.Errorf("key namespace cannot be empty")
	}
	if parts[1] == "" {
		return fmt.Errorf("key id cannot be empty")
	}
	if parts[2] == "" {
		return fmt.Errorf("key version cannot be empty")
	}
	return nil
}

func splitKeyID(keyID string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(keyID); i++ {
		if keyID[i] == '/' {
			parts = append(parts, keyID[start:i])
			start = i + 1
		}
	}
	parts = append(parts, keyID[start:])
	return parts
}
