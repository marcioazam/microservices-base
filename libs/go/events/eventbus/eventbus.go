// Package eventbus provides a generic typed event bus.
package eventbus

import "sync"

// Subscription represents a subscription to events.
type Subscription struct {
	id       int
	unsubFn  func()
}

// Unsubscribe removes the subscription.
func (s *Subscription) Unsubscribe() {
	if s.unsubFn != nil {
		s.unsubFn()
	}
}

// EventBus is a generic typed event bus.
type EventBus[T any] struct {
	mu          sync.RWMutex
	handlers    map[int]func(T)
	nextID      int
	closed      bool
}

// New creates a new EventBus.
func New[T any]() *EventBus[T] {
	return &EventBus[T]{
		handlers: make(map[int]func(T)),
	}
}

// Subscribe adds a handler for events.
func (eb *EventBus[T]) Subscribe(handler func(T)) *Subscription {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return &Subscription{}
	}

	id := eb.nextID
	eb.nextID++
	eb.handlers[id] = handler

	return &Subscription{
		id: id,
		unsubFn: func() {
			eb.mu.Lock()
			defer eb.mu.Unlock()
			delete(eb.handlers, id)
		},
	}
}

// SubscribeFiltered adds a handler that only receives matching events.
func (eb *EventBus[T]) SubscribeFiltered(predicate func(T) bool, handler func(T)) *Subscription {
	return eb.Subscribe(func(event T) {
		if predicate(event) {
			handler(event)
		}
	})
}

// Publish sends an event to all subscribers synchronously.
func (eb *EventBus[T]) Publish(event T) {
	eb.mu.RLock()
	handlers := make([]func(T), 0, len(eb.handlers))
	for _, h := range eb.handlers {
		handlers = append(handlers, h)
	}
	eb.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
}

// PublishAsync sends an event to all subscribers asynchronously.
func (eb *EventBus[T]) PublishAsync(event T) {
	eb.mu.RLock()
	handlers := make([]func(T), 0, len(eb.handlers))
	for _, h := range eb.handlers {
		handlers = append(handlers, h)
	}
	eb.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}
}

// PublishAsyncWait sends an event asynchronously and waits for all handlers.
func (eb *EventBus[T]) PublishAsyncWait(event T) {
	eb.mu.RLock()
	handlers := make([]func(T), 0, len(eb.handlers))
	for _, h := range eb.handlers {
		handlers = append(handlers, h)
	}
	eb.mu.RUnlock()

	var wg sync.WaitGroup
	wg.Add(len(handlers))
	for _, h := range handlers {
		go func(handler func(T)) {
			defer wg.Done()
			handler(event)
		}(h)
	}
	wg.Wait()
}

// SubscriberCount returns the number of subscribers.
func (eb *EventBus[T]) SubscriberCount() int {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return len(eb.handlers)
}

// Close closes the event bus and removes all subscribers.
func (eb *EventBus[T]) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.closed = true
	eb.handlers = make(map[int]func(T))
}

// IsClosed returns true if the event bus is closed.
func (eb *EventBus[T]) IsClosed() bool {
	eb.mu.RLock()
	defer eb.mu.RUnlock()
	return eb.closed
}

// SubscribeOnce adds a handler that is called only once.
func (eb *EventBus[T]) SubscribeOnce(handler func(T)) *Subscription {
	var sub *Subscription
	var once sync.Once
	sub = eb.Subscribe(func(event T) {
		once.Do(func() {
			handler(event)
			sub.Unsubscribe()
		})
	})
	return sub
}

// SubscribeChannel returns a channel that receives events.
func (eb *EventBus[T]) SubscribeChannel(bufferSize int) (<-chan T, *Subscription) {
	ch := make(chan T, bufferSize)
	sub := eb.Subscribe(func(event T) {
		select {
		case ch <- event:
		default:
			// Channel full, drop event
		}
	})
	return ch, sub
}
