// Package config provides centralized configuration management using viper.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Config represents the complete service configuration with comprehensive validation.
type Config struct {
	Server        ServerConfig        `mapstructure:"server" validate:"required"`
	Redis         RedisConfig         `mapstructure:"redis" validate:"required"`
	OpenTelemetry OTelConfig          `mapstructure:"opentelemetry" validate:"required"`
	Logging       LoggingConfig       `mapstructure:"logging" validate:"required"`
	Policies      PoliciesConfig      `mapstructure:"policies" validate:"required"`
	Defaults      DefaultsConfig      `mapstructure:"defaults" validate:"required"`
}

// ServerConfig defines gRPC server settings with validation.
type ServerConfig struct {
	Host            string        `mapstructure:"host" validate:"required,hostname_rfc1123|ip"`
	Port            int           `mapstructure:"port" validate:"min=1024,max=65535"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout" validate:"min=1s,max=5m"`
	MaxRecvMsgSize  int           `mapstructure:"max_recv_msg_size" validate:"min=1024,max=67108864"`
	MaxSendMsgSize  int           `mapstructure:"max_send_msg_size" validate:"min=1024,max=67108864"`
}

// RedisConfig defines Redis connection settings with comprehensive validation.
type RedisConfig struct {
	URL              string        `mapstructure:"url" validate:"required,url"`
	DB               int           `mapstructure:"db" validate:"min=0,max=15"`
	Password         string        `mapstructure:"password"`
	TLSEnabled       bool          `mapstructure:"tls_enabled"`
	TLSSkipVerify    bool          `mapstructure:"tls_skip_verify"`
	ConnectTimeout   time.Duration `mapstructure:"connect_timeout" validate:"min=1s,max=30s"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout" validate:"min=1s,max=30s"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout" validate:"min=1s,max=30s"`
	MaxRetries       int           `mapstructure:"max_retries" validate:"min=0,max=10"`
	PoolSize         int           `mapstructure:"pool_size" validate:"min=1,max=100"`
}

// OTelConfig defines OpenTelemetry settings with validation.
type OTelConfig struct {
	Endpoint       string            `mapstructure:"endpoint" validate:"required,url"`
	ServiceName    string            `mapstructure:"service_name" validate:"required,min=1,max=100"`
	ServiceVersion string            `mapstructure:"service_version" validate:"required,semver"`
	Environment    string            `mapstructure:"environment" validate:"required,oneof=development staging production"`
	Insecure       bool              `mapstructure:"insecure"`
	Headers        map[string]string `mapstructure:"headers"`
	Timeout        time.Duration     `mapstructure:"timeout" validate:"min=1s,max=30s"`
}

// LoggingConfig defines logging settings with validation.
type LoggingConfig struct {
	Level  string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
	Format string `mapstructure:"format" validate:"required,oneof=json text"`
}

// PoliciesConfig defines policy management settings.
type PoliciesConfig struct {
	ConfigPath     string        `mapstructure:"config_path" validate:"required"`
	ReloadInterval time.Duration `mapstructure:"reload_interval" validate:"min=1s,max=1h"`
	WatchEnabled   bool          `mapstructure:"watch_enabled"`
}

// DefaultsConfig defines default resilience settings with validation.
type DefaultsConfig struct {
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker" validate:"required"`
	Retry          RetryConfig          `mapstructure:"retry" validate:"required"`
	Timeout        TimeoutConfig        `mapstructure:"timeout" validate:"required"`
	RateLimit      RateLimitConfig      `mapstructure:"rate_limit" validate:"required"`
	Bulkhead       BulkheadConfig       `mapstructure:"bulkhead" validate:"required"`
}

// CircuitBreakerConfig defines circuit breaker parameters with validation.
type CircuitBreakerConfig struct {
	FailureThreshold int           `mapstructure:"failure_threshold" validate:"min=1,max=100"`
	SuccessThreshold int           `mapstructure:"success_threshold" validate:"min=1,max=10"`
	Timeout          time.Duration `mapstructure:"timeout" validate:"min=1s,max=5m"`
	ProbeCount       int           `mapstructure:"probe_count" validate:"min=1,max=10"`
}

// RetryConfig defines retry behavior parameters with validation.
type RetryConfig struct {
	MaxAttempts   int           `mapstructure:"max_attempts" validate:"min=1,max=10"`
	BaseDelay     time.Duration `mapstructure:"base_delay" validate:"min=1ms,max=10s"`
	MaxDelay      time.Duration `mapstructure:"max_delay" validate:"min=1s,max=5m"`
	Multiplier    float64       `mapstructure:"multiplier" validate:"min=1.0,max=10.0"`
	JitterPercent float64       `mapstructure:"jitter_percent" validate:"min=0.0,max=1.0"`
}

// TimeoutConfig defines timeout parameters with validation.
type TimeoutConfig struct {
	Default time.Duration `mapstructure:"default" validate:"min=100ms,max=5m"`
	Max     time.Duration `mapstructure:"max" validate:"min=1s,max=10m"`
}

// RateLimitConfig defines rate limiting parameters with validation.
type RateLimitConfig struct {
	Algorithm string        `mapstructure:"algorithm" validate:"oneof=token_bucket sliding_window"`
	Limit     int           `mapstructure:"limit" validate:"min=1,max=100000"`
	Window    time.Duration `mapstructure:"window" validate:"min=1s,max=1h"`
	BurstSize int           `mapstructure:"burst_size" validate:"min=1,max=10000"`
}

// BulkheadConfig defines bulkhead isolation parameters with validation.
type BulkheadConfig struct {
	MaxConcurrent int           `mapstructure:"max_concurrent" validate:"min=1,max=10000"`
	MaxQueue      int           `mapstructure:"max_queue" validate:"min=0,max=10000"`
	QueueTimeout  time.Duration `mapstructure:"queue_timeout" validate:"min=1ms,max=30s"`
}

var (
	configValidator = validator.New()
)

// Load loads configuration from file and environment variables using viper.
func Load() (*Config, error) {
	v := viper.New()

	// Set configuration defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./configs")
	v.AddConfigPath("/etc/resilience")

	// Enable environment variable support
	v.AutomaticEnv()
	v.SetEnvPrefix("RESILIENCE")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read configuration file (optional)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable, we'll use defaults and env vars
	}

	// Unmarshal configuration
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := Validate(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration using struct tags and custom rules.
func Validate(config *Config) error {
	if err := configValidator.Struct(config); err != nil {
		return formatValidationError(err)
	}

	// Custom validation rules
	if err := validateCustomRules(config); err != nil {
		return err
	}

	return nil
}

// setDefaults sets sensible defaults for all configuration options.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 50056)
	v.SetDefault("server.shutdown_timeout", "30s")
	v.SetDefault("server.max_recv_msg_size", 4194304) // 4MB
	v.SetDefault("server.max_send_msg_size", 4194304) // 4MB

	// Redis defaults
	v.SetDefault("redis.url", "redis://localhost:6379")
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.tls_enabled", false)
	v.SetDefault("redis.tls_skip_verify", false)
	v.SetDefault("redis.connect_timeout", "5s")
	v.SetDefault("redis.read_timeout", "3s")
	v.SetDefault("redis.write_timeout", "3s")
	v.SetDefault("redis.max_retries", 3)
	v.SetDefault("redis.pool_size", 10)

	// OpenTelemetry defaults
	v.SetDefault("opentelemetry.endpoint", "http://localhost:4317")
	v.SetDefault("opentelemetry.service_name", "resilience-service")
	v.SetDefault("opentelemetry.service_version", "1.0.0")
	v.SetDefault("opentelemetry.environment", "development")
	v.SetDefault("opentelemetry.insecure", true)
	v.SetDefault("opentelemetry.timeout", "10s")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Policies defaults
	v.SetDefault("policies.config_path", "/etc/resilience/policies.yaml")
	v.SetDefault("policies.reload_interval", "30s")
	v.SetDefault("policies.watch_enabled", true)

	// Circuit breaker defaults
	v.SetDefault("defaults.circuit_breaker.failure_threshold", 5)
	v.SetDefault("defaults.circuit_breaker.success_threshold", 3)
	v.SetDefault("defaults.circuit_breaker.timeout", "30s")
	v.SetDefault("defaults.circuit_breaker.probe_count", 1)

	// Retry defaults
	v.SetDefault("defaults.retry.max_attempts", 3)
	v.SetDefault("defaults.retry.base_delay", "100ms")
	v.SetDefault("defaults.retry.max_delay", "10s")
	v.SetDefault("defaults.retry.multiplier", 2.0)
	v.SetDefault("defaults.retry.jitter_percent", 0.1)

	// Timeout defaults
	v.SetDefault("defaults.timeout.default", "5s")
	v.SetDefault("defaults.timeout.max", "5m")

	// Rate limit defaults
	v.SetDefault("defaults.rate_limit.algorithm", "token_bucket")
	v.SetDefault("defaults.rate_limit.limit", 1000)
	v.SetDefault("defaults.rate_limit.window", "1m")
	v.SetDefault("defaults.rate_limit.burst_size", 100)

	// Bulkhead defaults
	v.SetDefault("defaults.bulkhead.max_concurrent", 100)
	v.SetDefault("defaults.bulkhead.max_queue", 50)
	v.SetDefault("defaults.bulkhead.queue_timeout", "5s")
}

// validateCustomRules applies custom validation rules beyond struct tags.
func validateCustomRules(config *Config) error {
	// Validate timeout relationships
	if config.Defaults.Timeout.Default > config.Defaults.Timeout.Max {
		return fmt.Errorf("default timeout (%v) cannot be greater than max timeout (%v)",
			config.Defaults.Timeout.Default, config.Defaults.Timeout.Max)
	}

	// Validate retry delay relationships
	if config.Defaults.Retry.BaseDelay > config.Defaults.Retry.MaxDelay {
		return fmt.Errorf("base delay (%v) cannot be greater than max delay (%v)",
			config.Defaults.Retry.BaseDelay, config.Defaults.Retry.MaxDelay)
	}

	// Validate circuit breaker thresholds
	if config.Defaults.CircuitBreaker.SuccessThreshold > config.Defaults.CircuitBreaker.FailureThreshold {
		return fmt.Errorf("success threshold (%d) cannot be greater than failure threshold (%d)",
			config.Defaults.CircuitBreaker.SuccessThreshold, config.Defaults.CircuitBreaker.FailureThreshold)
	}

	// Validate production security settings
	if config.OpenTelemetry.Environment == "production" {
		if config.OpenTelemetry.Insecure {
			return fmt.Errorf("insecure OpenTelemetry not allowed in production")
		}
		// Enforce TLS for Redis in production
		if !config.Redis.TLSEnabled {
			return fmt.Errorf("TLS must be enabled for Redis in production")
		}
		if config.Redis.TLSSkipVerify {
			return fmt.Errorf("TLS verification cannot be skipped in production")
		}
	}

	return nil
}

// formatValidationError formats validation errors with detailed messages.
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			message := fmt.Sprintf("field '%s' failed validation: %s (value: %v)",
				fieldError.Field(), fieldError.Tag(), fieldError.Value())
			messages = append(messages, message)
		}
		return fmt.Errorf("validation errors: %s", strings.Join(messages, "; "))
	}
	return err
}