// Package repositories provides type-safe codec for policy serialization.
package repositories

import (
	"encoding/json"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
)

// PolicyCodec provides type-safe JSON serialization for policies.
type PolicyCodec struct{}

// NewPolicyCodec creates a new policy codec.
func NewPolicyCodec() *PolicyCodec {
	return &PolicyCodec{}
}

// policyDTO is the serialization format for Policy.
type policyDTO struct {
	Name           string                `json:"name"`
	Version        int                   `json:"version"`
	CircuitBreaker *circuitBreakerDTO    `json:"circuit_breaker,omitempty"`
	Retry          *retryDTO             `json:"retry,omitempty"`
	Timeout        *timeoutDTO           `json:"timeout,omitempty"`
	RateLimit      *rateLimitDTO         `json:"rate_limit,omitempty"`
	Bulkhead       *bulkheadDTO          `json:"bulkhead,omitempty"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type circuitBreakerDTO struct {
	FailureThreshold int    `json:"failure_threshold"`
	SuccessThreshold int    `json:"success_threshold"`
	Timeout          string `json:"timeout"`
	ProbeCount       int    `json:"probe_count"`
}

type retryDTO struct {
	MaxAttempts   int     `json:"max_attempts"`
	BaseDelay     string  `json:"base_delay"`
	MaxDelay      string  `json:"max_delay"`
	Multiplier    float64 `json:"multiplier"`
	JitterPercent float64 `json:"jitter_percent"`
}

type timeoutDTO struct {
	Default string `json:"default"`
	Max     string `json:"max"`
}

type rateLimitDTO struct {
	Algorithm string `json:"algorithm"`
	Limit     int    `json:"limit"`
	Window    string `json:"window"`
	BurstSize int    `json:"burst_size"`
}

type bulkheadDTO struct {
	MaxConcurrent int    `json:"max_concurrent"`
	MaxQueue      int    `json:"max_queue"`
	QueueTimeout  string `json:"queue_timeout"`
}

// Encode serializes a policy to JSON string.
func (c *PolicyCodec) Encode(policy *entities.Policy) functional.Result[string] {
	dto := c.toDTO(policy)
	bytes, err := json.Marshal(dto)
	if err != nil {
		return functional.Err[string](err)
	}
	return functional.Ok(string(bytes))
}

// Decode deserializes JSON string to a policy.
func (c *PolicyCodec) Decode(data string) functional.Result[*entities.Policy] {
	var dto policyDTO
	if err := json.Unmarshal([]byte(data), &dto); err != nil {
		return functional.Err[*entities.Policy](err)
	}
	return c.fromDTO(&dto)
}

func (c *PolicyCodec) toDTO(policy *entities.Policy) *policyDTO {
	dto := &policyDTO{
		Name:      policy.Name(),
		Version:   policy.Version(),
		CreatedAt: policy.CreatedAt(),
		UpdatedAt: policy.UpdatedAt(),
	}

	if policy.CircuitBreaker().IsSome() {
		cb := policy.CircuitBreaker().Unwrap()
		dto.CircuitBreaker = &circuitBreakerDTO{
			FailureThreshold: cb.FailureThreshold,
			SuccessThreshold: cb.SuccessThreshold,
			Timeout:          cb.Timeout.String(),
			ProbeCount:       cb.ProbeCount,
		}
	}

	if policy.Retry().IsSome() {
		r := policy.Retry().Unwrap()
		dto.Retry = &retryDTO{
			MaxAttempts:   r.MaxAttempts,
			BaseDelay:     r.BaseDelay.String(),
			MaxDelay:      r.MaxDelay.String(),
			Multiplier:    r.Multiplier,
			JitterPercent: r.JitterPercent,
		}
	}

	if policy.Timeout().IsSome() {
		t := policy.Timeout().Unwrap()
		dto.Timeout = &timeoutDTO{
			Default: t.Default.String(),
			Max:     t.Max.String(),
		}
	}

	if policy.RateLimit().IsSome() {
		rl := policy.RateLimit().Unwrap()
		dto.RateLimit = &rateLimitDTO{
			Algorithm: rl.Algorithm,
			Limit:     rl.Limit,
			Window:    rl.Window.String(),
			BurstSize: rl.BurstSize,
		}
	}

	if policy.Bulkhead().IsSome() {
		bh := policy.Bulkhead().Unwrap()
		dto.Bulkhead = &bulkheadDTO{
			MaxConcurrent: bh.MaxConcurrent,
			MaxQueue:      bh.MaxQueue,
			QueueTimeout:  bh.QueueTimeout.String(),
		}
	}

	return dto
}

func (c *PolicyCodec) fromDTO(dto *policyDTO) functional.Result[*entities.Policy] {
	policy, err := entities.NewPolicy(dto.Name)
	if err != nil {
		return functional.Err[*entities.Policy](err)
	}

	if dto.CircuitBreaker != nil {
		cb, err := c.circuitBreakerFromDTO(dto.CircuitBreaker)
		if err != nil {
			return functional.Err[*entities.Policy](err)
		}
		if result := policy.SetCircuitBreaker(cb); result.IsErr() {
			return functional.Err[*entities.Policy](result.UnwrapErr())
		}
	}

	if dto.Retry != nil {
		r, err := c.retryFromDTO(dto.Retry)
		if err != nil {
			return functional.Err[*entities.Policy](err)
		}
		if result := policy.SetRetry(r); result.IsErr() {
			return functional.Err[*entities.Policy](result.UnwrapErr())
		}
	}

	if dto.Timeout != nil {
		t, err := c.timeoutFromDTO(dto.Timeout)
		if err != nil {
			return functional.Err[*entities.Policy](err)
		}
		if result := policy.SetTimeout(t); result.IsErr() {
			return functional.Err[*entities.Policy](result.UnwrapErr())
		}
	}

	if dto.RateLimit != nil {
		rl, err := c.rateLimitFromDTO(dto.RateLimit)
		if err != nil {
			return functional.Err[*entities.Policy](err)
		}
		if result := policy.SetRateLimit(rl); result.IsErr() {
			return functional.Err[*entities.Policy](result.UnwrapErr())
		}
	}

	if dto.Bulkhead != nil {
		bh, err := c.bulkheadFromDTO(dto.Bulkhead)
		if err != nil {
			return functional.Err[*entities.Policy](err)
		}
		if result := policy.SetBulkhead(bh); result.IsErr() {
			return functional.Err[*entities.Policy](result.UnwrapErr())
		}
	}

	return functional.Ok(policy)
}

func (c *PolicyCodec) circuitBreakerFromDTO(dto *circuitBreakerDTO) (*entities.CircuitBreakerConfig, error) {
	timeout, err := time.ParseDuration(dto.Timeout)
	if err != nil {
		return nil, err
	}
	return &entities.CircuitBreakerConfig{
		FailureThreshold: dto.FailureThreshold,
		SuccessThreshold: dto.SuccessThreshold,
		Timeout:          timeout,
		ProbeCount:       dto.ProbeCount,
	}, nil
}

func (c *PolicyCodec) retryFromDTO(dto *retryDTO) (*entities.RetryConfig, error) {
	baseDelay, err := time.ParseDuration(dto.BaseDelay)
	if err != nil {
		return nil, err
	}
	maxDelay, err := time.ParseDuration(dto.MaxDelay)
	if err != nil {
		return nil, err
	}
	return &entities.RetryConfig{
		MaxAttempts:   dto.MaxAttempts,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		Multiplier:    dto.Multiplier,
		JitterPercent: dto.JitterPercent,
	}, nil
}

func (c *PolicyCodec) timeoutFromDTO(dto *timeoutDTO) (*entities.TimeoutConfig, error) {
	defaultTimeout, err := time.ParseDuration(dto.Default)
	if err != nil {
		return nil, err
	}
	maxTimeout, err := time.ParseDuration(dto.Max)
	if err != nil {
		return nil, err
	}
	return &entities.TimeoutConfig{
		Default: defaultTimeout,
		Max:     maxTimeout,
	}, nil
}

func (c *PolicyCodec) rateLimitFromDTO(dto *rateLimitDTO) (*entities.RateLimitConfig, error) {
	window, err := time.ParseDuration(dto.Window)
	if err != nil {
		return nil, err
	}
	return &entities.RateLimitConfig{
		Algorithm: dto.Algorithm,
		Limit:     dto.Limit,
		Window:    window,
		BurstSize: dto.BurstSize,
	}, nil
}

func (c *PolicyCodec) bulkheadFromDTO(dto *bulkheadDTO) (*entities.BulkheadConfig, error) {
	queueTimeout, err := time.ParseDuration(dto.QueueTimeout)
	if err != nil {
		return nil, err
	}
	return &entities.BulkheadConfig{
		MaxConcurrent: dto.MaxConcurrent,
		MaxQueue:      dto.MaxQueue,
		QueueTimeout:  queueTimeout,
	}, nil
}
