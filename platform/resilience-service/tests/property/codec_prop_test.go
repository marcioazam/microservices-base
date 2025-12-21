package property

import (
	"testing"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain/entities"
	"github.com/auth-platform/platform/resilience-service/internal/infrastructure/repositories"
	"pgregory.net/rapid"
)

// TestCodecRoundtripPreservesPolicy validates codec roundtrip.
func TestCodecRoundtripPreservesPolicy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codec := repositories.NewPolicyCodec()

		name := rapid.StringMatching(`[a-z]{5,20}`).Draw(t, "name")
		policy, err := entities.NewPolicy(name)
		if err != nil {
			t.Fatalf("Failed to create policy: %v", err)
		}

		cbConfig := &entities.CircuitBreakerConfig{
			FailureThreshold: rapid.IntRange(1, 100).Draw(t, "failure_threshold"),
			SuccessThreshold: rapid.IntRange(1, 10).Draw(t, "success_threshold"),
			Timeout:          time.Duration(rapid.IntRange(1, 300).Draw(t, "timeout")) * time.Second,
			ProbeCount:       rapid.IntRange(1, 10).Draw(t, "probe_count"),
		}
		policy.SetCircuitBreaker(cbConfig)

		encodeResult := codec.Encode(policy)
		if encodeResult.IsErr() {
			t.Fatalf("Encode failed: %v", encodeResult.UnwrapErr())
		}

		decodeResult := codec.Decode(encodeResult.Unwrap())
		if decodeResult.IsErr() {
			t.Fatalf("Decode failed: %v", decodeResult.UnwrapErr())
		}

		decoded := decodeResult.Unwrap()
		if decoded.Name() != policy.Name() {
			t.Errorf("Name = %s, want %s", decoded.Name(), policy.Name())
		}
	})
}

// TestCodecRoundtripPreservesRetryConfig validates retry config roundtrip.
func TestCodecRoundtripPreservesRetryConfig(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codec := repositories.NewPolicyCodec()

		name := rapid.StringMatching(`[a-z]{5,20}`).Draw(t, "name")
		policy, _ := entities.NewPolicy(name)

		// MaxDelay validation: 1s to 5min (300000ms), BaseDelay: 1ms to 10s (10000ms)
		baseDelayMs := rapid.IntRange(1, 10000).Draw(t, "base_delay_ms")
		// Ensure maxDelayMs is within valid range (1000ms to 300000ms) and > baseDelayMs
		maxDelayMs := rapid.IntRange(max(baseDelayMs+1, 1000), 300000).Draw(t, "max_delay_ms")

		retryConfig := &entities.RetryConfig{
			MaxAttempts:   rapid.IntRange(1, 10).Draw(t, "max_attempts"),
			BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
			MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
			Multiplier:    float64(rapid.IntRange(10, 100).Draw(t, "multiplier")) / 10.0,
			JitterPercent: float64(rapid.IntRange(0, 100).Draw(t, "jitter")) / 100.0,
		}
		policy.SetRetry(retryConfig)

		encodeResult := codec.Encode(policy)
		if encodeResult.IsErr() {
			t.Fatalf("Encode failed: %v", encodeResult.UnwrapErr())
		}

		decodeResult := codec.Decode(encodeResult.Unwrap())
		if decodeResult.IsErr() {
			t.Fatalf("Decode failed: %v", decodeResult.UnwrapErr())
		}

		decoded := decodeResult.Unwrap()
		if !decoded.Retry().IsSome() {
			t.Fatal("Retry should be Some after roundtrip")
		}
	})
}

// TestCodecHandlesNoneOptions validates None options handling.
func TestCodecHandlesNoneOptions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codec := repositories.NewPolicyCodec()

		name := rapid.StringMatching(`[a-z]{5,20}`).Draw(t, "name")
		policy, _ := entities.NewPolicy(name)

		cbConfig := &entities.CircuitBreakerConfig{
			FailureThreshold: 5,
			SuccessThreshold: 3,
			Timeout:          30 * time.Second,
			ProbeCount:       2,
		}
		policy.SetCircuitBreaker(cbConfig)

		encodeResult := codec.Encode(policy)
		if encodeResult.IsErr() {
			t.Fatalf("Encode failed: %v", encodeResult.UnwrapErr())
		}

		decodeResult := codec.Decode(encodeResult.Unwrap())
		if decodeResult.IsErr() {
			t.Fatalf("Decode failed: %v", decodeResult.UnwrapErr())
		}

		decoded := decodeResult.Unwrap()

		if !decoded.CircuitBreaker().IsSome() {
			t.Error("CircuitBreaker should be Some")
		}
		if decoded.Retry().IsSome() {
			t.Error("Retry should be None")
		}
	})
}

// TestCodecDecodeFailsOnInvalidJSON validates invalid JSON handling.
func TestCodecDecodeFailsOnInvalidJSON(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		codec := repositories.NewPolicyCodec()

		invalidJSON := rapid.StringMatching(`[^{}]+`).Draw(t, "invalid_json")

		result := codec.Decode(invalidJSON)
		if result.IsOk() {
			t.Error("Decode should fail on invalid JSON")
		}
	})
}
