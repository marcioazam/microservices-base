package testing

import (
	"reflect"
	"testing"
)

// Assert provides fluent assertion helpers for tests.
type Assert struct {
	t *testing.T
}

// NewAssert creates a new Assert instance.
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// Equal asserts that two values are equal.
func (a *Assert) Equal(expected, actual interface{}) {
	a.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		a.t.Errorf("Expected %v, got %v", expected, actual)
	}
}

// NotEqual asserts that two values are not equal.
func (a *Assert) NotEqual(expected, actual interface{}) {
	a.t.Helper()
	if reflect.DeepEqual(expected, actual) {
		a.t.Errorf("Expected values to be different, both are %v", expected)
	}
}

// True asserts that a condition is true.
func (a *Assert) True(condition bool, msg ...string) {
	a.t.Helper()
	if !condition {
		if len(msg) > 0 {
			a.t.Errorf("Expected true: %s", msg[0])
		} else {
			a.t.Error("Expected true, got false")
		}
	}
}

// False asserts that a condition is false.
func (a *Assert) False(condition bool, msg ...string) {
	a.t.Helper()
	if condition {
		if len(msg) > 0 {
			a.t.Errorf("Expected false: %s", msg[0])
		} else {
			a.t.Error("Expected false, got true")
		}
	}
}

// Nil asserts that a value is nil.
func (a *Assert) Nil(value interface{}) {
	a.t.Helper()
	if value != nil && !reflect.ValueOf(value).IsNil() {
		a.t.Errorf("Expected nil, got %v", value)
	}
}

// NotNil asserts that a value is not nil.
func (a *Assert) NotNil(value interface{}) {
	a.t.Helper()
	if value == nil || reflect.ValueOf(value).IsNil() {
		a.t.Error("Expected non-nil value")
	}
}

// NoError asserts that an error is nil.
func (a *Assert) NoError(err error) {
	a.t.Helper()
	if err != nil {
		a.t.Errorf("Expected no error, got: %v", err)
	}
}

// Error asserts that an error is not nil.
func (a *Assert) Error(err error) {
	a.t.Helper()
	if err == nil {
		a.t.Error("Expected an error, got nil")
	}
}

// Len asserts that a collection has the expected length.
func (a *Assert) Len(collection interface{}, expected int) {
	a.t.Helper()
	v := reflect.ValueOf(collection)
	if v.Len() != expected {
		a.t.Errorf("Expected length %d, got %d", expected, v.Len())
	}
}

// Contains asserts that a string contains a substring.
func (a *Assert) Contains(s, substr string) {
	a.t.Helper()
	if !containsString(s, substr) {
		a.t.Errorf("Expected %q to contain %q", s, substr)
	}
}

// Panics asserts that a function panics.
func (a *Assert) Panics(fn func()) {
	a.t.Helper()
	defer func() {
		if r := recover(); r == nil {
			a.t.Error("Expected panic, but function did not panic")
		}
	}()
	fn()
}

// NoPanic asserts that a function does not panic.
func (a *Assert) NoPanic(fn func()) {
	a.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			a.t.Errorf("Expected no panic, but got: %v", r)
		}
	}()
	fn()
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
