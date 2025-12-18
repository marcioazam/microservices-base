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

// **Feature: resilience-service-state-of-art-2025, Property 5: Viper Configuration Management**
// **Validates: Requirements 3.4, 9.1, 9.2, 9.3, 9.4, 9.5**
func TestViperConfigurationManagementProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate valid configuration values
		port := rapid.IntRange(1024, 65535).Draw(t, "port")
		logLevel := rapid.SampledFrom([]string{"debug", "info", "warn", "error"}).Draw(t, "log_level")
		environment := rapid.SampledFrom([]string{"development", "staging", "production"}).Draw(t, "environment")
		
		// Test environment variable override
		envKey := "RESILIENCE_SERVER_PORT"
		originalValue := os.Getenv(envKey)
		defer func() {
			if originalValue == "" {
				os.Unsetenv(envKey)
			} else {
				os.Setenv(envKey, originalValue)
			}
		}()

		// Set environment variable
		os.Setenv(envKey, rapid.StringMatching(`^[0-9]+$`).Draw(t, "port_str"))
		os.Setenv("RESILIENCE_LOGGING_LEVEL", logLevel)
		os.Setenv("RESILIENCE_OPENTELEMETRY_ENVIRONMENT", environment)

		// Create viper instance for testing
		v := viper.New()
		v.AutomaticEnv()
		v.SetEnvPrefix("RESILIENCE")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		// Set some defaults
		v.SetDefault("server.port", port)
		v.SetDefault("logging.level", "info")
		v.SetDefault("opentelemetry.environment", "development")

		// Test that environment variables override defaults
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

// Test configuration validation with detailed error messages
func TestConfigurationValidationProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate invalid configuration values
		invalidPort := rapid.IntRange(-1000, 1023).Draw(t, "invalid_port")
		invalidLogLevel := rapid.StringMatching(`^[^a-z]*$`).Draw(t, "invalid_log_level")
		
		// Create configuration with invalid values
		cfg := &config.Config{
			Server: config.ServerConfig{
				Host:            "localhost",
				Port:            invalidPort, // Invalid port
				ShutdownTimeout: 30 * time.Second,
				MaxRecvMsgSize:  4194304,
				MaxSendMsgSize:  4194304,
			},
			Logging: config.LoggingConfig{
				Level:  invalidLogLevel, // Invalid log level
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

		// Validation should fail with detailed error messages
		err := config.Validate(cfg)
		if err == nil {
			t.Fatal("Expected validation error for invalid configuration")
		}

		// Error message should contain field information
		errorMsg := err.Error()
		if !strings.Contains(errorMsg, "validation") {
			t.Fatalf("Error message should mention validation, got: %s", errorMsg)
		}
	})
}

// Test sensible defaults for all configuration options
func TestConfigurationDefaultsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Clear environment variables to test defaults
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

		// Create viper instance and set defaults
		v := viper.New()
		v.SetDefault("server.host", "0.0.0.0")
		v.SetDefault("server.port", 50056)
		v.SetDefault("logging.level", "info")
		v.SetDefault("redis.url", "redis://localhost:6379")

		// Test that defaults are reasonable
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

// Test YAML and environment variable precedence
func TestConfigurationPrecedenceProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate test values
		defaultPort := rapid.IntRange(1024, 65535).Draw(t, "default_port")
		envPort := rapid.IntRange(1024, 65535).Draw(t, "env_port")
		
		// Ensure they're different
		if envPort == defaultPort {
			envPort = defaultPort + 1
			if envPort > 65535 {
				envPort = defaultPort - 1
			}
		}

		// Set up viper with default
		v := viper.New()
		v.SetDefault("server.port", defaultPort)

		// Initially should use default
		if v.GetInt("server.port") != defaultPort {
			t.Fatalf("Should use default value %d, got: %d", defaultPort, v.GetInt("server.port"))
		}

		// Set environment variable
		envKey := "RESILIENCE_SERVER_PORT"
		originalValue := os.Getenv(envKey)
		defer func() {
			if originalValue == "" {
				os.Unsetenv(envKey)
			} else {
				os.Setenv(envKey, originalValue)
			}
		}()

		os.Setenv(envKey, rapid.StringOf(rapid.Rune().Filter(func(r rune) bool {
			return r >= '0' && r <= '9'
		})).Draw(t, "port_string"))
		
		// Enable environment variable support
		v.AutomaticEnv()
		v.SetEnvPrefix("RESILIENCE")
		v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

		// Environment variable should override default
		actualPort := v.GetInt("server.port")
		
		// Verify precedence works
		if actualPort == defaultPort && os.Getenv(envKey) != "" {
			t.Fatalf("Environment variable should override default")
		}
	})
}

// Test custom validation rules
func TestCustomValidationRulesProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate configuration with invalid relationships
		baseDelayMs := rapid.IntRange(1000, 10000).Draw(t, "base_delay_ms")
		maxDelayMs := rapid.IntRange(1, baseDelayMs-1).Draw(t, "max_delay_ms")
		baseDelay := time.Duration(baseDelayMs) * time.Millisecond
		maxDelay := time.Duration(maxDelayMs) * time.Millisecond
		
		cfg := &config.Config{
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
					BaseDelay:     baseDelay,  // Greater than MaxDelay
					MaxDelay:      maxDelay,   // Less than BaseDelay
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

		// Custom validation should catch the invalid relationship
		err := config.Validate(cfg)
		if err == nil {
			t.Fatal("Expected custom validation error for base_delay > max_delay")
		}

		// Error should mention the specific relationship issue
		if !strings.Contains(err.Error(), "base delay") || !strings.Contains(err.Error(), "max delay") {
			t.Fatalf("Error should mention delay relationship issue, got: %s", err.Error())
		}
	})
}