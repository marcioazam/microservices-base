// Package retry implements retry logic with exponential backoff and jitter.
package retry

import (
	"context"
	"math"
	"time"

	liberror "github.com/auth-platform/libs/go/error"
	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/rand"
)

// Handler implements the RetryHandler interface.
type Handler struct {
	config        resilience.RetryConfig
	serviceName   string
	eventEmitter  resilience.EventEmitter
	correlationFn func() string
	randSource    rand.RandSource
}

// Config holds retry handler creation options.
type Config struct {
	ServiceName   string
	Config        resilience.RetryConfig
	EventEmitter  resilience.EventEmitter
	CorrelationFn func() string
	RandSource    rand.RandSource // Optional: defaults to CryptoRandSource
}

// New creates a new retry handler.
func New(cfg Config) *Handler {
	randSource := cfg.RandSource
	if randSource == nil {
		randSource = rand.NewCryptoRandSource()
	}
	return &Handler{
		config:        cfg.Config,
		serviceName:   cfg.ServiceName,
		eventEmitter:  cfg.EventEmitter,
		correlationFn: resilience.EnsureCorrelationFunc(cfg.CorrelationFn),
		randSource:    randSource,
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

	return liberror.NewRetryExhaustedError(h.serviceName, h.config.MaxAttempts, lastErr)
}


// ExecuteWithCircuitBreaker runs operation with retry and circuit breaker.
func (h *Handler) ExecuteWithCircuitBreaker(ctx context.Context, cb resilience.CircuitBreaker, operation func() error) error {
	// Check circuit state first
	if cb.GetState() == resilience.StateOpen {
		return liberror.NewCircuitOpenError(h.serviceName)
	}

	var lastErr error

	for attempt := 0; attempt < h.config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Check circuit before each attempt
		if cb.GetState() == resilience.StateOpen {
			return liberror.NewCircuitOpenError(h.serviceName)
		}

		err := operation()
		if err == nil {
			cb.RecordSuccess()
			return nil
		}

		lastErr = err
		cb.RecordFailure()

		// Check if circuit opened after failure
		if cb.GetState() == resilience.StateOpen {
			return liberror.NewCircuitOpenError(h.serviceName)
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

	return liberror.NewRetryExhaustedError(h.serviceName, h.config.MaxAttempts, lastErr)
}

// CalculateDelay returns next retry delay for given attempt.
func (h *Handler) CalculateDelay(attempt int) time.Duration {
	// Calculate base delay with exponential backoff
	baseDelay := float64(h.config.BaseDelay) * math.Pow(h.config.Multiplier, float64(attempt))

	// Cap at max delay before jitter
	if baseDelay > float64(h.config.MaxDelay) {
		baseDelay = float64(h.config.MaxDelay)
	}

	// Apply jitter
	jitterRange := baseDelay * h.config.JitterPercent
	jitter := (h.randSource.Float64()*2 - 1) * jitterRange // Random value in [-jitterRange, +jitterRange]

	finalDelay := baseDelay + jitter

	// Ensure non-negative
	if finalDelay < 0 {
		finalDelay = 0
	}

	// Cap at max delay after jitter to ensure we never exceed MaxDelay
	if finalDelay > float64(h.config.MaxDelay) {
		finalDelay = float64(h.config.MaxDelay)
	}

	return time.Duration(finalDelay)
}

// emitRetryEvent emits a retry attempt event.
func (h *Handler) emitRetryEvent(attempt int, delay time.Duration, err error) {
	event := resilience.Event{
		ID:            resilience.GenerateEventID(),
		Type:          resilience.EventRetryAttempt,
		ServiceName:   h.serviceName,
		Timestamp:     resilience.NowUTC(),
		CorrelationID: h.correlationFn(),
		Metadata: map[string]any{
			"attempt":      attempt,
			"max_attempts": h.config.MaxAttempts,
			"delay":        delay.String(),
			"error":        err.Error(),
		},
	}

	resilience.EmitEvent(h.eventEmitter, event)
}
