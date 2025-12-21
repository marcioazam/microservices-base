package fault_test

import (
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
)

func TestCircuitBreakerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  fault.CircuitBreakerConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  fault.DefaultCircuitBreakerConfig(),
			wantErr: false,
		},
		{
			name: "zero failure threshold",
			config: fault.CircuitBreakerConfig{
				FailureThreshold: 0,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero success threshold",
			config: fault.CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 0,
				Timeout:          30 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero timeout",
			config: fault.CircuitBreakerConfig{
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
		config  fault.RetryConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  fault.DefaultRetryConfig(),
			wantErr: false,
		},
		{
			name: "zero max attempts",
			config: fault.RetryConfig{
				MaxAttempts:     0,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      2.0,
			},
			wantErr: true,
		},
		{
			name: "multiplier less than 1",
			config: fault.RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 100 * time.Millisecond,
				MaxInterval:     10 * time.Second,
				Multiplier:      0.5,
			},
			wantErr: true,
		},
		{
			name: "max interval less than initial",
			config: fault.RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Second,
				MaxInterval:     100 * time.Millisecond,
				Multiplier:      2.0,
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
		config  fault.TimeoutConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  fault.DefaultTimeoutConfig(),
			wantErr: false,
		},
		{
			name: "zero timeout",
			config: fault.TimeoutConfig{
				Timeout: 0,
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
		config  fault.RateLimitConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  fault.DefaultRateLimitConfig(),
			wantErr: false,
		},
		{
			name: "zero rate",
			config: fault.RateLimitConfig{
				Rate:   0,
				Window: time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero window",
			config: fault.RateLimitConfig{
				Rate:   100,
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
		config  fault.BulkheadConfig
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  fault.DefaultBulkheadConfig(),
			wantErr: false,
		},
		{
			name: "zero max concurrent",
			config: fault.BulkheadConfig{
				MaxConcurrent: 0,
				QueueSize:     100,
			},
			wantErr: true,
		},
		{
			name: "negative queue size",
			config: fault.BulkheadConfig{
				MaxConcurrent: 10,
				QueueSize:     -1,
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
