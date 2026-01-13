// Package loggingclient provides a gRPC client for the centralized logging-service.
package loggingclient

import "time"

// Config holds logging client configuration.
type Config struct {
	// Address is the logging-service gRPC address.
	Address string
	// ServiceID identifies this service in logs.
	ServiceID string
	// BatchSize is the max entries before flush (default: 100).
	BatchSize int
	// FlushInterval is the max time before flush (default: 5s).
	FlushInterval time.Duration
	// BufferSize is the max buffer size (default: 10000).
	BufferSize int
	// Enabled controls whether logging to service is enabled.
	Enabled bool
	// CircuitBreakerThreshold is failures before opening circuit.
	CircuitBreakerThreshold int
	// CircuitBreakerTimeout is time before half-open.
	CircuitBreakerTimeout time.Duration
}

// DefaultConfig returns the default logging client configuration.
func DefaultConfig() Config {
	return Config{
		Address:                 "localhost:50052",
		ServiceID:               "cache-service",
		BatchSize:               100,
		FlushInterval:           5 * time.Second,
		BufferSize:              10000,
		Enabled:                 true,
		CircuitBreakerThreshold: 5,
		CircuitBreakerTimeout:   30 * time.Second,
	}
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.Enabled && c.Address == "" {
		return ErrInvalidConfig("address is required when enabled")
	}
	if c.ServiceID == "" {
		c.ServiceID = "unknown-service"
	}
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = 5 * time.Second
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 10000
	}
	return nil
}

// ConfigError represents a configuration error.
type ConfigError struct {
	Message string
}

func (e *ConfigError) Error() string {
	return "loggingclient config error: " + e.Message
}

// ErrInvalidConfig creates a new configuration error.
func ErrInvalidConfig(msg string) error {
	return &ConfigError{Message: msg}
}
