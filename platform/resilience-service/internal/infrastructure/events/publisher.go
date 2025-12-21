// Package events provides typed event publishing infrastructure.
package events

import (
	"context"
	"log/slog"
	"sync"

	"github.com/auth-platform/platform/resilience-service/internal/domain/valueobjects"
)

// PolicyEventHandler handles policy events.
type PolicyEventHandler func(ctx context.Context, event valueobjects.PolicyEvent)

// DomainEventHandler handles generic domain events.
type DomainEventHandler func(ctx context.Context, event valueobjects.DomainEvent)

// PolicyEventPublisher publishes typed policy events.
type PolicyEventPublisher struct {
	handlers []PolicyEventHandler
	eventCh  chan valueobjects.PolicyEvent
	logger   *slog.Logger
	mu       sync.RWMutex
	closed   bool
}

// NewPolicyEventPublisher creates a new policy event publisher.
func NewPolicyEventPublisher(logger *slog.Logger) *PolicyEventPublisher {
	p := &PolicyEventPublisher{
		handlers: make([]PolicyEventHandler, 0),
		eventCh:  make(chan valueobjects.PolicyEvent, 100),
		logger:   logger,
	}

	// Start event dispatcher
	go p.dispatch()

	return p
}

// Publish publishes a policy event.
func (p *PolicyEventPublisher) Publish(ctx context.Context, event valueobjects.PolicyEvent) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil
	}
	p.mu.RUnlock()

	select {
	case p.eventCh <- event:
		p.logger.DebugContext(ctx, "policy event published",
			slog.String("event_id", event.ID),
			slog.String("event_type", string(event.Type)),
			slog.String("policy_name", event.PolicyName))
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Subscribe registers a handler for policy events.
func (p *PolicyEventPublisher) Subscribe(handler PolicyEventHandler) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.handlers = append(p.handlers, handler)
	p.logger.Info("policy event handler subscribed",
		slog.Int("total_handlers", len(p.handlers)))
}

// dispatch dispatches events to all handlers.
func (p *PolicyEventPublisher) dispatch() {
	for event := range p.eventCh {
		p.mu.RLock()
		handlers := make([]PolicyEventHandler, len(p.handlers))
		copy(handlers, p.handlers)
		p.mu.RUnlock()

		ctx := context.Background()
		for _, handler := range handlers {
			go func(h PolicyEventHandler, e valueobjects.PolicyEvent) {
				defer func() {
					if r := recover(); r != nil {
						p.logger.Error("policy event handler panicked",
							slog.Any("panic", r),
							slog.String("event_id", e.ID))
					}
				}()
				h(ctx, e)
			}(handler, event)
		}
	}
}

// Close closes the publisher.
func (p *PolicyEventPublisher) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.closed {
		p.closed = true
		close(p.eventCh)
		p.logger.Info("policy event publisher closed")
	}
}

// EventBus provides a generic typed event bus.
type EventBus[T any] struct {
	handlers []func(ctx context.Context, event T)
	eventCh  chan T
	logger   *slog.Logger
	mu       sync.RWMutex
	closed   bool
}

// NewEventBus creates a new typed event bus.
func NewEventBus[T any](logger *slog.Logger, bufferSize int) *EventBus[T] {
	bus := &EventBus[T]{
		handlers: make([]func(ctx context.Context, event T), 0),
		eventCh:  make(chan T, bufferSize),
		logger:   logger,
	}

	go bus.dispatch()

	return bus
}

// Publish publishes an event to the bus.
func (b *EventBus[T]) Publish(ctx context.Context, event T) error {
	b.mu.RLock()
	if b.closed {
		b.mu.RUnlock()
		return nil
	}
	b.mu.RUnlock()

	select {
	case b.eventCh <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Subscribe registers a handler for events.
func (b *EventBus[T]) Subscribe(handler func(ctx context.Context, event T)) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.handlers = append(b.handlers, handler)
}

// dispatch dispatches events to handlers.
func (b *EventBus[T]) dispatch() {
	for event := range b.eventCh {
		b.mu.RLock()
		handlers := make([]func(ctx context.Context, event T), len(b.handlers))
		copy(handlers, b.handlers)
		b.mu.RUnlock()

		ctx := context.Background()
		for _, handler := range handlers {
			go func(h func(ctx context.Context, event T), e T) {
				defer func() {
					if r := recover(); r != nil {
						b.logger.Error("event handler panicked", slog.Any("panic", r))
					}
				}()
				h(ctx, e)
			}(handler, event)
		}
	}
}

// Close closes the event bus.
func (b *EventBus[T]) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.closed {
		b.closed = true
		close(b.eventCh)
	}
}

// EventEmitterImpl implements the EventEmitter interface.
type EventEmitterImpl struct {
	policyPublisher *PolicyEventPublisher
	logger          *slog.Logger
}

// NewEventEmitter creates a new event emitter.
func NewEventEmitter(logger *slog.Logger) *EventEmitterImpl {
	return &EventEmitterImpl{
		policyPublisher: NewPolicyEventPublisher(logger),
		logger:          logger,
	}
}

// Emit emits a generic domain event.
func (e *EventEmitterImpl) Emit(ctx context.Context, event valueobjects.DomainEvent) error {
	e.logger.DebugContext(ctx, "domain event emitted",
		slog.String("event_id", event.EventID()),
		slog.String("event_type", event.EventType()))
	return nil
}

// EmitPolicyEvent emits a policy event.
func (e *EventEmitterImpl) EmitPolicyEvent(ctx context.Context, event valueobjects.PolicyEvent) error {
	return e.policyPublisher.Publish(ctx, event)
}

// SubscribePolicyEvents subscribes to policy events.
func (e *EventEmitterImpl) SubscribePolicyEvents(handler PolicyEventHandler) {
	e.policyPublisher.Subscribe(handler)
}

// Close closes the event emitter.
func (e *EventEmitterImpl) Close() {
	e.policyPublisher.Close()
}
