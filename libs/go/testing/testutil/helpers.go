// Package testutil provides test utilities and helpers for property-based testing.
package testutil

import (
	"fmt"
	"reflect"
	"testing"
)

// TestParameters holds configuration for property-based tests.
type TestParameters struct {
	MinIterations int
	MaxIterations int
	Seed          int64
}

// DefaultTestParameters returns default test parameters.
// Uses 100 minimum iterations as per requirements.
func DefaultTestParameters() TestParameters {
	return TestParameters{
		MinIterations: 100,
		MaxIterations: 1000,
		Seed:          0, // 0 means use current time
	}
}

// PropertyTest represents a property-based test.
type PropertyTest[T any] struct {
	Name       string
	Generator  func() T
	Property   func(T) bool
	Parameters TestParameters
}

// RunPropertyTest runs a property-based test with the given parameters.
func RunPropertyTest[T any](t *testing.T, test PropertyTest[T]) {
	t.Helper()

	params := test.Parameters
	if params.MinIterations == 0 {
		params = DefaultTestParameters()
	}

	for i := 0; i < params.MinIterations; i++ {
		value := test.Generator()
		if !test.Property(value) {
			t.Errorf("property %q failed for value: %v (iteration %d)", test.Name, value, i)
			return
		}
	}
}

// Assert provides assertion helpers for tests.
type Assert struct {
	t *testing.T
}

// NewAssert creates a new Assert helper.
func NewAssert(t *testing.T) *Assert {
	return &Assert{t: t}
}

// True asserts that the condition is true.
func (a *Assert) True(condition bool, msg ...string) {
	a.t.Helper()
	if !condition {
		if len(msg) > 0 {
			a.t.Errorf("expected true: %s", msg[0])
		} else {
			a.t.Error("expected true")
		}
	}
}

// False asserts that the condition is false.
func (a *Assert) False(condition bool, msg ...string) {
	a.t.Helper()
	if condition {
		if len(msg) > 0 {
			a.t.Errorf("expected false: %s", msg[0])
		} else {
			a.t.Error("expected false")
		}
	}
}

// Equal asserts that two values are equal.
func (a *Assert) Equal(expected, actual interface{}, msg ...string) {
	a.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected %v, got %v", msg[0], expected, actual)
		} else {
			a.t.Errorf("expected %v, got %v", expected, actual)
		}
	}
}

// NotEqual asserts that two values are not equal.
func (a *Assert) NotEqual(expected, actual interface{}, msg ...string) {
	a.t.Helper()
	if reflect.DeepEqual(expected, actual) {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected values to differ, both are %v", msg[0], expected)
		} else {
			a.t.Errorf("expected values to differ, both are %v", expected)
		}
	}
}

// Nil asserts that the value is nil.
func (a *Assert) Nil(value interface{}, msg ...string) {
	a.t.Helper()
	if !isNil(value) {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected nil, got %v", msg[0], value)
		} else {
			a.t.Errorf("expected nil, got %v", value)
		}
	}
}

// NotNil asserts that the value is not nil.
func (a *Assert) NotNil(value interface{}, msg ...string) {
	a.t.Helper()
	if isNil(value) {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected non-nil value", msg[0])
		} else {
			a.t.Error("expected non-nil value")
		}
	}
}

// NoError asserts that the error is nil.
func (a *Assert) NoError(err error, msg ...string) {
	a.t.Helper()
	if err != nil {
		if len(msg) > 0 {
			a.t.Errorf("%s: unexpected error: %v", msg[0], err)
		} else {
			a.t.Errorf("unexpected error: %v", err)
		}
	}
}

// Error asserts that the error is not nil.
func (a *Assert) Error(err error, msg ...string) {
	a.t.Helper()
	if err == nil {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected error, got nil", msg[0])
		} else {
			a.t.Error("expected error, got nil")
		}
	}
}

// Panics asserts that the function panics.
func (a *Assert) Panics(fn func(), msg ...string) {
	a.t.Helper()
	defer func() {
		if r := recover(); r == nil {
			if len(msg) > 0 {
				a.t.Errorf("%s: expected panic", msg[0])
			} else {
				a.t.Error("expected panic")
			}
		}
	}()
	fn()
}

// NotPanics asserts that the function does not panic.
func (a *Assert) NotPanics(fn func(), msg ...string) {
	a.t.Helper()
	defer func() {
		if r := recover(); r != nil {
			if len(msg) > 0 {
				a.t.Errorf("%s: unexpected panic: %v", msg[0], r)
			} else {
				a.t.Errorf("unexpected panic: %v", r)
			}
		}
	}()
	fn()
}

// Contains asserts that the slice contains the element.
func Contains[T comparable](a *Assert, slice []T, element T, msg ...string) {
	a.t.Helper()
	for _, v := range slice {
		if v == element {
			return
		}
	}
	if len(msg) > 0 {
		a.t.Errorf("%s: slice does not contain %v", msg[0], element)
	} else {
		a.t.Errorf("slice does not contain %v", element)
	}
}

// NotContains asserts that the slice does not contain the element.
func NotContains[T comparable](a *Assert, slice []T, element T, msg ...string) {
	a.t.Helper()
	for _, v := range slice {
		if v == element {
			if len(msg) > 0 {
				a.t.Errorf("%s: slice contains %v", msg[0], element)
			} else {
				a.t.Errorf("slice contains %v", element)
			}
			return
		}
	}
}

// Len asserts that the slice has the expected length.
func Len[T any](a *Assert, slice []T, expected int, msg ...string) {
	a.t.Helper()
	if len(slice) != expected {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected length %d, got %d", msg[0], expected, len(slice))
		} else {
			a.t.Errorf("expected length %d, got %d", expected, len(slice))
		}
	}
}

// Empty asserts that the slice is empty.
func Empty[T any](a *Assert, slice []T, msg ...string) {
	a.t.Helper()
	if len(slice) != 0 {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected empty slice, got %d elements", msg[0], len(slice))
		} else {
			a.t.Errorf("expected empty slice, got %d elements", len(slice))
		}
	}
}

// NotEmpty asserts that the slice is not empty.
func NotEmpty[T any](a *Assert, slice []T, msg ...string) {
	a.t.Helper()
	if len(slice) == 0 {
		if len(msg) > 0 {
			a.t.Errorf("%s: expected non-empty slice", msg[0])
		} else {
			a.t.Error("expected non-empty slice")
		}
	}
}

// isNil checks if a value is nil.
func isNil(value interface{}) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

// Require provides assertion helpers that fail immediately.
type Require struct {
	t *testing.T
}

// NewRequire creates a new Require helper.
func NewRequire(t *testing.T) *Require {
	return &Require{t: t}
}

// True requires that the condition is true.
func (r *Require) True(condition bool, msg ...string) {
	r.t.Helper()
	if !condition {
		if len(msg) > 0 {
			r.t.Fatalf("required true: %s", msg[0])
		} else {
			r.t.Fatal("required true")
		}
	}
}

// NoError requires that the error is nil.
func (r *Require) NoError(err error, msg ...string) {
	r.t.Helper()
	if err != nil {
		if len(msg) > 0 {
			r.t.Fatalf("%s: unexpected error: %v", msg[0], err)
		} else {
			r.t.Fatalf("unexpected error: %v", err)
		}
	}
}

// NotNil requires that the value is not nil.
func (r *Require) NotNil(value interface{}, msg ...string) {
	r.t.Helper()
	if isNil(value) {
		if len(msg) > 0 {
			r.t.Fatalf("%s: required non-nil value", msg[0])
		} else {
			r.t.Fatal("required non-nil value")
		}
	}
}

// Equal requires that two values are equal.
func (r *Require) Equal(expected, actual interface{}, msg ...string) {
	r.t.Helper()
	if !reflect.DeepEqual(expected, actual) {
		if len(msg) > 0 {
			r.t.Fatalf("%s: required %v, got %v", msg[0], expected, actual)
		} else {
			r.t.Fatalf("required %v, got %v", expected, actual)
		}
	}
}

// FormatTestName formats a test name with parameters.
func FormatTestName(name string, params ...interface{}) string {
	if len(params) == 0 {
		return name
	}
	return fmt.Sprintf("%s_%v", name, params)
}

// TableTest represents a table-driven test case.
type TableTest[T any] struct {
	Name     string
	Input    T
	Expected interface{}
	WantErr  bool
}

// RunTableTests runs table-driven tests.
func RunTableTests[T any](t *testing.T, tests []TableTest[T], fn func(T) (interface{}, error)) {
	for _, tc := range tests {
		t.Run(tc.Name, func(t *testing.T) {
			result, err := fn(tc.Input)
			if tc.WantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !reflect.DeepEqual(result, tc.Expected) {
				t.Errorf("expected %v, got %v", tc.Expected, result)
			}
		})
	}
}
