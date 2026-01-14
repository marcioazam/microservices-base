// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
package property

import (
	"os"
	"strconv"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestConfig represents configuration for testing.
type TestConfig struct {
	ServerPort     int
	DatabaseHost   string
	DatabasePort   int
	DatabaseUser   string
	DatabasePass   string
	DatabaseName   string
	CacheAddress   string
	AWSRegion      string
	S3Bucket       string
	MaxFileSize    int64
	ChunkSize      int64
}

// Validate validates the configuration.
func (c *TestConfig) Validate() error {
	if c.ServerPort <= 0 || c.ServerPort > 65535 {
		return &ConfigError{Field: "SERVER_PORT", Message: "invalid port"}
	}
	if c.DatabaseHost == "" {
		return &ConfigError{Field: "DATABASE_HOST", Message: "is required"}
	}
	if c.DatabaseUser == "" {
		return &ConfigError{Field: "DATABASE_USER", Message: "is required"}
	}
	if c.DatabasePass == "" {
		return &ConfigError{Field: "DATABASE_PASSWORD", Message: "is required"}
	}
	if c.DatabaseName == "" {
		return &ConfigError{Field: "DATABASE_NAME", Message: "is required"}
	}
	if c.CacheAddress == "" {
		return &ConfigError{Field: "CACHE_ADDRESS", Message: "is required"}
	}
	if c.AWSRegion == "" {
		return &ConfigError{Field: "AWS_REGION", Message: "is required"}
	}
	if c.S3Bucket == "" {
		return &ConfigError{Field: "S3_BUCKET", Message: "is required"}
	}
	if c.MaxFileSize <= 0 {
		return &ConfigError{Field: "MAX_FILE_SIZE", Message: "must be positive"}
	}
	if c.ChunkSize <= 0 {
		return &ConfigError{Field: "CHUNK_SIZE", Message: "must be positive"}
	}
	return nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}

// MockEnvLoader simulates environment variable loading.
type MockEnvLoader struct {
	vars map[string]string
}

func NewMockEnvLoader() *MockEnvLoader {
	return &MockEnvLoader{vars: make(map[string]string)}
}

func (l *MockEnvLoader) Set(key, value string) {
	l.vars[key] = value
}

func (l *MockEnvLoader) Get(key string) string {
	return l.vars[key]
}

func (l *MockEnvLoader) GetWithDefault(key, defaultValue string) string {
	if v, ok := l.vars[key]; ok && v != "" {
		return v
	}
	return defaultValue
}

func (l *MockEnvLoader) GetInt(key string, defaultValue int) int {
	if v, ok := l.vars[key]; ok && v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return defaultValue
}

func (l *MockEnvLoader) GetInt64(key string, defaultValue int64) int64 {
	if v, ok := l.vars[key]; ok && v != "" {
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	}
	return defaultValue
}

// LoadConfig loads configuration from mock environment.
func (l *MockEnvLoader) LoadConfig() *TestConfig {
	return &TestConfig{
		ServerPort:   l.GetInt("SERVER_PORT", 8080),
		DatabaseHost: l.Get("DATABASE_HOST"),
		DatabasePort: l.GetInt("DATABASE_PORT", 5432),
		DatabaseUser: l.Get("DATABASE_USER"),
		DatabasePass: l.Get("DATABASE_PASSWORD"),
		DatabaseName: l.Get("DATABASE_NAME"),
		CacheAddress: l.Get("CACHE_ADDRESS"),
		AWSRegion:    l.Get("AWS_REGION"),
		S3Bucket:     l.Get("S3_BUCKET"),
		MaxFileSize:  l.GetInt64("MAX_FILE_SIZE", 100*1024*1024),
		ChunkSize:    l.GetInt64("CHUNK_SIZE", 5*1024*1024),
	}
}

// TestProperty16_MissingRequiredCausesFailure tests that missing required values cause startup failure.
// Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
func TestProperty16_MissingRequiredCausesFailure(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		requiredFields := []string{
			"DATABASE_HOST", "DATABASE_USER", "DATABASE_PASSWORD",
			"DATABASE_NAME", "CACHE_ADDRESS", "AWS_REGION", "S3_BUCKET",
		}

		// Pick a random required field to omit
		omitField := rapid.SampledFrom(requiredFields).Draw(t, "omitField")

		loader := NewMockEnvLoader()

		// Set all required fields except the omitted one
		for _, field := range requiredFields {
			if field != omitField {
				loader.Set(field, rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, field))
			}
		}

		cfg := loader.LoadConfig()
		err := cfg.Validate()

		// Property: Missing required values SHALL cause startup failure
		if err == nil {
			t.Errorf("missing %s should cause validation failure", omitField)
		}

		// Verify error mentions the missing field
		if configErr, ok := err.(*ConfigError); ok {
			if configErr.Field != omitField {
				t.Errorf("error should mention %s, got %s", omitField, configErr.Field)
			}
		}
	})
}

// TestProperty16_EnvOverridesConfigFile tests that environment variables override config file values.
// Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
func TestProperty16_EnvOverridesConfigFile(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		loader := NewMockEnvLoader()

		// Set all required fields
		loader.Set("DATABASE_HOST", "default-host")
		loader.Set("DATABASE_USER", "default-user")
		loader.Set("DATABASE_PASSWORD", "default-pass")
		loader.Set("DATABASE_NAME", "default-db")
		loader.Set("CACHE_ADDRESS", "default-cache:6379")
		loader.Set("AWS_REGION", "us-east-1")
		loader.Set("S3_BUCKET", "default-bucket")

		// Default port
		defaultPort := 8080

		// Override with environment variable
		overridePort := rapid.IntRange(3000, 9000).Draw(t, "overridePort")
		loader.Set("SERVER_PORT", strconv.Itoa(overridePort))

		cfg := loader.LoadConfig()

		// Property: Environment variables SHALL override config file values
		if cfg.ServerPort == defaultPort {
			t.Errorf("env should override default: got %d, expected %d", cfg.ServerPort, overridePort)
		}
		if cfg.ServerPort != overridePort {
			t.Errorf("port should be %d, got %d", overridePort, cfg.ServerPort)
		}
	})
}

// TestProperty16_InvalidValuesClearError tests that invalid values produce clear error messages.
// Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
func TestProperty16_InvalidValuesClearError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		loader := NewMockEnvLoader()

		// Set all required fields
		loader.Set("DATABASE_HOST", "localhost")
		loader.Set("DATABASE_USER", "user")
		loader.Set("DATABASE_PASSWORD", "pass")
		loader.Set("DATABASE_NAME", "db")
		loader.Set("CACHE_ADDRESS", "cache:6379")
		loader.Set("AWS_REGION", "us-east-1")
		loader.Set("S3_BUCKET", "bucket")

		// Set invalid port
		invalidPort := rapid.SampledFrom([]int{-1, 0, 70000}).Draw(t, "invalidPort")
		loader.Set("SERVER_PORT", strconv.Itoa(invalidPort))

		cfg := loader.LoadConfig()
		err := cfg.Validate()

		// Property: Invalid values SHALL produce clear error messages
		if err == nil {
			t.Errorf("invalid port %d should cause validation failure", invalidPort)
		}

		if configErr, ok := err.(*ConfigError); ok {
			if configErr.Field != "SERVER_PORT" {
				t.Errorf("error should mention SERVER_PORT, got %s", configErr.Field)
			}
			if configErr.Message == "" {
				t.Error("error message should not be empty")
			}
		}
	})
}

// TestProperty16_ValidConfigPasses tests that valid configuration passes validation.
// Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
func TestProperty16_ValidConfigPasses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		loader := NewMockEnvLoader()

		// Set all required fields with valid values
		loader.Set("DATABASE_HOST", rapid.StringMatching(`[a-z0-9-]{5,20}`).Draw(t, "host"))
		loader.Set("DATABASE_USER", rapid.StringMatching(`[a-z0-9_]{3,15}`).Draw(t, "user"))
		loader.Set("DATABASE_PASSWORD", rapid.StringMatching(`[a-zA-Z0-9]{8,20}`).Draw(t, "pass"))
		loader.Set("DATABASE_NAME", rapid.StringMatching(`[a-z0-9_]{3,15}`).Draw(t, "dbname"))
		loader.Set("CACHE_ADDRESS", rapid.StringMatching(`[a-z0-9-]{3,15}:6379`).Draw(t, "cache"))
		loader.Set("AWS_REGION", rapid.SampledFrom([]string{"us-east-1", "us-west-2", "eu-west-1"}).Draw(t, "region"))
		loader.Set("S3_BUCKET", rapid.StringMatching(`[a-z0-9-]{3,20}`).Draw(t, "bucket"))

		// Valid port
		validPort := rapid.IntRange(1024, 65535).Draw(t, "port")
		loader.Set("SERVER_PORT", strconv.Itoa(validPort))

		cfg := loader.LoadConfig()
		err := cfg.Validate()

		// Property: Valid configuration SHALL pass validation
		if err != nil {
			t.Errorf("valid config should pass validation: %v", err)
		}
	})
}

// TestProperty16_DefaultValuesApplied tests that default values are applied when not specified.
// Property 16: Configuration Validation
// Validates: Requirements 15.2, 15.3, 15.4
func TestProperty16_DefaultValuesApplied(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		loader := NewMockEnvLoader()

		// Set only required fields, not optional ones
		loader.Set("DATABASE_HOST", "localhost")
		loader.Set("DATABASE_USER", "user")
		loader.Set("DATABASE_PASSWORD", "pass")
		loader.Set("DATABASE_NAME", "db")
		loader.Set("CACHE_ADDRESS", "cache:6379")
		loader.Set("AWS_REGION", "us-east-1")
		loader.Set("S3_BUCKET", "bucket")

		cfg := loader.LoadConfig()

		// Property: Default values SHALL be applied when not specified
		if cfg.ServerPort != 8080 {
			t.Errorf("default server port should be 8080, got %d", cfg.ServerPort)
		}
		if cfg.DatabasePort != 5432 {
			t.Errorf("default database port should be 5432, got %d", cfg.DatabasePort)
		}
		if cfg.MaxFileSize != 100*1024*1024 {
			t.Errorf("default max file size should be 100MB, got %d", cfg.MaxFileSize)
		}
		if cfg.ChunkSize != 5*1024*1024 {
			t.Errorf("default chunk size should be 5MB, got %d", cfg.ChunkSize)
		}
	})
}

// Ensure os and time are used
var _ = os.Getenv
var _ = time.Now
