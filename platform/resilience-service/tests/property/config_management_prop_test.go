package property

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/config"
	"github.com/spf13/viper"
	"pgregory.net/rapid"
)

// TestViperConfigurationManagementProperty validates viper configuration management.
func TestViperConfigurationManagementProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		port := rapid.IntRange(1024, 65535).Draw(t, "port")
		logLevel := rapid.SampledFrom([]string{"debug", "info", "warn", "error"}).Draw(t, "log_level")
		environment := rapid.SampledFrom([]string{"development", "staging", "production"}).Draw(t, "environment")

		envKey := "RESILIENCE_SERVER_PORT"
		originalValue := os.Getenv(envKey)
		defer func() {
			if originalValue == "" {
				os.Unsetenv(envKey)
			} else {
				os.Setenv(envKey, originalValue)
			}
		}()

		os.Setenv("RESILIENCE_LOGGING_LEVEL", logLevel)
		os.Setenv("RESILIENCE_OPENTELEMETRY_ENVIRONMENT", environment)

		v := viper.New()
		v.AutomaticEnv()
		v.SetEnvPrefix("RESILIENCE")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		v.SetDefault("server.port", port)
		v.SetDefault("logging.level", "info")
		v.SetDefault("opentelemetry.environment", "development")

		if v.GetString("logging.level") != logLevel {
			t.Fatalf("Environment variable should override default, expected %s, got %s",
				logLevel, v.GetString("logging.level"))
		}

		if v.GetString("opentelemetry.environment") != environment {
			t.Fatalf("Environment variable should override default, expected %s, got %s",
				environment, v.GetString("opentelemetry.environment"))
		}
	})
}

// TestConfigurationValidationProperty tests configuration validation.
func TestConfigurationValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidPort := rapid.IntRange(-1000, 1023).Draw(t, "invalid_port")

		cfg := createValidConfig()
		cfg.Server.Port = invalidPort

		err := config.Validate(cfg)
		if err == nil {
			t.Fatal("Expected validation error for invalid configuration")
		}

		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "validation") {
			t.Fatalf("Error message should mention validation, got: %s", errorMsg)
		}
	})
}

// TestConfigurationDefaultsProperty tests configuration defaults.
func TestConfigurationDefaultsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		envVars := []string{
			"RESILIENCE_SERVER_HOST",
			"RESILIENCE_SERVER_PORT",
			"RESILIENCE_LOGGING_LEVEL",
			"RESILIENCE_REDIS_URL",
		}

		originalValues := make(map[string]string)
		for _, env := range envVars {
			originalValues[env] = os.Getenv(env)
			os.Unsetenv(env)
		}
		defer func() {
			for env, value := range originalValues {
				if value == "" {
					os.Unsetenv(env)
				} else {
					os.Setenv(env, value)
				}
			}
		}()

		v := viper.New()
		v.SetDefault("server.host", "0.0.0.0")
		v.SetDefault("server.port", 50056)
		v.SetDefault("logging.level", "info")
		v.SetDefault("redis.url", "redis://localhost:6379")

		if v.GetString("server.host") != "0.0.0.0" {
			t.Fatalf("Expected default host '0.0.0.0', got: %s", v.GetString("server.host"))
		}

		port := v.GetInt("server.port")
		if port < 1024 || port > 65535 {
			t.Fatalf("Default port should be in valid range, got: %d", port)
		}

		logLevel := v.GetString("logging.level")
		validLevels := []string{"debug", "info", "warn", "error"}
		found := false
		for _, level := range validLevels {
			if logLevel == level {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Default log level should be valid, got: %s", logLevel)
		}

		redisURL := v.GetString("redis.url")
		if !strings.HasPrefix(redisURL, "redis://") {
			t.Fatalf("Default Redis URL should have redis:// scheme, got: %s", redisURL)
		}
	})
}


// TestCustomValidationRulesProperty tests custom validation rules.
func TestCustomValidationRulesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelayMs := rapid.IntRange(2000, 10000).Draw(t, "base_delay_ms")
		maxDelayMs := rapid.IntRange(1000, baseDelayMs-1).Draw(t, "max_delay_ms")
		baseDelay := time.Duration(baseDelayMs) * time.Millisecond
		maxDelay := time.Duration(maxDelayMs) * time.Millisecond

		cfg := createValidConfig()
		cfg.Defaults.Retry.BaseDelay = baseDelay
		cfg.Defaults.Retry.MaxDelay = maxDelay

		err := config.Validate(cfg)
		if err == nil {
			t.Fatal("Expected custom validation error for base_delay > max_delay")
		}
	})
}

// createValidConfig creates a valid configuration for testing.
func createValidConfig() *config.Config {
	return &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            8080,
			ShutdownTimeout: 30 * time.Second,
			MaxRecvMsgSize:  4194304,
			MaxSendMsgSize:  4194304,
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		OpenTelemetry: config.OTelConfig{
			Endpoint:       "http://localhost:4317",
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "development",
			Timeout:        10 * time.Second,
		},
		Redis: config.RedisConfig{
			URL:            "redis://localhost:6379",
			DB:             0,
			ConnectTimeout: 5 * time.Second,
			ReadTimeout:    3 * time.Second,
			WriteTimeout:   3 * time.Second,
			MaxRetries:     3,
			PoolSize:       10,
		},
		Policies: config.PoliciesConfig{
			ConfigPath:     "/etc/resilience/policies.yaml",
			ReloadInterval: 30 * time.Second,
		},
		Defaults: config.DefaultsConfig{
			CircuitBreaker: config.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 3,
				Timeout:          30 * time.Second,
				ProbeCount:       1,
			},
			Retry: config.RetryConfig{
				MaxAttempts:   3,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			Timeout: config.TimeoutConfig{
				Default: 5 * time.Second,
				Max:     5 * time.Minute,
			},
			RateLimit: config.RateLimitConfig{
				Algorithm: "token_bucket",
				Limit:     1000,
				Window:    time.Minute,
				BurstSize: 100,
			},
			Bulkhead: config.BulkheadConfig{
				MaxConcurrent: 100,
				MaxQueue:      50,
				QueueTimeout:  5 * time.Second,
			},
		},
	}
}
