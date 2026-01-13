package broker

import (
	"context"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/auth-platform/cache-service/internal/loggingclient"
)

// InvalidationService handles cache invalidation events.
type InvalidationService struct {
	broker       Broker
	cacheService cache.Service
	topic        string
	logger       *loggingclient.Client
}

// NewInvalidationService creates a new invalidation service.
func NewInvalidationService(broker Broker, cacheService cache.Service, topic string, logger *loggingclient.Client) *InvalidationService {
	return &InvalidationService{
		broker:       broker,
		cacheService: cacheService,
		topic:        topic,
		logger:       logger,
	}
}

// Start starts listening for invalidation events.
func (s *InvalidationService) Start(ctx context.Context) error {
	return s.broker.Subscribe(ctx, s.topic, s.handleInvalidation)
}

// handleInvalidation processes an invalidation event.
func (s *InvalidationService) handleInvalidation(event cache.InvalidationEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch event.Action {
	case "delete":
		for _, key := range event.Keys {
			if _, err := s.cacheService.Delete(ctx, event.Namespace, key); err != nil {
				s.logger.Error(ctx, "failed to delete key",
					loggingclient.String("namespace", event.Namespace),
					loggingclient.String("key", key),
					loggingclient.Error(err))
			}
		}
	case "update":
		// For update, we just delete to force a cache miss and refresh
		for _, key := range event.Keys {
			if _, err := s.cacheService.Delete(ctx, event.Namespace, key); err != nil {
				s.logger.Error(ctx, "failed to invalidate key",
					loggingclient.String("namespace", event.Namespace),
					loggingclient.String("key", key),
					loggingclient.Error(err))
			}
		}
	}

	return nil
}

// PublishInvalidation publishes an invalidation event.
func (s *InvalidationService) PublishInvalidation(ctx context.Context, namespace string, keys []string, action string) error {
	event := cache.InvalidationEvent{
		Namespace: namespace,
		Keys:      keys,
		Action:    action,
		Timestamp: time.Now().Unix(),
	}

	return s.broker.Publish(ctx, s.topic, event)
}

// Close closes the invalidation service.
func (s *InvalidationService) Close() error {
	return s.broker.Close()
}

// InvalidationHandler wraps cache service for handling invalidation events.
type InvalidationHandler struct {
	cacheService cache.Service
	logger       *loggingclient.Client
}

// NewInvalidationHandler creates a new invalidation handler.
func NewInvalidationHandler(cacheService cache.Service, logger *loggingclient.Client) *InvalidationHandler {
	return &InvalidationHandler{
		cacheService: cacheService,
		logger:       logger,
	}
}

// Handle processes an invalidation event.
func (h *InvalidationHandler) Handle(event cache.InvalidationEvent) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch event.Action {
	case "delete":
		for _, key := range event.Keys {
			if _, err := h.cacheService.Delete(ctx, event.Namespace, key); err != nil {
				h.logger.Error(ctx, "failed to delete key",
					loggingclient.String("namespace", event.Namespace),
					loggingclient.String("key", key),
					loggingclient.Error(err))
			}
		}
	case "update":
		for _, key := range event.Keys {
			if _, err := h.cacheService.Delete(ctx, event.Namespace, key); err != nil {
				h.logger.Error(ctx, "failed to invalidate key",
					loggingclient.String("namespace", event.Namespace),
					loggingclient.String("key", key),
					loggingclient.Error(err))
			}
		}
	}

	return nil
}
