// Package retry implements retry logic with exponential backoff and jitter.
package retry

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"math"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

// Handler implements the RetryHandler interface.
type Handler struct {
	config       domain.RetryConfig
	serviceName  string
	eventBuilder *domain.EventBuilder
}

// Config holds retry handler creation options.
type Config struct {
	ServiceName  string
	Config       domain.RetryConfig
	EventBuilder *domain.EventBuilder
}

// New creates a new retry handler.
func New(cfg Config) *Handler {
	return &Handler{
		config:       cfg.Config,
		serviceName:  cfg.ServiceName,
		eventBuilder: cfg.EventBuilder,
	}
}

// Execute runs operation with retry policy.
func (h *Handler) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt < h.config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't wait after the last attempt
		if attempt < h.config.MaxAttempts-1 {
			delay := h.CalculateDelay(attempt)
			h.emitRetryEvent(attempt+1, delay, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return domain.NewRetryExhaustedError(h.serviceName, h.config.MaxAttempts, lastErr)
}

// ExecuteWithCircuitBreaker runs operation with retry and circuit breaker.
func (h *Handler) ExecuteWithCircuitBreaker(ctx context.Context, cb domain.CircuitBreaker, operation func() error) error {
	// Check circuit state first
	if cb.GetState() == domain.StateOpen {
		return domain.NewCircuitOpenError(h.serviceName)
	}

	var lastErr error

	for attempt := 0; attempt < h.config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check circuit before each attempt
		if cb.GetState() == domain.StateOpen {
			return domain.NewCircuitOpenError(h.serviceName)
		}

		err := operation()
		if err == nil {
			cb.RecordSuccess()
			return nil
		}

		lastErr = err
		cb.RecordFailure()

		// Check if circuit opened after failure
		if cb.GetState() == domain.StateOpen {
			return domain.NewCircuitOpenError(h.serviceName)
		}

		// Don't wait after the last attempt
		if attempt < h.config.MaxAttempts-1 {
			delay := h.CalculateDelay(attempt)
			h.emitRetryEvent(attempt+1, delay, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
	}

	return domain.NewRetryExhaustedError(h.serviceName, h.config.MaxAttempts, lastErr)
}

// CalculateDelay returns next retry delay for given attempt.
func (h *Handler) CalculateDelay(attempt int) time.Duration {
	// Calculate base delay with exponential backoff
	baseDelay := float64(h.config.BaseDelay) * math.Pow(h.config.Multiplier, float64(attempt))

	// Cap at max delay
	if baseDelay > float64(h.config.MaxDelay) {
		baseDelay = float64(h.config.MaxDelay)
	}

	// Apply jitter using crypto/rand for security
	jitterRange := baseDelay * h.config.JitterPercent
	jitter := (cryptoRandFloat64()*2 - 1) * jitterRange // Random value in [-jitterRange, +jitterRange]

	finalDelay := baseDelay + jitter

	// Ensure non-negative
	if finalDelay < 0 {
		finalDelay = 0
	}

	return time.Duration(finalDelay)
}

// cryptoRandFloat64 returns a cryptographically random float64 in [0, 1).
func cryptoRandFloat64() float64 {
	var b [8]byte
	_, err := rand.Read(b[:])
	if err != nil {
		// Fallback to time-based entropy (should never happen)
		return float64(time.Now().UnixNano()%1000) / 1000.0
	}
	// Convert to uint64 and normalize to [0, 1)
	n := binary.BigEndian.Uint64(b[:])
	return float64(n) / float64(^uint64(0))
}

// emitRetryEvent emits a retry attempt event using EventBuilder.
func (h *Handler) emitRetryEvent(attempt int, delay time.Duration, err error) {
	if h.eventBuilder == nil {
		return
	}

	h.eventBuilder.Emit(domain.EventRetryAttempt, map[string]any{
		"attempt":      attempt,
		"max_attempts": h.config.MaxAttempts,
		"delay":        delay.String(),
		"error":        err.Error(),
	})
}
