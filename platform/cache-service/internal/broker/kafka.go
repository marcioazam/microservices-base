package broker

import (
	"context"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/authcorp/libs/go/src/fault"
)

// KafkaBroker implements the Broker interface using Kafka.
type KafkaBroker struct {
	mu          sync.RWMutex
	writer      *kafka.Writer
	brokers     []string
	groupID     string
	retryConfig fault.RetryConfig
	healthy     bool
	closed      bool
}

// NewKafkaBroker creates a new Kafka broker.
func NewKafkaBroker(brokers []string, groupID string, retryConfig fault.RetryConfig) (*KafkaBroker, error) {
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}

	return &KafkaBroker{
		writer:      writer,
		brokers:     brokers,
		groupID:     groupID,
		retryConfig: retryConfig,
		healthy:     true,
	}, nil
}

// Subscribe registers a handler for invalidation events.
func (b *KafkaBroker) Subscribe(ctx context.Context, topic string, handler cache.InvalidationHandler) error {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  b.brokers,
		Topic:    topic,
		GroupID:  b.groupID,
		MinBytes: 1,
		MaxBytes: 10e6,
	})

	// Process messages in goroutine
	go func() {
		defer reader.Close()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				msg, err := reader.ReadMessage(ctx)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					continue
				}

				event, err := DecodeEvent(msg.Value)
				if err != nil {
					continue
				}
				_ = handler(event) // Error logged by handler internally
			}
		}
	}()

	return nil
}

// Publish sends an invalidation event.
func (b *KafkaBroker) Publish(ctx context.Context, topic string, event cache.InvalidationEvent) error {
	b.mu.RLock()
	writer := b.writer
	closed := b.closed
	b.mu.RUnlock()

	if closed || writer == nil {
		return cache.ErrBrokerDown
	}

	body, err := EncodeEvent(event)
	if err != nil {
		return err
	}

	err = writer.WriteMessages(ctx, kafka.Message{
		Topic: topic,
		Value: body,
		Time:  time.Now(),
	})
	if err != nil {
		b.mu.Lock()
		b.healthy = false
		b.mu.Unlock()
		return cache.WrapError(cache.ErrBrokerDown, "failed to publish message", err)
	}

	return nil
}

// Close closes the broker connection.
func (b *KafkaBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	b.healthy = false

	if b.writer != nil {
		return b.writer.Close()
	}
	return nil
}

// Healthy returns whether the broker is healthy.
func (b *KafkaBroker) Healthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.healthy && !b.closed
}
