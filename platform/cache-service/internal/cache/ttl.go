package cache

import (
	"time"
)

// TTLConfig holds TTL-related configuration.
type TTLConfig struct {
	DefaultTTL time.Duration
	MinTTL     time.Duration
	MaxTTL     time.Duration
}

// DefaultTTLConfig returns the default TTL configuration.
func DefaultTTLConfig() TTLConfig {
	return TTLConfig{
		DefaultTTL: time.Hour,
		MinTTL:     time.Second,
		MaxTTL:     24 * time.Hour * 30, // 30 days
	}
}

// ValidateTTL validates and normalizes a TTL value.
func (c TTLConfig) ValidateTTL(ttl time.Duration) (time.Duration, error) {
	if ttl == 0 {
		return c.DefaultTTL, nil
	}

	if ttl < 0 {
		return 0, ErrInvalidTTL
	}

	if ttl < c.MinTTL {
		return c.MinTTL, nil
	}

	if ttl > c.MaxTTL {
		return c.MaxTTL, nil
	}

	return ttl, nil
}

// TTLValidator provides TTL validation functionality.
type TTLValidator struct {
	config TTLConfig
}

// NewTTLValidator creates a new TTL validator.
func NewTTLValidator(config TTLConfig) *TTLValidator {
	return &TTLValidator{config: config}
}

// Validate validates and returns a normalized TTL.
func (v *TTLValidator) Validate(ttl time.Duration) (time.Duration, error) {
	return v.config.ValidateTTL(ttl)
}

// DefaultTTL returns the default TTL.
func (v *TTLValidator) DefaultTTL() time.Duration {
	return v.config.DefaultTTL
}

// IsExpired checks if a given expiration time has passed.
func IsExpired(expiresAt time.Time) bool {
	if expiresAt.IsZero() {
		return false
	}
	return time.Now().After(expiresAt)
}

// CalculateExpiration calculates the expiration time from now.
func CalculateExpiration(ttl time.Duration) time.Time {
	if ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(ttl)
}

// RemainingTTL calculates the remaining TTL from an expiration time.
func RemainingTTL(expiresAt time.Time) time.Duration {
	if expiresAt.IsZero() {
		return -1 // No expiration
	}
	remaining := time.Until(expiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}
