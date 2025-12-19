// Package testutil provides generators for property-based testing.
package testutil

import (
	"math/rand"
	"time"
)

// Generator is a function that generates random values.
type Generator[T any] func(rng *rand.Rand) T

// NewRand creates a new random number generator.
func NewRand() *rand.Rand {
	return rand.New(rand.NewSource(time.Now().UnixNano()))
}

// NewSeededRand creates a seeded random number generator.
func NewSeededRand(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// IntGen generates random integers in range [min, max].
func IntGen(min, max int) Generator[int] {
	return func(rng *rand.Rand) int {
		if min >= max {
			return min
		}
		return min + rng.Intn(max-min+1)
	}
}

// Int64Gen generates random int64 in range [min, max].
func Int64Gen(min, max int64) Generator[int64] {
	return func(rng *rand.Rand) int64 {
		if min >= max {
			return min
		}
		return min + rng.Int63n(max-min+1)
	}
}

// Float64Gen generates random float64 in range [min, max).
func Float64Gen(min, max float64) Generator[float64] {
	return func(rng *rand.Rand) float64 {
		return min + rng.Float64()*(max-min)
	}
}

// BoolGen generates random booleans.
func BoolGen() Generator[bool] {
	return func(rng *rand.Rand) bool {
		return rng.Intn(2) == 1
	}
}

// StringGen generates random strings of given length.
func StringGen(length int) Generator[string] {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return func(rng *rand.Rand) string {
		b := make([]byte, length)
		for i := range b {
			b[i] = charset[rng.Intn(len(charset))]
		}
		return string(b)
	}
}

// StringRangeGen generates random strings with length in range [minLen, maxLen].
func StringRangeGen(minLen, maxLen int) Generator[string] {
	return func(rng *rand.Rand) string {
		length := minLen
		if maxLen > minLen {
			length = minLen + rng.Intn(maxLen-minLen+1)
		}
		return StringGen(length)(rng)
	}
}

// SliceGen generates slices of random length with elements from elemGen.
func SliceGen[T any](minLen, maxLen int, elemGen Generator[T]) Generator[[]T] {
	return func(rng *rand.Rand) []T {
		length := minLen
		if maxLen > minLen {
			length = minLen + rng.Intn(maxLen-minLen+1)
		}
		result := make([]T, length)
		for i := range result {
			result[i] = elemGen(rng)
		}
		return result
	}
}

// MapGen generates maps with random keys and values.
func MapGen[K comparable, V any](minSize, maxSize int, keyGen Generator[K], valGen Generator[V]) Generator[map[K]V] {
	return func(rng *rand.Rand) map[K]V {
		size := minSize
		if maxSize > minSize {
			size = minSize + rng.Intn(maxSize-minSize+1)
		}
		result := make(map[K]V, size)
		for i := 0; i < size; i++ {
			result[keyGen(rng)] = valGen(rng)
		}
		return result
	}
}

// OneOf generates a random element from the given choices.
func OneOf[T any](choices ...T) Generator[T] {
	return func(rng *rand.Rand) T {
		return choices[rng.Intn(len(choices))]
	}
}

// DurationGen generates random durations in range [min, max].
func DurationGen(min, max time.Duration) Generator[time.Duration] {
	return func(rng *rand.Rand) time.Duration {
		if min >= max {
			return min
		}
		return min + time.Duration(rng.Int63n(int64(max-min)))
	}
}

// TimeGen generates random times in range [min, max].
func TimeGen(min, max time.Time) Generator[time.Time] {
	return func(rng *rand.Rand) time.Time {
		minUnix := min.Unix()
		maxUnix := max.Unix()
		if minUnix >= maxUnix {
			return min
		}
		return time.Unix(minUnix+rng.Int63n(maxUnix-minUnix), 0)
	}
}

// HealthStatus represents health status values.
type HealthStatus int

const (
	HealthStatusHealthy HealthStatus = iota
	HealthStatusDegraded
	HealthStatusUnhealthy
)

// HealthStatusGen generates random health statuses.
func HealthStatusGen() Generator[HealthStatus] {
	return OneOf(HealthStatusHealthy, HealthStatusDegraded, HealthStatusUnhealthy)
}

// CircuitBreakerConfig represents circuit breaker configuration.
type CircuitBreakerConfig struct {
	FailureThreshold int
	SuccessThreshold int
	Timeout          time.Duration
}

// CircuitBreakerConfigGen generates valid circuit breaker configurations.
func CircuitBreakerConfigGen() Generator[CircuitBreakerConfig] {
	return func(rng *rand.Rand) CircuitBreakerConfig {
		return CircuitBreakerConfig{
			FailureThreshold: IntGen(1, 10)(rng),
			SuccessThreshold: IntGen(1, 5)(rng),
			Timeout:          DurationGen(time.Second, time.Minute)(rng),
		}
	}
}

// RetryConfig represents retry configuration.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	Multiplier  float64
	Jitter      float64
}

// RetryConfigGen generates valid retry configurations.
func RetryConfigGen() Generator[RetryConfig] {
	return func(rng *rand.Rand) RetryConfig {
		baseDelay := DurationGen(10*time.Millisecond, time.Second)(rng)
		maxDelay := baseDelay + DurationGen(time.Second, 30*time.Second)(rng)
		return RetryConfig{
			MaxAttempts: IntGen(1, 10)(rng),
			BaseDelay:   baseDelay,
			MaxDelay:    maxDelay,
			Multiplier:  Float64Gen(1.5, 3.0)(rng),
			Jitter:      Float64Gen(0.0, 0.5)(rng),
		}
	}
}

// RateLimitConfig represents rate limit configuration.
type RateLimitConfig struct {
	Algorithm string
	Limit     int
	Window    time.Duration
	BurstSize int
}

// RateLimitConfigGen generates valid rate limit configurations.
func RateLimitConfigGen() Generator[RateLimitConfig] {
	return func(rng *rand.Rand) RateLimitConfig {
		limit := IntGen(1, 1000)(rng)
		return RateLimitConfig{
			Algorithm: OneOf("token_bucket", "sliding_window", "fixed_window")(rng),
			Limit:     limit,
			Window:    DurationGen(time.Second, time.Minute)(rng),
			BurstSize: IntGen(1, limit*2)(rng),
		}
	}
}

// BulkheadConfig represents bulkhead configuration.
type BulkheadConfig struct {
	MaxConcurrent int
	MaxQueue      int
	QueueTimeout  time.Duration
}

// BulkheadConfigGen generates valid bulkhead configurations.
func BulkheadConfigGen() Generator[BulkheadConfig] {
	return func(rng *rand.Rand) BulkheadConfig {
		return BulkheadConfig{
			MaxConcurrent: IntGen(1, 100)(rng),
			MaxQueue:      IntGen(0, 1000)(rng),
			QueueTimeout:  DurationGen(100*time.Millisecond, 10*time.Second)(rng),
		}
	}
}

// TimeoutConfig represents timeout configuration.
type TimeoutConfig struct {
	Default time.Duration
	Max     time.Duration
}

// TimeoutConfigGen generates valid timeout configurations.
func TimeoutConfigGen() Generator[TimeoutConfig] {
	return func(rng *rand.Rand) TimeoutConfig {
		defaultTimeout := DurationGen(100*time.Millisecond, 5*time.Second)(rng)
		maxTimeout := defaultTimeout + DurationGen(time.Second, 30*time.Second)(rng)
		return TimeoutConfig{
			Default: defaultTimeout,
			Max:     maxTimeout,
		}
	}
}

// ResiliencePolicy represents a complete resilience policy.
type ResiliencePolicy struct {
	Name           string
	Version        int
	CircuitBreaker *CircuitBreakerConfig
	Retry          *RetryConfig
	RateLimit      *RateLimitConfig
	Bulkhead       *BulkheadConfig
	Timeout        *TimeoutConfig
}

// ResiliencePolicyGen generates valid complete resilience policies.
func ResiliencePolicyGen() Generator[ResiliencePolicy] {
	return func(rng *rand.Rand) ResiliencePolicy {
		cb := CircuitBreakerConfigGen()(rng)
		retry := RetryConfigGen()(rng)
		rl := RateLimitConfigGen()(rng)
		bh := BulkheadConfigGen()(rng)
		to := TimeoutConfigGen()(rng)

		return ResiliencePolicy{
			Name:           StringGen(10)(rng),
			Version:        IntGen(1, 100)(rng),
			CircuitBreaker: &cb,
			Retry:          &retry,
			RateLimit:      &rl,
			Bulkhead:       &bh,
			Timeout:        &to,
		}
	}
}

// Sample generates n samples using the generator.
func Sample[T any](gen Generator[T], n int) []T {
	rng := NewRand()
	result := make([]T, n)
	for i := range result {
		result[i] = gen(rng)
	}
	return result
}

// SampleSeeded generates n samples using a seeded generator.
func SampleSeeded[T any](gen Generator[T], n int, seed int64) []T {
	rng := NewSeededRand(seed)
	result := make([]T, n)
	for i := range result {
		result[i] = gen(rng)
	}
	return result
}

// Filter creates a generator that only produces values matching the predicate.
func Filter[T any](gen Generator[T], predicate func(T) bool, maxAttempts int) Generator[T] {
	return func(rng *rand.Rand) T {
		for i := 0; i < maxAttempts; i++ {
			v := gen(rng)
			if predicate(v) {
				return v
			}
		}
		// Return last attempt if max attempts reached
		return gen(rng)
	}
}

// Map transforms generator output.
func Map[T, U any](gen Generator[T], fn func(T) U) Generator[U] {
	return func(rng *rand.Rand) U {
		return fn(gen(rng))
	}
}

// FlatMap chains generators.
func FlatMap[T, U any](gen Generator[T], fn func(T) Generator[U]) Generator[U] {
	return func(rng *rand.Rand) U {
		return fn(gen(rng))(rng)
	}
}

// Constant returns a generator that always produces the same value.
func Constant[T any](value T) Generator[T] {
	return func(rng *rand.Rand) T {
		return value
	}
}

// Nullable wraps a generator to sometimes return zero value.
func Nullable[T any](gen Generator[T], nullProbability float64) Generator[T] {
	return func(rng *rand.Rand) T {
		if rng.Float64() < nullProbability {
			var zero T
			return zero
		}
		return gen(rng)
	}
}
