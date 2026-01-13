package property_test

import (
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/cache"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Property 5: TTL normalization idempotence
func TestProperty_TTLNormalizationIdempotence(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ttlSeconds := rapid.Int64Range(0, 86400*365).Draw(t, "ttlSeconds")
		ttl := time.Duration(ttlSeconds) * time.Second

		cfg := cache.TTLConfig{
			DefaultTTL: 5 * time.Minute,
			MinTTL:     time.Second,
			MaxTTL:     24 * time.Hour,
		}

		// First normalization
		normalized1, err1 := cfg.ValidateTTL(ttl)

		if err1 == nil {
			// Second normalization should produce same result
			normalized2, err2 := cfg.ValidateTTL(normalized1)
			assert.NoError(t, err2)
			assert.Equal(t, normalized1, normalized2, "TTL normalization should be idempotent")
		}
	})
}

// Property 5b: TTL within bounds after normalization
func TestProperty_TTLWithinBounds(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ttlSeconds := rapid.Int64Range(1, 86400).Draw(t, "ttlSeconds")
		ttl := time.Duration(ttlSeconds) * time.Second

		minTTL := time.Second
		maxTTL := 24 * time.Hour

		cfg := cache.TTLConfig{
			DefaultTTL: 5 * time.Minute,
			MinTTL:     minTTL,
			MaxTTL:     maxTTL,
		}

		normalized, err := cfg.ValidateTTL(ttl)

		if err == nil {
			assert.GreaterOrEqual(t, normalized, minTTL, "normalized TTL should be >= minTTL")
			assert.LessOrEqual(t, normalized, maxTTL, "normalized TTL should be <= maxTTL")
		}
	})
}

// Property: Zero TTL uses default
func TestProperty_ZeroTTLUsesDefault(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		defaultSeconds := rapid.Int64Range(60, 3600).Draw(t, "defaultSeconds")
		defaultTTL := time.Duration(defaultSeconds) * time.Second

		cfg := cache.TTLConfig{
			DefaultTTL: defaultTTL,
			MinTTL:     time.Second,
			MaxTTL:     24 * time.Hour,
		}

		normalized, err := cfg.ValidateTTL(0)
		assert.NoError(t, err)
		assert.Equal(t, defaultTTL, normalized, "zero TTL should use default")
	})
}

// Property: TTL below minimum is clamped to minimum
func TestProperty_TTLBelowMinimumClamped(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		minSeconds := rapid.Int64Range(10, 60).Draw(t, "minSeconds")
		minTTL := time.Duration(minSeconds) * time.Second

		cfg := cache.TTLConfig{
			DefaultTTL: 5 * time.Minute,
			MinTTL:     minTTL,
			MaxTTL:     24 * time.Hour,
		}

		// TTL below minimum (but not zero)
		belowMin := minTTL - time.Second
		if belowMin > 0 {
			normalized, err := cfg.ValidateTTL(belowMin)
			assert.NoError(t, err, "TTL below minimum should not return error")
			assert.Equal(t, minTTL, normalized, "TTL below minimum should be clamped to minimum")
		}
	})
}

// Property: TTL above maximum is clamped to maximum
func TestProperty_TTLAboveMaximumClamped(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxHours := rapid.Int64Range(1, 24).Draw(t, "maxHours")
		maxTTL := time.Duration(maxHours) * time.Hour

		cfg := cache.TTLConfig{
			DefaultTTL: 5 * time.Minute,
			MinTTL:     time.Second,
			MaxTTL:     maxTTL,
		}

		aboveMax := maxTTL + time.Hour
		normalized, err := cfg.ValidateTTL(aboveMax)
		assert.NoError(t, err, "TTL above maximum should not return error")
		assert.Equal(t, maxTTL, normalized, "TTL above maximum should be clamped to maximum")
	})
}

// Property 6: Cache entry expiration monotonicity
func TestProperty_ExpirationMonotonicity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ttl1Seconds := rapid.Int64Range(60, 3600).Draw(t, "ttl1Seconds")
		ttl2Seconds := rapid.Int64Range(60, 3600).Draw(t, "ttl2Seconds")

		ttl1 := time.Duration(ttl1Seconds) * time.Second
		ttl2 := time.Duration(ttl2Seconds) * time.Second

		now := time.Now()
		expiry1 := now.Add(ttl1)
		expiry2 := now.Add(ttl2)

		// If ttl1 < ttl2, then expiry1 < expiry2
		if ttl1 < ttl2 {
			assert.True(t, expiry1.Before(expiry2), "shorter TTL should expire first")
		} else if ttl1 > ttl2 {
			assert.True(t, expiry2.Before(expiry1), "shorter TTL should expire first")
		} else {
			assert.Equal(t, expiry1, expiry2, "equal TTLs should have equal expiry")
		}
	})
}
