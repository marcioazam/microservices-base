// Package config provides configuration types for the resilience service.
package config

import (
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
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
	URL      string `yaml:"url" env:"REDIS_URL" default:"redis://localhost:6379"`
	DB       int    `yaml:"db" default:"0"`
	Password string `yaml:"password" env:"REDIS_PASSWORD"`
}

// OTELConfig defines OpenTelemetry settings.
type OTELConfig struct {
	Endpoint    string `yaml:"endpoint" env:"OTEL_EXPORTER_OTLP_ENDPOINT" default:"http://localhost:4317"`
	ServiceName string `yaml:"serviceName" default:"resilience-service"`
	Insecure    bool   `yaml:"insecure" default:"true"`
}

// PolicyConfig defines policy management settings.
type PolicyConfig struct {
	ConfigPath     string        `yaml:"configPath" env:"POLICY_CONFIG_PATH" default:"/etc/resilience/policies.yaml"`
	ReloadInterval time.Duration `yaml:"reloadInterval" default:"30s"`
}

// DefaultsConfig defines default resilience settings.
type DefaultsConfig struct {
	CircuitBreaker domain.CircuitBreakerConfig `yaml:"circuitBreaker"`
	Retry          domain.RetryConfig          `yaml:"retry"`
	Timeout        domain.TimeoutConfig        `yaml:"timeout"`
	RateLimit      domain.RateLimitConfig      `yaml:"rateLimit"`
	Bulkhead       domain.BulkheadConfig       `yaml:"bulkhead"`
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
			URL: "redis://localhost:6379",
			DB:  0,
		},
		OTEL: OTELConfig{
			Endpoint:    "http://localhost:4317",
			ServiceName: "resilience-service",
			Insecure:    true,
		},
		Policy: PolicyConfig{
			ConfigPath:     "/etc/resilience/policies.yaml",
			ReloadInterval: 30 * time.Second,
		},
		Defaults: DefaultsConfig{
			CircuitBreaker: domain.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
				ProbeCount:       1,
			},
			Retry: domain.RetryConfig{
				MaxAttempts:   3,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			Timeout: domain.TimeoutConfig{
				Default: 5 * time.Second,
				Max:     5 * time.Minute,
			},
			RateLimit: domain.RateLimitConfig{
				Algorithm: domain.TokenBucket,
				Limit:     1000,
				Window:    time.Minute,
				BurstSize: 100,
			},
			Bulkhead: domain.BulkheadConfig{
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
