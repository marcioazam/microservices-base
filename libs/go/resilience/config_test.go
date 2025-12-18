package resilience

import (
	"testing"
	"time"
)

func TestCircuitBreakerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  CircuitBreakerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultCircuitBreakerConfig(),
			wantErr: false,
		},
		{
			name: "zero failure threshold",
			config: CircuitBreakerConfig{
				FailureThreshold: 0,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero success threshold",
			config: CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 0,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRetryConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  RetryConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultRetryConfig(),
			wantErr: false,
		},
		{
			name: "zero max attempts",
			config: RetryConfig{
				MaxAttempts: 0,
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    10 * time.Second,
				Multiplier:  2.0,
			},
			wantErr: true,
		},
		{
			name: "multiplier less than 1",
			config: RetryConfig{
				MaxAttempts: 3,
				BaseDelay:   100 * time.Millisecond,
				MaxDelay:    10 * time.Second,
				Multiplier:  0.5,
			},
			wantErr: true,
		},
		{
			name: "jitter out of range",
			config: RetryConfig{
				MaxAttempts:   3,
				BaseDelay:     100 * time.Millisecond,
				MaxDelay:      10 * time.Second,
				Multiplier:    2.0,
				JitterPercent: 1.5,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTimeoutConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  TimeoutConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultTimeoutConfig(),
			wantErr: false,
		},
		{
			name: "zero default",
			config: TimeoutConfig{
				Default: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRateLimitConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  RateLimitConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultRateLimitConfig(),
			wantErr: false,
		},
		{
			name: "zero limit",
			config: RateLimitConfig{
				Limit:  0,
				Window: time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero window",
			config: RateLimitConfig{
				Limit:  100,
				Window: 0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBulkheadConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  BulkheadConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultBulkheadConfig(),
			wantErr: false,
		},
		{
			name: "zero max concurrent",
			config: BulkheadConfig{
				MaxConcurrent: 0,
				MaxQueue:      100,
			},
			wantErr: true,
		},
		{
			name: "negative max queue",
			config: BulkheadConfig{
				MaxConcurrent: 10,
				MaxQueue:      -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResiliencePolicyValidate(t *testing.T) {
	tests := []struct {
		name    string
		policy  ResiliencePolicy
		wantErr bool
	}{
		{
			name: "valid policy with all configs",
			policy: ResiliencePolicy{
				Name:           "test-policy",
				CircuitBreaker: &CircuitBreakerConfig{FailureThreshold: 5, SuccessThreshold: 2, Timeout: 30 * time.Second},
				Retry:          &RetryConfig{MaxAttempts: 3, BaseDelay: 100 * time.Millisecond, MaxDelay: 10 * time.Second, Multiplier: 2.0},
				Timeout:        &TimeoutConfig{Default: 30 * time.Second},
				RateLimit:      &RateLimitConfig{Limit: 100, Window: time.Second},
				Bulkhead:       &BulkheadConfig{MaxConcurrent: 10, MaxQueue: 100},
			},
			wantErr: false,
		},
		{
			name: "empty name",
			policy: ResiliencePolicy{
				Name: "",
			},
			wantErr: true,
		},
		{
			name: "invalid circuit breaker",
			policy: ResiliencePolicy{
				Name:           "test",
				CircuitBreaker: &CircuitBreakerConfig{FailureThreshold: 0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCircuitStateString(t *testing.T) {
	tests := []struct {
		state    CircuitState
		expected string
	}{
		{StateClosed, "CLOSED"},
		{StateOpen, "OPEN"},
		{StateHalfOpen, "HALF_OPEN"},
		{CircuitState(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.state.String(); got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %s, want %s", tt.state, got, tt.expected)
		}
	}
}
