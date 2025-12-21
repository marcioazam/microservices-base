// Package config provides configuration types for the resilience service.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/auth-platform/libs/go/resilience"
)

// Config represents the complete service configuration.
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Redis    RedisConfig    `yaml:"redis"`
	OTEL     OTELConfig     `yaml:"otel"`
	Policy   PolicyConfig   `yaml:"policy"`
	Defaults DefaultsConfig `yaml:"defaults"`
	Log      LogConfig      `yaml:"log"`
}

// ServerConfig defines server settings.
type ServerConfig struct {
	Host            string        `yaml:"host" env:"RESILIENCE_HOST" default:"0.0.0.0"`
	Port            int           `yaml:"port" env:"RESILIENCE_PORT" default:"50056"`
	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" default:"30s"`
}

// RedisConfig defines Redis connection settings.
type RedisConfig struct {
	URL           string `yaml:"url" env:"REDIS_URL" default:"redis://localhost:6379"`
	DB            int    `yaml:"db" default:"0"`
	Password      string `yaml:"password" env:"REDIS_PASSWORD"`
	TLSEnabled    bool   `yaml:"tlsEnabled" env:"REDIS_TLS_ENABLED" default:"true"`
	TLSSkipVerify bool   `yaml:"tlsSkipVerify" env:"REDIS_TLS_SKIP_VERIFY" default:"false"`
}

// OTELConfig defines OpenTelemetry settings.
type OTELConfig struct {
	Endpoint    string `yaml:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"http://localhost:4317"`
	ServiceName string `yaml:"serviceName" default:"resilience-service"`
	Insecure    bool   `yaml:"insecure" default:"false"`
}

// PolicyConfig defines policy management settings.
type PolicyConfig struct {
	ConfigPath     string        `yaml:"configPath" env:"POLICY_CONFIG_PATH" default:"/etc/resilience/policies.yaml"`
	ReloadInterval time.Duration `yaml:"reloadInterval" default:"30s"`
}

// DefaultsConfig defines default resilience settings.
type DefaultsConfig struct {
	CircuitBreaker resilience.CircuitBreakerConfig `yaml:"circuitBreaker"`
	Retry          resilience.RetryConfig          `yaml:"retry"`
	Timeout        resilience.TimeoutConfig        `yaml:"timeout"`
	RateLimit      resilience.RateLimitConfig      `yaml:"rateLimit"`
	Bulkhead       resilience.BulkheadConfig       `yaml:"bulkhead"`
}

// LogConfig defines logging settings.
type LogConfig struct {
	Level  string `yaml:"level" env:"LOG_LEVEL" default:"info"`
	Format string `yaml:"format" default:"json"`
}

// NewDefaultConfig returns configuration with sensible defaults.
func NewDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:            "0.0.0.0",
			Port:            50056,
			ShutdownTimeout: 30 * time.Second,
		},
		Redis: RedisConfig{
			URL:           "redis://localhost:6379",
			DB:            0,
			TLSEnabled:    false, // Default false for local development
			TLSSkipVerify: false,
		},
		OTEL: OTELConfig{
			Endpoint:    "http://localhost:4317",
			ServiceName: "resilience-service",
			Insecure:    true, // Default true for local development
		},
		Policy: PolicyConfig{
			ConfigPath:     "/etc/resilience/policies.yaml",
			ReloadInterval: 30 * time.Second,
		},
		Defaults: DefaultsConfig{
			CircuitBreaker: resilience.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
				ProbeCount:       1,
			},
			Retry: resilience.RetryConfig{
				MaxAttempts:   3,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			Timeout: resilience.TimeoutConfig{
				Default: 5 * time.Second,
				Max:     5 * time.Minute,
			},
			RateLimit: resilience.RateLimitConfig{
				Algorithm: resilience.TokenBucket,
				Limit:     1000,
				Window:    time.Minute,
				BurstSize: 100,
			},
			Bulkhead: resilience.BulkheadConfig{
				MaxConcurrent: 100,
				MaxQueue:      50,
				QueueTimeout:  5 * time.Second,
			},
		},
		Log: LogConfig{
			Level:  "info",
			Format: "json",
		},
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.Server.Validate(); err != nil {
		return fmt.Errorf("server config: %w", err)
	}
	if err := c.Redis.Validate(); err != nil {
		return fmt.Errorf("redis config: %w", err)
	}
	if err := c.OTEL.Validate(); err != nil {
		return fmt.Errorf("otel config: %w", err)
	}
	if err := c.Policy.Validate(); err != nil {
		return fmt.Errorf("policy config: %w", err)
	}
	if err := c.Log.Validate(); err != nil {
		return fmt.Errorf("log config: %w", err)
	}
	return nil
}

// Validate validates server configuration.
func (s *ServerConfig) Validate() error {
	if s.Port < 1024 || s.Port > 65535 {
		return fmt.Errorf("invalid port: %d (must be 1024-65535)", s.Port)
	}
	if s.ShutdownTimeout <= 0 {
		return fmt.Errorf("shutdown timeout must be positive")
	}
	return nil
}

// Validate validates Redis configuration.
func (r *RedisConfig) Validate() error {
	if r.URL == "" {
		return fmt.Errorf("redis URL is required")
	}

	// Enforce TLS in production
	if isProd() && r.TLSEnabled {
		if !strings.HasPrefix(r.URL, "rediss://") {
			return fmt.Errorf("production requires TLS: use rediss:// scheme")
		}
		if r.TLSSkipVerify {
			return fmt.Errorf("TLS verification cannot be skipped in production")
		}
	}

	return nil
}

// Validate validates OTEL configuration.
func (o *OTELConfig) Validate() error {
	if o.ServiceName == "" {
		return fmt.Errorf("service name is required")
	}
	if o.Endpoint == "" {
		return fmt.Errorf("OTLP endpoint is required")
	}

	// Warn about insecure configuration in production
	if isProd() && o.Insecure {
		return fmt.Errorf("insecure OTLP not allowed in production")
	}

	return nil
}

// Validate validates policy configuration.
func (p *PolicyConfig) Validate() error {
	if p.ReloadInterval <= 0 {
		return fmt.Errorf("reload interval must be positive")
	}
	return nil
}

// Validate validates log configuration.
func (l *LogConfig) Validate() error {
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
	}
	if !validLevels[strings.ToLower(l.Level)] {
		return fmt.Errorf("invalid log level: %s (must be debug/info/warn/error)", l.Level)
	}
	return nil
}

// isProd checks if running in production environment.
func isProd() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	return env == "production" || env == "prod"
}
