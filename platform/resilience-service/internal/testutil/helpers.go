package testutil

import (
	"testing"

	"github.com/leanovate/gopter"
)

// DefaultTestParameters returns standard gopter parameters for property tests.
// Uses 100 iterations as specified in the design document.
func DefaultTestParameters() *gopter.TestParameters {
	params := gopter.DefaultTestParameters()
	params.MinSuccessfulTests = 100
	params.MaxSize = 100
	return params
}

// RunPropertyTest runs a property test with standard parameters.
func RunPropertyTest(t *testing.T, name string, prop gopter.Prop) {
	t.Helper()
	params := DefaultTestParameters()
	props := gopter.NewProperties(params)
	props.Property(name, prop)
	props.TestingRun(t)
}

// AssertNoError fails the test if err is not nil.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil.
func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

// AssertEqual fails the test if got != want.
func AssertEqual[T comparable](t *testing.T, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// AssertTrue fails the test if condition is false.
func AssertTrue(t *testing.T, condition bool, msg string) {
	t.Helper()
	if !condition {
		t.Fatalf("assertion failed: %s", msg)
	}
}

// GenerateAlphaString generates a deterministic alphabetic string of given length.
func GenerateAlphaString(length int) string {
	if length <= 0 {
		return ""
	}
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = alphabet[i%len(alphabet)]
	}
	return string(result)
}

// GenerateAlphanumericString generates a deterministic alphanumeric string of given length.
func GenerateAlphanumericString(length int) string {
	if length <= 0 {
		return ""
	}
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := 0; i < length; i++ {
		result[i] = chars[i%len(chars)]
	}
	return string(result)
}
