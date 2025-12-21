// Package rand provides random number generation for resilience patterns.
package rand

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mathrand "math/rand"
	"sync"
)

// RandSource provides random number generation for jitter calculations.
type RandSource interface {
	// Float64 returns a random float64 in [0.0, 1.0).
	Float64() float64
}

// CryptoRandSource uses crypto/rand for secure random number generation.
type CryptoRandSource struct {
	mu   sync.Mutex
	rand *mathrand.Rand
}

// NewCryptoRandSource creates a new cryptographically seeded random source.
func NewCryptoRandSource() *CryptoRandSource {
	return &CryptoRandSource{
		rand: mathrand.New(mathrand.NewSource(cryptoSeed())),
	}
}

// Float64 returns a cryptographically seeded random float64 in [0.0, 1.0).
func (c *CryptoRandSource) Float64() float64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rand.Float64()
}

// cryptoSeed generates a cryptographically secure seed.
func cryptoSeed() int64 {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fallback to a fixed seed if crypto/rand fails (should never happen)
		return 0
	}
	return int64(binary.LittleEndian.Uint64(b[:]))
}

// DeterministicRandSource provides deterministic random numbers for testing.
type DeterministicRandSource struct {
	rand *mathrand.Rand
}

// NewDeterministicRandSource creates a deterministic random source with a fixed seed.
func NewDeterministicRandSource(seed int64) *DeterministicRandSource {
	return &DeterministicRandSource{
		rand: mathrand.New(mathrand.NewSource(seed)),
	}
}

// Float64 returns a deterministic random float64 in [0.0, 1.0).
func (d *DeterministicRandSource) Float64() float64 {
	return d.rand.Float64()
}

// FixedRandSource always returns a fixed value for testing.
type FixedRandSource struct {
	value float64
}

// NewFixedRandSource creates a random source that always returns the same value.
func NewFixedRandSource(value float64) *FixedRandSource {
	return &FixedRandSource{value: math.Max(0, math.Min(value, 0.9999999999))}
}

// Float64 returns the fixed value.
func (f *FixedRandSource) Float64() float64 {
	return f.value
}
