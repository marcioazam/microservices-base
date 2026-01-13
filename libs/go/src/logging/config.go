// Package logging provides a structured logging client for logging-service.
package logging

import "time"

// ClientConfig configures the logging client.
type ClientConfig struct {
	// Address is the logging-service gRPC address.
	Address string

	// ServiceName identifies the service sending logs.
	ServiceName string

	// BufferSize is the max number of logs to buffer before flush.
	BufferSize int

	// FlushInterval is how often to flush buffered logs.
	FlushInterval time.Duration

	// Timeout for logging operations.
	Timeout time.Duration

	// LocalFallback enables stdout logging when remote is unavailable.
	LocalFallback bool

	// MinLevel is the minimum log level to send.
	MinLevel Level
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() ClientConfig {
	return ClientConfig{
		Address:       "localhost:50052",
		ServiceName:   "unknown",
		BufferSize:    100,
		FlushInterval: 5 * time.Second,
		Timeout:       5 * time.Second,
		LocalFallback: true,
		MinLevel:      LevelInfo,
	}
}

// Validate validates the configuration.
func (c *ClientConfig) Validate() error {
	if c.ServiceName == "" {
		c.ServiceName = "unknown"
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 100
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = 5 * time.Second
	}
	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}
	return nil
}
