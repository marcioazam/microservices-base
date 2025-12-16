// Package testutil provides test utilities and generators for property-based testing.
package testutil

import (
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
)

// GenCircuitState generates random circuit states.
func GenCircuitState() gopter.Gen {
	return gen.IntRange(0, 2).Map(func(i int) domain.CircuitState {
		return domain.CircuitState(i)
	})
}

// GenCircuitBreakerConfig generates valid circuit breaker configurations.
func GenCircuitBreakerConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(1, 20),       // FailureThreshold
		gen.IntRange(1, 10),       // SuccessThreshold
		gen.IntRange(1000, 60000), // Timeout in ms
		gen.IntRange(1, 5),        // ProbeCount
	).Map(func(vals []interface{}) domain.CircuitBreakerConfig {
		return domain.CircuitBreakerConfig{
			FailureThreshold: vals[0].(int),
			SuccessThreshold: vals[1].(int),
			Timeout:          time.Duration(vals[2].(int)) * time.Millisecond,
			ProbeCount:       vals[3].(int),
		}
	})
}

// GenCircuitBreakerState generates valid circuit breaker states.
func GenCircuitBreakerState() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		GenCircuitState(),
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
		gen.Int64Range(0, time.Now().UnixNano()),
	).Map(func(vals []interface{}) domain.CircuitBreakerState {
		ts := time.Unix(0, vals[4].(int64))
		return domain.CircuitBreakerState{
			ServiceName:     vals[0].(string),
			State:           vals[1].(domain.CircuitState),
			FailureCount:    vals[2].(int),
			SuccessCount:    vals[3].(int),
			LastStateChange: ts,
			Version:         1,
		}
	})
}

// GenRetryConfig generates valid retry configurations.
func GenRetryConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(1, 10),        // MaxAttempts
		gen.IntRange(10, 1000),     // BaseDelay in ms
		gen.IntRange(1000, 60000),  // MaxDelay in ms
		gen.Float64Range(1.0, 5.0), // Multiplier
		gen.Float64Range(0.0, 0.5), // JitterPercent
	).Map(func(vals []interface{}) domain.RetryConfig {
		return domain.RetryConfig{
			MaxAttempts:   vals[0].(int),
			BaseDelay:     time.Duration(vals[1].(int)) * time.Millisecond,
			MaxDelay:      time.Duration(vals[2].(int)) * time.Millisecond,
			Multiplier:    vals[3].(float64),
			JitterPercent: vals[4].(float64),
		}
	})
}

// GenTimeoutConfig generates valid timeout configurations.
func GenTimeoutConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(100, 30000),    // Default in ms
		gen.IntRange(30000, 300000), // Max in ms
	).Map(func(vals []interface{}) domain.TimeoutConfig {
		return domain.TimeoutConfig{
			Default: time.Duration(vals[0].(int)) * time.Millisecond,
			Max:     time.Duration(vals[1].(int)) * time.Millisecond,
		}
	})
}

// GenRateLimitConfig generates valid rate limit configurations.
func GenRateLimitConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.OneConstOf(domain.TokenBucket, domain.SlidingWindow),
		gen.IntRange(1, 10000),    // Limit
		gen.IntRange(1000, 60000), // Window in ms
		gen.IntRange(1, 1000),     // BurstSize
	).Map(func(vals []interface{}) domain.RateLimitConfig {
		return domain.RateLimitConfig{
			Algorithm: vals[0].(domain.RateLimitAlgorithm),
			Limit:     vals[1].(int),
			Window:    time.Duration(vals[2].(int)) * time.Millisecond,
			BurstSize: vals[3].(int),
		}
	})
}

// GenBulkheadConfig generates valid bulkhead configurations.
func GenBulkheadConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.IntRange(1, 500),     // MaxConcurrent
		gen.IntRange(0, 200),     // MaxQueue
		gen.IntRange(100, 30000), // QueueTimeout in ms
	).Map(func(vals []interface{}) domain.BulkheadConfig {
		return domain.BulkheadConfig{
			MaxConcurrent: vals[0].(int),
			MaxQueue:      vals[1].(int),
			QueueTimeout:  time.Duration(vals[2].(int)) * time.Millisecond,
		}
	})
}

// GenHealthStatus generates random health statuses.
func GenHealthStatus() gopter.Gen {
	return gen.OneConstOf(
		domain.HealthHealthy,
		domain.HealthDegraded,
		domain.HealthUnhealthy,
	)
}

// GenResiliencePolicy generates valid resilience policies.
func GenResiliencePolicy() gopter.Gen {
	return gopter.CombineGens(
		gen.AlphaString().SuchThat(func(s string) bool { return len(s) > 0 && len(s) < 50 }),
		gen.IntRange(1, 100),
		GenCircuitBreakerConfig(),
		GenRetryConfig(),
		GenTimeoutConfig(),
		GenRateLimitConfig(),
		GenBulkheadConfig(),
	).Map(func(vals []interface{}) domain.ResiliencePolicy {
		cb := vals[2].(domain.CircuitBreakerConfig)
		retry := vals[3].(domain.RetryConfig)
		timeout := vals[4].(domain.TimeoutConfig)
		rl := vals[5].(domain.RateLimitConfig)
		bh := vals[6].(domain.BulkheadConfig)
		return domain.ResiliencePolicy{
			Name:           vals[0].(string),
			Version:        vals[1].(int),
			CircuitBreaker: &cb,
			Retry:          &retry,
			Timeout:        &timeout,
			RateLimit:      &rl,
			Bulkhead:       &bh,
		}
	})
}

// GenCorrelationID generates valid correlation IDs.
func GenCorrelationID() gopter.Gen {
	return gen.RegexMatch("[a-zA-Z0-9]{8,36}")
}

// GenServiceName generates valid service names.
func GenServiceName() gopter.Gen {
	return gen.AlphaString().SuchThat(func(s string) bool {
		return len(s) > 0 && len(s) < 64
	})
}
