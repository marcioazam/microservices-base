package retry

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/auth-platform/libs/go/resilience"
)

// PolicyDefinition is the serializable retry policy configuration.
type PolicyDefinition struct {
	MaxAttempts   int      `json:"max_attempts"`
	BaseDelayMs   int      `json:"base_delay_ms"`
	MaxDelayMs    int      `json:"max_delay_ms"`
	Multiplier    float64  `json:"multiplier"`
	JitterPercent float64  `json:"jitter_percent"`
	RetryOn       []string `json:"retry_on,omitempty"`
}

// ValidationError represents a policy validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ParsePolicy parses and validates a retry policy from JSON.
func ParsePolicy(data []byte) (*resilience.RetryConfig, error) {
	var def PolicyDefinition
	if err := json.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("parse retry policy: %w", err)
	}

	if err := ValidatePolicy(def); err != nil {
		return nil, err
	}

	return &resilience.RetryConfig{
		MaxAttempts:     def.MaxAttempts,
		BaseDelay:       time.Duration(def.BaseDelayMs) * time.Millisecond,
		MaxDelay:        time.Duration(def.MaxDelayMs) * time.Millisecond,
		Multiplier:      def.Multiplier,
		JitterPercent:   def.JitterPercent,
		RetryableErrors: def.RetryOn,
	}, nil
}

// ValidatePolicy validates a retry policy definition.
func ValidatePolicy(def PolicyDefinition) error {
	if def.MaxAttempts < 1 {
		return ValidationError{Field: "max_attempts", Message: "must be at least 1"}
	}
	if def.MaxAttempts > 10 {
		return ValidationError{Field: "max_attempts", Message: "must not exceed 10"}
	}

	if def.BaseDelayMs < 10 {
		return ValidationError{Field: "base_delay_ms", Message: "must be at least 10ms"}
	}
	if def.BaseDelayMs > 60000 {
		return ValidationError{Field: "base_delay_ms", Message: "must not exceed 60000ms"}
	}

	if def.MaxDelayMs < 100 {
		return ValidationError{Field: "max_delay_ms", Message: "must be at least 100ms"}
	}
	if def.MaxDelayMs > 300000 {
		return ValidationError{Field: "max_delay_ms", Message: "must not exceed 300000ms"}
	}

	if def.BaseDelayMs >= def.MaxDelayMs {
		return ValidationError{Field: "base_delay_ms", Message: "must be less than max_delay_ms"}
	}

	if def.Multiplier < 1.0 {
		return ValidationError{Field: "multiplier", Message: "must be at least 1.0"}
	}
	if def.Multiplier > 5.0 {
		return ValidationError{Field: "multiplier", Message: "must not exceed 5.0"}
	}

	if def.JitterPercent < 0 {
		return ValidationError{Field: "jitter_percent", Message: "must not be negative"}
	}
	if def.JitterPercent > 0.5 {
		return ValidationError{Field: "jitter_percent", Message: "must not exceed 0.5"}
	}

	return nil
}

// MarshalPolicy serializes a retry config to JSON.
func MarshalPolicy(cfg *resilience.RetryConfig) ([]byte, error) {
	def := ToDefinition(cfg)
	return json.Marshal(def)
}

// PrettyPrint returns a human-readable representation of the retry policy.
func PrettyPrint(cfg *resilience.RetryConfig) string {
	var sb strings.Builder

	sb.WriteString("Retry Policy:\n")
	sb.WriteString(fmt.Sprintf("  Max Attempts:   %d\n", cfg.MaxAttempts))
	sb.WriteString(fmt.Sprintf("  Base Delay:     %v\n", cfg.BaseDelay))
	sb.WriteString(fmt.Sprintf("  Max Delay:      %v\n", cfg.MaxDelay))
	sb.WriteString(fmt.Sprintf("  Multiplier:     %.2f\n", cfg.Multiplier))
	sb.WriteString(fmt.Sprintf("  Jitter:         %.0f%%\n", cfg.JitterPercent*100))

	if len(cfg.RetryableErrors) > 0 {
		sb.WriteString(fmt.Sprintf("  Retry On:       %s\n", strings.Join(cfg.RetryableErrors, ", ")))
	}

	return sb.String()
}

// ToDefinition converts a RetryConfig to PolicyDefinition.
func ToDefinition(cfg *resilience.RetryConfig) PolicyDefinition {
	return PolicyDefinition{
		MaxAttempts:   cfg.MaxAttempts,
		BaseDelayMs:   int(cfg.BaseDelay / time.Millisecond),
		MaxDelayMs:    int(cfg.MaxDelay / time.Millisecond),
		Multiplier:    cfg.Multiplier,
		JitterPercent: cfg.JitterPercent,
		RetryOn:       cfg.RetryableErrors,
	}
}

// FromDefinition converts a PolicyDefinition to RetryConfig.
func FromDefinition(def PolicyDefinition) *resilience.RetryConfig {
	return &resilience.RetryConfig{
		MaxAttempts:     def.MaxAttempts,
		BaseDelay:       time.Duration(def.BaseDelayMs) * time.Millisecond,
		MaxDelay:        time.Duration(def.MaxDelayMs) * time.Millisecond,
		Multiplier:      def.Multiplier,
		JitterPercent:   def.JitterPercent,
		RetryableErrors: def.RetryOn,
	}
}
