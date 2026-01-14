// Package config provides configuration management.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all service configuration.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Cache    CacheConfig
	Storage  StorageConfig
	Logging  LoggingConfig
	Upload   UploadConfig
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// DatabaseConfig holds database configuration.
type DatabaseConfig struct {
	Host         string
	Port         int
	User         string
	Password     string
	Database     string
	MaxOpenConns int
	MaxIdleConns int
}

// CacheConfig holds cache service configuration.
type CacheConfig struct {
	Address   string
	Namespace string
}

// StorageConfig holds S3 storage configuration.
type StorageConfig struct {
	Region   string
	Bucket   string
	Endpoint string
}

// LoggingConfig holds logging service configuration.
type LoggingConfig struct {
	Address   string
	ServiceID string
}

// UploadConfig holds upload-specific configuration.
type UploadConfig struct {
	MaxFileSize    int64
	MaxChunkedSize int64
	ChunkSize      int64
	AllowedTypes   []string
}

// Load loads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{}

	// Server config
	cfg.Server.Port = getEnvInt("SERVER_PORT", 8080)
	cfg.Server.ReadTimeout = getEnvDuration("SERVER_READ_TIMEOUT", 30*time.Second)
	cfg.Server.WriteTimeout = getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second)
	cfg.Server.ShutdownTimeout = getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 30*time.Second)

	// Database config (required)
	cfg.Database.Host = getEnvRequired("DATABASE_HOST")
	cfg.Database.Port = getEnvInt("DATABASE_PORT", 5432)
	cfg.Database.User = getEnvRequired("DATABASE_USER")
	cfg.Database.Password = getEnvRequired("DATABASE_PASSWORD")
	cfg.Database.Database = getEnvRequired("DATABASE_NAME")
	cfg.Database.MaxOpenConns = getEnvInt("DATABASE_MAX_OPEN_CONNS", 25)
	cfg.Database.MaxIdleConns = getEnvInt("DATABASE_MAX_IDLE_CONNS", 5)

	// Cache config (required)
	cfg.Cache.Address = getEnvRequired("CACHE_ADDRESS")
	cfg.Cache.Namespace = getEnv("CACHE_NAMESPACE", "file-upload")

	// Storage config (required)
	cfg.Storage.Region = getEnvRequired("AWS_REGION")
	cfg.Storage.Bucket = getEnvRequired("S3_BUCKET")
	cfg.Storage.Endpoint = getEnv("S3_ENDPOINT", "")

	// Logging config
	cfg.Logging.Address = getEnv("LOGGING_ADDRESS", "localhost:5001")
	cfg.Logging.ServiceID = getEnv("SERVICE_ID", "file-upload")

	// Upload config
	cfg.Upload.MaxFileSize = getEnvInt64("MAX_FILE_SIZE", 100*1024*1024)
	cfg.Upload.MaxChunkedSize = getEnvInt64("MAX_CHUNKED_SIZE", 5*1024*1024*1024)
	cfg.Upload.ChunkSize = getEnvInt64("CHUNK_SIZE", 5*1024*1024)
	cfg.Upload.AllowedTypes = getEnvSlice("ALLOWED_TYPES", []string{
		"image/jpeg", "image/png", "image/gif", "application/pdf",
	})

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Database.Host == "" {
		return fmt.Errorf("DATABASE_HOST is required")
	}
	if c.Database.User == "" {
		return fmt.Errorf("DATABASE_USER is required")
	}
	if c.Database.Password == "" {
		return fmt.Errorf("DATABASE_PASSWORD is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("DATABASE_NAME is required")
	}
	if c.Cache.Address == "" {
		return fmt.Errorf("CACHE_ADDRESS is required")
	}
	if c.Storage.Region == "" {
		return fmt.Errorf("AWS_REGION is required")
	}
	if c.Storage.Bucket == "" {
		return fmt.Errorf("S3_BUCKET is required")
	}
	if c.Upload.MaxFileSize <= 0 {
		return fmt.Errorf("MAX_FILE_SIZE must be positive")
	}
	if c.Upload.ChunkSize <= 0 {
		return fmt.Errorf("CHUNK_SIZE must be positive")
	}
	return nil
}

// DatabaseDSN returns the database connection string.
func (c *Config) DatabaseDSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		c.Database.Host, c.Database.Port, c.Database.User, c.Database.Password, c.Database.Database)
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvRequired(key string) string {
	return os.Getenv(key)
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.Atoi(value); err == nil {
			return i
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			return i
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

func getEnvSlice(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		var result []string
		current := ""
		for _, c := range value {
			if c == ',' {
				if current != "" {
					result = append(result, current)
				}
				current = ""
			} else {
				current += string(c)
			}
		}
		if current != "" {
			result = append(result, current)
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}
