package broker

import (
	"context"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/authcorp/libs/go/src/fault"
)

// RabbitMQBroker implements the Broker interface using RabbitMQ.
type RabbitMQBroker struct {
	mu          sync.RWMutex
	conn        *amqp.Connection
	channel     *amqp.Channel
	url         string
	retryConfig fault.RetryConfig
	healthy     bool
	closed      bool
	notifyClose chan *amqp.Error
}

// NewRabbitMQBroker creates a new RabbitMQ broker with auto-recovery.
func NewRabbitMQBroker(url string, retryConfig fault.RetryConfig) (*RabbitMQBroker, error) {
	b := &RabbitMQBroker{
		url:         url,
		retryConfig: retryConfig,
		notifyClose: make(chan *amqp.Error),
	}

	if err := b.connect(); err != nil {
		return nil, err
	}

	// Start connection recovery goroutine
	go b.handleReconnect()

	return b, nil
}

func (b *RabbitMQBroker) connect() error {
	conn, err := amqp.Dial(b.url)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to connect to RabbitMQ", err)
	}

	channel, err := conn.Channel()
	if err != nil {
		_ = conn.Close() // Error intentionally ignored - already handling channel error
		return cache.WrapError(cache.ErrBrokerDown, "failed to open channel", err)
	}

	b.mu.Lock()
	b.conn = conn
	b.channel = channel
	b.healthy = true
	b.notifyClose = make(chan *amqp.Error)
	b.conn.NotifyClose(b.notifyClose)
	b.mu.Unlock()

	return nil
}

func (b *RabbitMQBroker) handleReconnect() {
	for {
		select {
		case err := <-b.notifyClose:
			if err == nil {
				return // Connection closed normally
			}

			b.mu.Lock()
			b.healthy = false
			b.mu.Unlock()

			// Attempt reconnection using lib retry
			ctx := context.Background()
			_ = fault.Retry(ctx, b.retryConfig, func(ctx context.Context) error {
				b.mu.RLock()
				closed := b.closed
				b.mu.RUnlock()

				if closed {
					return nil // Stop retrying if closed
				}

				return b.connect()
			})
		}
	}
}

// Subscribe registers a handler for invalidation events.
func (b *RabbitMQBroker) Subscribe(ctx context.Context, topic string, handler cache.InvalidationHandler) error {
	b.mu.RLock()
	channel := b.channel
	b.mu.RUnlock()

	if channel == nil {
		return cache.ErrBrokerDown
	}

	// Declare exchange
	err := channel.ExchangeDeclare(
		topic,    // name
		"fanout", // type
		true,     // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to declare exchange", err)
	}

	// Declare queue
	queue, err := channel.QueueDeclare(
		"",    // name (auto-generated)
		false, // durable
		true,  // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to declare queue", err)
	}

	// Bind queue to exchange
	err = channel.QueueBind(
		queue.Name, // queue name
		"",         // routing key
		topic,      // exchange
		false,      // no-wait
		nil,        // arguments
	)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to bind queue", err)
	}

	// Start consuming
	msgs, err := channel.Consume(
		queue.Name, // queue
		"",         // consumer
		true,       // auto-ack
		false,      // exclusive
		false,      // no-local
		false,      // no-wait
		nil,        // args
	)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to start consuming", err)
	}

	// Process messages in goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg, ok := <-msgs:
				if !ok {
					return
				}
				event, err := DecodeEvent(msg.Body)
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
func (b *RabbitMQBroker) Publish(ctx context.Context, topic string, event cache.InvalidationEvent) error {
	b.mu.RLock()
	channel := b.channel
	b.mu.RUnlock()

	if channel == nil {
		return cache.ErrBrokerDown
	}

	body, err := EncodeEvent(event)
	if err != nil {
		return err
	}

	err = channel.PublishWithContext(ctx,
		topic, // exchange
		"",    // routing key
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		return cache.WrapError(cache.ErrBrokerDown, "failed to publish message", err)
	}

	return nil
}

// Close closes the broker connection.
func (b *RabbitMQBroker) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.closed = true
	b.healthy = false

	var errs []error
	if b.channel != nil {
		if err := b.channel.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if b.conn != nil {
		if err := b.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Healthy returns whether the broker is healthy.
func (b *RabbitMQBroker) Healthy() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.healthy && !b.closed
}
