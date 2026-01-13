// Package broker provides message broker implementations for cache invalidation.
package broker

import (
	"context"
	"encoding/json"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/authcorp/libs/go/src/fault"
)

// Broker defines the message broker interface.
type Broker interface {
	// Subscribe registers a handler for invalidation events.
	Subscribe(ctx context.Context, topic string, handler cache.InvalidationHandler) error

	// Publish sends an invalidation event.
	Publish(ctx context.Context, topic string, event cache.InvalidationEvent) error

	// Close closes the broker connection.
	Close() error

	// Healthy returns whether the broker is healthy.
	Healthy() bool
}

// Config holds broker configuration.
type Config struct {
	Type    string // "rabbitmq" or "kafka"
	URL     string
	Topic   string
	GroupID string
}

// DefaultRetryConfig returns the default retry configuration using lib fault.
func DefaultRetryConfig() fault.RetryConfig {
	return fault.NewRetryConfig(
		fault.WithMaxAttempts(5),
		fault.WithInitialInterval(time.Second),
		fault.WithMaxInterval(30*time.Second),
		fault.WithMultiplier(2.0),
		fault.WithJitterStrategy(fault.FullJitter),
	)
}

// EncodeEvent encodes an invalidation event to JSON.
func EncodeEvent(event cache.InvalidationEvent) ([]byte, error) {
	return json.Marshal(event)
}

// DecodeEvent decodes an invalidation event from JSON.
func DecodeEvent(data []byte) (cache.InvalidationEvent, error) {
	var event cache.InvalidationEvent
	err := json.Unmarshal(data, &event)
	return event, err
}

// NoOpBroker is a no-operation broker for when messaging is disabled.
type NoOpBroker struct{}

// NewNoOpBroker creates a new no-op broker.
func NewNoOpBroker() *NoOpBroker {
	return &NoOpBroker{}
}

// Subscribe does nothing.
func (b *NoOpBroker) Subscribe(ctx context.Context, topic string, handler cache.InvalidationHandler) error {
	return nil
}

// Publish does nothing.
func (b *NoOpBroker) Publish(ctx context.Context, topic string, event cache.InvalidationEvent) error {
	return nil
}

// Close does nothing.
func (b *NoOpBroker) Close() error {
	return nil
}

// Healthy always returns true.
func (b *NoOpBroker) Healthy() bool {
	return true
}
