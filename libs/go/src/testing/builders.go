package testing

import (
	"time"
)

// Builder provides a fluent interface for building test fixtures.
type Builder[T any] struct {
	value T
}

// NewBuilder creates a new builder with a zero value.
func NewBuilder[T any]() *Builder[T] {
	var zero T
	return &Builder[T]{value: zero}
}

// NewBuilderFrom creates a new builder from an existing value.
func NewBuilderFrom[T any](value T) *Builder[T] {
	return &Builder[T]{value: value}
}

// With applies a modifier function to the value.
func (b *Builder[T]) With(fn func(*T)) *Builder[T] {
	fn(&b.value)
	return b
}

// Build returns the built value.
func (b *Builder[T]) Build() T {
	return b.value
}

// TestConfig represents common test configuration.
type TestConfig struct {
	Timeout     time.Duration
	Retries     int
	Parallel    bool
	SkipCleanup bool
}

// DefaultTestConfig returns default test configuration.
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Timeout:     30 * time.Second,
		Retries:     3,
		Parallel:    true,
		SkipCleanup: false,
	}
}

// TestFixture provides common test fixture functionality.
type TestFixture struct {
	config   TestConfig
	cleanups []func()
}

// NewTestFixture creates a new test fixture.
func NewTestFixture(config TestConfig) *TestFixture {
	return &TestFixture{
		config:   config,
		cleanups: make([]func(), 0),
	}
}

// AddCleanup adds a cleanup function to be called on teardown.
func (f *TestFixture) AddCleanup(fn func()) {
	f.cleanups = append(f.cleanups, fn)
}

// Cleanup runs all cleanup functions in reverse order.
func (f *TestFixture) Cleanup() {
	if f.config.SkipCleanup {
		return
	}
	for i := len(f.cleanups) - 1; i >= 0; i-- {
		f.cleanups[i]()
	}
}

// Config returns the test configuration.
func (f *TestFixture) Config() TestConfig {
	return f.config
}
