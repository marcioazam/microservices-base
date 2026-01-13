// Package testutil provides test utilities and helpers.
package testutil

import (
	"context"
	"testing"
	"time"
)

// RedisContainer provides a Redis instance for testing.
// Returns the connection string and a cleanup function.
func RedisContainer(t *testing.T) (string, func()) {
	t.Helper()

	// For unit tests, return a mock address
	// In integration tests, this would use testcontainers
	return "localhost:6379", func() {}
}

// RabbitMQContainer provides a RabbitMQ instance for testing.
// Returns the connection URL and a cleanup function.
func RabbitMQContainer(t *testing.T) (string, func()) {
	t.Helper()

	// For unit tests, return a mock URL
	// In integration tests, this would use testcontainers
	return "amqp://guest:guest@localhost:5672/", func() {}
}

// WithTimeout creates a context with timeout for tests.
func WithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), timeout)
}

// DefaultTestTimeout is the default timeout for tests.
const DefaultTestTimeout = 30 * time.Second
