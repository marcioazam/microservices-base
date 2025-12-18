package retry

import (
	"testing"
	"time"

	"github.com/auth-platform/libs/go/resilience"
	"github.com/auth-platform/libs/go/resilience/rand"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// **Feature: resilience-lib-extraction, Property 3: Retry Delay Bounds**
// **Validates: Requirements 1.2**
func TestRetryDelayBounds(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("delay is non-negative and <= MaxDelay", prop.ForAll(
		func(baseDelayMs int, maxDelayMs int, multiplier float64, jitter float64, attempt int) bool {
			if baseDelayMs < 10 {
				baseDelayMs = 10
			}
			if maxDelayMs < baseDelayMs+100 {
				maxDelayMs = baseDelayMs + 100
			}
			if multiplier < 1.0 {
				multiplier = 1.0
			}
			if jitter < 0 {
				jitter = 0
			}
			if jitter > 0.5 {
				jitter = 0.5
			}
			if attempt < 0 {
				attempt = 0
			}
			if attempt > 10 {
				attempt = 10
			}

			h := New(Config{
				ServiceName: "test",
				Config: resilience.RetryConfig{
					MaxAttempts:   5,
					BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
					MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
					Multiplier:    multiplier,
					JitterPercent: jitter,
				},
				RandSource: rand.NewDeterministicRandSource(42),
			})

			delay := h.CalculateDelay(attempt)
			return delay >= 0 && delay <= time.Duration(maxDelayMs)*time.Millisecond
		},
		gen.IntRange(10, 1000),
		gen.IntRange(1000, 60000),
		gen.Float64Range(1.0, 5.0),
		gen.Float64Range(0.0, 0.5),
		gen.IntRange(0, 10),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 4: Retry Exponential Backoff**
// **Validates: Requirements 1.2**
func TestRetryExponentialBackoff(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("base delay increases exponentially (before jitter)", prop.ForAll(
		func(baseDelayMs int, maxDelayMs int, multiplier float64) bool {
			if baseDelayMs < 10 {
				baseDelayMs = 10
			}
			if maxDelayMs < baseDelayMs*10 {
				maxDelayMs = baseDelayMs * 10
			}
			if multiplier < 1.0 {
				multiplier = 1.0
			}
			if multiplier > 5.0 {
				multiplier = 5.0
			}

			// Use fixed rand source at 0.5 to eliminate jitter effect
			h := New(Config{
				ServiceName: "test",
				Config: resilience.RetryConfig{
					MaxAttempts:   5,
					BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
					MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
					Multiplier:    multiplier,
					JitterPercent: 0, // No jitter for this test
				},
				RandSource: rand.NewFixedRandSource(0.5),
			})

			prevDelay := h.CalculateDelay(0)
			for attempt := 1; attempt < 5; attempt++ {
				currDelay := h.CalculateDelay(attempt)
				// Current delay should be >= previous (until capped at MaxDelay)
				if currDelay < prevDelay && currDelay < time.Duration(maxDelayMs)*time.Millisecond {
					return false
				}
				prevDelay = currDelay
			}
			return true
		},
		gen.IntRange(10, 1000),
		gen.IntRange(10000, 60000),
		gen.Float64Range(1.0, 5.0),
	))

	properties.TestingRun(t)
}

// **Feature: resilience-lib-extraction, Property 12: Retry Policy Round-Trip**
// **Validates: Requirements 5.2, 5.3**
func TestRetryPolicyRoundTrip(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	properties.Property("ToDefinition then FromDefinition preserves config", prop.ForAll(
		func(maxAttempts int, baseDelayMs int, maxDelayMs int, multiplier float64, jitter float64) bool {
			if maxAttempts < 1 {
				maxAttempts = 1
			}
			if maxAttempts > 10 {
				maxAttempts = 10
			}
			if baseDelayMs < 10 {
				baseDelayMs = 10
			}
			if maxDelayMs < baseDelayMs+100 {
				maxDelayMs = baseDelayMs + 100
			}
			if multiplier < 1.0 {
				multiplier = 1.0
			}
			if multiplier > 5.0 {
				multiplier = 5.0
			}
			if jitter < 0 {
				jitter = 0
			}
			if jitter > 0.5 {
				jitter = 0.5
			}

			original := &resilience.RetryConfig{
				MaxAttempts:   maxAttempts,
				BaseDelay:     time.Duration(baseDelayMs) * time.Millisecond,
				MaxDelay:      time.Duration(maxDelayMs) * time.Millisecond,
				Multiplier:    multiplier,
				JitterPercent: jitter,
			}

			def := ToDefinition(original)
			restored := FromDefinition(def)

			return original.MaxAttempts == restored.MaxAttempts &&
				original.BaseDelay == restored.BaseDelay &&
				original.MaxDelay == restored.MaxDelay &&
				original.Multiplier == restored.Multiplier &&
				original.JitterPercent == restored.JitterPercent
		},
		gen.IntRange(1, 10),
		gen.IntRange(10, 1000),
		gen.IntRange(1000, 60000),
		gen.Float64Range(1.0, 5.0),
		gen.Float64Range(0.0, 0.5),
	))

	properties.TestingRun(t)
}

func TestValidatePolicy(t *testing.T) {
	tests := []struct {
		name    string
		def     PolicyDefinition
		wantErr bool
	}{
		{
			name: "valid policy",
			def: PolicyDefinition{
				MaxAttempts:   3,
				BaseDelayMs:   100,
				MaxDelayMs:    10000,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			wantErr: false,
		},
		{
			name: "max_attempts too low",
			def: PolicyDefinition{
				MaxAttempts:   0,
				BaseDelayMs:   100,
				MaxDelayMs:    10000,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			wantErr: true,
		},
		{
			name: "base_delay >= max_delay",
			def: PolicyDefinition{
				MaxAttempts:   3,
				BaseDelayMs:   10000,
				MaxDelayMs:    1000,
				Multiplier:    2.0,
				JitterPercent: 0.1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePolicy(tt.def)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePolicy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPrettyPrint(t *testing.T) {
	cfg := &resilience.RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      10 * time.Second,
		Multiplier:    2.0,
		JitterPercent: 0.1,
	}

	output := PrettyPrint(cfg)
	if output == "" {
		t.Error("expected non-empty output")
	}
}
