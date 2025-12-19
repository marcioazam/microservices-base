package events

import (
	"context"
	"sync"
)

// Event represents a generic event.
type Event interface {
	Type() string
}

// Handler handles events of type E.
type Handler[E Event] func(ctx context.Context, event E) error

// Filter filters events.
type Filter[E Event] func(E) bool

// Subscription represents an event subscription.
type Subscription struct {
	id       string
	cancel   func()
	eventType string
}

// Cancel cancels the subscription.
func (s *Subscription) Cancel() {
	if s.cancel != nil {
		s.cancel()
	}
}

// EventBus provides publish/subscribe functionality.
type EventBus[E Event] struct {
	handlers map[string][]handlerEntry[E]
	mu       sync.RWMutex
	async    bool
	nextID   int
}

type handlerEntry[E Event] struct {
	id      string
	handler Handler[E]
	filter  Filter[E]
}

// NewEventBus creates a new event bus.
func NewEventBus[E Event]() *EventBus[E] {
	return &EventBus[E]{
		handlers: make(map[string][]handlerEntry[E]),
	}
}

// NewAsyncEventBus creates an async event bus.
func NewAsyncEventBus[E Event]() *EventBus[E] {
	return &EventBus[E]{
		handlers: make(map[string][]handlerEntry[E]),
		async:    true,
	}
}

// Subscribe registers a handler for an event type.
func (b *EventBus[E]) Subscribe(eventType string, handler Handler[E]) *Subscription {
	return b.SubscribeWithFilter(eventType, handler, nil)
}

// SubscribeWithFilter registers a handler with a filter.
func (b *EventBus[E]) SubscribeWithFilter(eventType string, handler Handler[E], filter Filter[E]) *Subscription {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	id := string(rune(b.nextID))

	entry := handlerEntry[E]{
		id:      id,
		handler: handler,
		filter:  filter,
	}

	b.handlers[eventType] = append(b.handlers[eventType], entry)

	return &Subscription{
		id:        id,
		eventType: eventType,
		cancel: func() {
			b.unsubscribe(eventType, id)
		},
	}
}

func (b *EventBus[E]) unsubscribe(eventType, id string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	handlers := b.handlers[eventType]
	for i, h := range handlers {
		if h.id == id {
			b.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			return
		}
	}
}

// Publish publishes an event to all subscribers.
func (b *EventBus[E]) Publish(ctx context.Context, event E) error {
	b.mu.RLock()
	handlers := b.handlers[event.Type()]
	b.mu.RUnlock()

	var wg sync.WaitGroup
	errChan := make(chan error, len(handlers))

	for _, h := range handlers {
		if h.filter != nil && !h.filter(event) {
			continue
		}

		if b.async {
			wg.Add(1)
			go func(handler Handler[E]) {
				defer wg.Done()
				if err := handler(ctx, event); err != nil {
					errChan <- err
				}
			}(h.handler)
		} else {
			if err := h.handler(ctx, event); err != nil {
				return err
			}
		}
	}

	if b.async {
		wg.Wait()
		close(errChan)
		for err := range errChan {
			return err
		}
	}

	return nil
}

// PublishAll publishes multiple events.
func (b *EventBus[E]) PublishAll(ctx context.Context, events []E) error {
	for _, event := range events {
		if err := b.Publish(ctx, event); err != nil {
			return err
		}
	}
	return nil
}

// HasSubscribers returns true if event type has subscribers.
func (b *EventBus[E]) HasSubscribers(eventType string) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventType]) > 0
}

// SubscriberCount returns number of subscribers for event type.
func (b *EventBus[E]) SubscriberCount(eventType string) int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.handlers[eventType])
}

// Clear removes all subscriptions.
func (b *EventBus[E]) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers = make(map[string][]handlerEntry[E])
}
