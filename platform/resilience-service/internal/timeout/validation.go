package timeout

import (
	"fmt"
	"time"

	"github.com/auth-platform/platform/resilience-service/internal/domain"
)

const (
	// MinTimeout is the minimum allowed timeout.
	MinTimeout = time.Millisecond

	// MaxTimeout is the maximum allowed timeout.
	MaxTimeout = 5 * time.Minute
)

// ValidationError represents a timeout validation error.
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidateConfig validates a timeout configuration.
func ValidateConfig(cfg domain.TimeoutConfig) error {
	if err := validateDuration("default", cfg.Default); err != nil {
		return err
	}

	if cfg.Max > 0 {
		if err := validateDuration("max", cfg.Max); err != nil {
			return err
		}
	}

	for op, timeout := range cfg.PerOp {
		if err := validateDuration(fmt.Sprintf("per_operation[%s]", op), timeout); err != nil {
			return err
		}
	}

	return nil
}

// validateDuration validates a single duration value.
func validateDuration(field string, d time.Duration) error {
	if d <= 0 {
		return ValidationError{
			Field:   field,
			Message: "must be a positive duration",
		}
	}

	if d < MinTimeout {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %v", MinTimeout),
		}
	}

	if d > MaxTimeout {
		return ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must not exceed %v", MaxTimeout),
		}
	}

	return nil
}

// IsValidTimeout checks if a timeout value is valid.
func IsValidTimeout(d time.Duration) bool {
	return d > 0 && d >= MinTimeout && d <= MaxTimeout
}
