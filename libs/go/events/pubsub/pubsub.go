// Package pubsub provides a generic pub/sub system with topics.
package pubsub

import (
	"strings"
	"sync"
)

// Subscription represents a subscription to a topic.
type Subscription struct {
	unsubFn func()
}

// Unsubscribe removes the subscription.
func (s *Subscription) Unsubscribe() {
	if s.unsubFn != nil {
		s.unsubFn()
	}
}

// PubSub is a generic pub/sub system with topics.
type PubSub[T any] struct {
	mu       sync.RWMutex
	topics   map[string]map[int]func(T)
	patterns map[int]patternSub[T]
	nextID   int
	closed   bool
}

type patternSub[T any] struct {
	pattern string
	handler func(string, T)
}

// New creates a new PubSub.
func New[T any]() *PubSub[T] {
	return &PubSub[T]{
		topics:   make(map[string]map[int]func(T)),
		patterns: make(map[int]patternSub[T]),
	}
}

// Subscribe adds a handler for a specific topic.
func (ps *PubSub[T]) Subscribe(topic string, handler func(T)) *Subscription {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return &Subscription{}
	}

	if ps.topics[topic] == nil {
		ps.topics[topic] = make(map[int]func(T))
	}

	id := ps.nextID
	ps.nextID++
	ps.topics[topic][id] = handler

	return &Subscription{
		unsubFn: func() {
			ps.mu.Lock()
			defer ps.mu.Unlock()
			delete(ps.topics[topic], id)
			if len(ps.topics[topic]) == 0 {
				delete(ps.topics, topic)
			}
		},
	}
}

// SubscribePattern adds a handler for topics matching a pattern.
// Pattern supports * as wildcard (e.g., "user.*" matches "user.created", "user.deleted").
func (ps *PubSub[T]) SubscribePattern(pattern string, handler func(topic string, event T)) *Subscription {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if ps.closed {
		return &Subscription{}
	}

	id := ps.nextID
	ps.nextID++
	ps.patterns[id] = patternSub[T]{pattern: pattern, handler: handler}

	return &Subscription{
		unsubFn: func() {
			ps.mu.Lock()
			defer ps.mu.Unlock()
			delete(ps.patterns, id)
		},
	}
}

// Publish sends an event to all subscribers of a topic.
func (ps *PubSub[T]) Publish(topic string, event T) {
	ps.mu.RLock()
	handlers := make([]func(T), 0)
	if subs, ok := ps.topics[topic]; ok {
		for _, h := range subs {
			handlers = append(handlers, h)
		}
	}

	patternHandlers := make([]func(string, T), 0)
	for _, p := range ps.patterns {
		if matchPattern(p.pattern, topic) {
			patternHandlers = append(patternHandlers, p.handler)
		}
	}
	ps.mu.RUnlock()

	for _, h := range handlers {
		h(event)
	}
	for _, h := range patternHandlers {
		h(topic, event)
	}
}

// PublishAsync sends an event asynchronously.
func (ps *PubSub[T]) PublishAsync(topic string, event T) {
	ps.mu.RLock()
	handlers := make([]func(T), 0)
	if subs, ok := ps.topics[topic]; ok {
		for _, h := range subs {
			handlers = append(handlers, h)
		}
	}

	patternHandlers := make([]func(string, T), 0)
	for _, p := range ps.patterns {
		if matchPattern(p.pattern, topic) {
			patternHandlers = append(patternHandlers, p.handler)
		}
	}
	ps.mu.RUnlock()

	for _, h := range handlers {
		go h(event)
	}
	for _, h := range patternHandlers {
		go h(topic, event)
	}
}

// Topics returns all active topics.
func (ps *PubSub[T]) Topics() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	topics := make([]string, 0, len(ps.topics))
	for t := range ps.topics {
		topics = append(topics, t)
	}
	return topics
}

// SubscriberCount returns the number of subscribers for a topic.
func (ps *PubSub[T]) SubscriberCount(topic string) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if subs, ok := ps.topics[topic]; ok {
		return len(subs)
	}
	return 0
}

// Close closes the pub/sub and removes all subscribers.
func (ps *PubSub[T]) Close() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.closed = true
	ps.topics = make(map[string]map[int]func(T))
	ps.patterns = make(map[int]patternSub[T])
}

// IsClosed returns true if the pub/sub is closed.
func (ps *PubSub[T]) IsClosed() bool {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	return ps.closed
}

// matchPattern checks if a topic matches a pattern.
// Supports * as wildcard for a single segment.
func matchPattern(pattern, topic string) bool {
	patternParts := strings.Split(pattern, ".")
	topicParts := strings.Split(topic, ".")

	if len(patternParts) != len(topicParts) {
		// Check for trailing wildcard
		if len(patternParts) > 0 && patternParts[len(patternParts)-1] == "*" {
			if len(topicParts) >= len(patternParts)-1 {
				patternParts = patternParts[:len(patternParts)-1]
				topicParts = topicParts[:len(patternParts)]
			}
		} else {
			return false
		}
	}

	for i, p := range patternParts {
		if p != "*" && p != topicParts[i] {
			return false
		}
	}
	return true
}
