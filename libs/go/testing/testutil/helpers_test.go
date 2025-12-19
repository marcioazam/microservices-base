// Package testutil provides tests for the test utilities.
package testutil

import (
	"errors"
	"testing"
)

func TestDefaultTestParameters(t *testing.T) {
	params := DefaultTestParameters()

	if params.MinIterations != 100 {
		t.Errorf("expected MinIterations=100, got %d", params.MinIterations)
	}
}

func TestRunPropertyTest(t *testing.T) {
	test := PropertyTest[int]{
		Name:      "positive numbers",
		Generator: func() int { return 42 },
		Property:  func(n int) bool { return n > 0 },
	}

	// Should not fail
	RunPropertyTest(t, test)
}

func TestAssert_True(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.True(true)
	// No error expected
}

func TestAssert_False(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.False(false)
	// No error expected
}

func TestAssert_Equal(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.Equal(42, 42)
	a.Equal("hello", "hello")
	a.Equal([]int{1, 2, 3}, []int{1, 2, 3})
}

func TestAssert_NotEqual(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.NotEqual(42, 43)
	a.NotEqual("hello", "world")
}

func TestAssert_Nil(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.Nil(nil)

	var ptr *int
	a.Nil(ptr)
}

func TestAssert_NotNil(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.NotNil(42)
	a.NotNil("hello")

	ptr := new(int)
	a.NotNil(ptr)
}

func TestAssert_NoError(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.NoError(nil)
}

func TestAssert_Error(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.Error(errors.New("test error"))
}

func TestAssert_Panics(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.Panics(func() { panic("test") })
}

func TestAssert_NotPanics(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	a.NotPanics(func() {})
}

func TestContains(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	Contains(a, []int{1, 2, 3}, 2)
	Contains(a, []string{"a", "b", "c"}, "b")
}

func TestNotContains(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	NotContains(a, []int{1, 2, 3}, 4)
	NotContains(a, []string{"a", "b", "c"}, "d")
}

func TestLen(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	Len(a, []int{1, 2, 3}, 3)
	Len(a, []string{}, 0)
}

func TestEmpty(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	Empty(a, []int{})
	Empty(a, []string{})
}

func TestNotEmpty(t *testing.T) {
	mockT := &testing.T{}
	a := NewAssert(mockT)
	NotEmpty(a, []int{1})
	NotEmpty(a, []string{"a"})
}

func TestRequire_True(t *testing.T) {
	mockT := &testing.T{}
	r := NewRequire(mockT)
	r.True(true)
}

func TestRequire_NoError(t *testing.T) {
	mockT := &testing.T{}
	r := NewRequire(mockT)
	r.NoError(nil)
}

func TestRequire_NotNil(t *testing.T) {
	mockT := &testing.T{}
	r := NewRequire(mockT)
	r.NotNil(42)
}

func TestRequire_Equal(t *testing.T) {
	mockT := &testing.T{}
	r := NewRequire(mockT)
	r.Equal(42, 42)
}

func TestFormatTestName(t *testing.T) {
	name := FormatTestName("test", 1, "a")
	if name != "test_[1 a]" {
		t.Errorf("unexpected name: %s", name)
	}

	name = FormatTestName("simple")
	if name != "simple" {
		t.Errorf("unexpected name: %s", name)
	}
}

func TestRunTableTests(t *testing.T) {
	tests := []TableTest[int]{
		{Name: "positive", Input: 5, Expected: 10, WantErr: false},
		{Name: "zero", Input: 0, Expected: 0, WantErr: false},
		{Name: "negative", Input: -1, Expected: nil, WantErr: true},
	}

	fn := func(n int) (interface{}, error) {
		if n < 0 {
			return nil, errors.New("negative")
		}
		return n * 2, nil
	}

	RunTableTests(t, tests, fn)
}

func TestIsNil(t *testing.T) {
	if !isNil(nil) {
		t.Error("nil should be nil")
	}

	var ptr *int
	if !isNil(ptr) {
		t.Error("nil pointer should be nil")
	}

	var slice []int
	if !isNil(slice) {
		t.Error("nil slice should be nil")
	}

	var m map[string]int
	if !isNil(m) {
		t.Error("nil map should be nil")
	}

	if isNil(42) {
		t.Error("42 should not be nil")
	}

	if isNil("hello") {
		t.Error("string should not be nil")
	}
}
