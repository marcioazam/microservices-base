// Feature: go-libs-state-of-art-2025, Property 3: Validation Composition
// Validates: Requirements 3.3, 3.4, 3.5
package validation_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/validation"
	"pgregory.net/rapid"
)

// TestAndComposition verifies And(V1, V2)(v) fails if either V1 or V2 fails.
func TestAndComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(-100, 100).Draw(t, "value")

		// V1: must be positive
		v1 := validation.Positive[int]()
		// V2: must be less than 50
		v2 := validation.Max(50)

		combined := validation.And(v1, v2)
		result := combined(value)

		v1Result := v1(value)
		v2Result := v2(value)

		// And should fail if either fails
		if v1Result != nil || v2Result != nil {
			if result == nil {
				t.Fatalf("And should fail when either validator fails: v1=%v, v2=%v, value=%d",
					v1Result, v2Result, value)
			}
		} else {
			if result != nil {
				t.Fatalf("And should pass when both validators pass: value=%d", value)
			}
		}
	})
}

// TestOrComposition verifies Or(V1, V2)(v) succeeds if either V1 or V2 succeeds.
func TestOrComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(-100, 100).Draw(t, "value")

		// V1: must be negative
		v1 := validation.Max(-1)
		// V2: must be greater than 50
		v2 := validation.Min(51)

		combined := validation.Or(v1, v2)
		result := combined(value)

		v1Result := v1(value)
		v2Result := v2(value)

		// Or should succeed if either succeeds
		if v1Result == nil || v2Result == nil {
			if result != nil {
				t.Fatalf("Or should pass when either validator passes: v1=%v, v2=%v, value=%d",
					v1Result, v2Result, value)
			}
		} else {
			if result == nil {
				t.Fatalf("Or should fail when both validators fail: value=%d", value)
			}
		}
	})
}

// TestNotComposition verifies Not(V)(v) succeeds iff V(v) fails.
func TestNotComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(-100, 100).Draw(t, "value")

		// V: must be positive
		v := validation.Positive[int]()
		notV := validation.Not(v, "must not be positive", "not_positive")

		vResult := v(value)
		notVResult := notV(value)

		// Not should succeed iff original fails
		if vResult == nil {
			// Original passed, Not should fail
			if notVResult == nil {
				t.Fatalf("Not should fail when original passes: value=%d", value)
			}
		} else {
			// Original failed, Not should pass
			if notVResult != nil {
				t.Fatalf("Not should pass when original fails: value=%d", value)
			}
		}
	})
}

// TestErrorAccumulation verifies error accumulation collects all errors.
func TestErrorAccumulation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.StringMatching(`[a-z]{0,5}`).Draw(t, "value")

		// Multiple validators that may fail
		validators := []validation.Validator[string]{
			validation.Required(),
			validation.MinLength(3),
			validation.MaxLength(4),
		}

		result := validation.ValidateAll(value, validators...)

		// Count expected failures
		expectedFailures := 0
		for _, v := range validators {
			if v(value) != nil {
				expectedFailures++
			}
		}

		actualFailures := len(result.Errors())
		if actualFailures != expectedFailures {
			t.Fatalf("expected %d errors, got %d for value %q",
				expectedFailures, actualFailures, value)
		}
	})
}

// TestNestedFieldPaths verifies nested field paths are correctly constructed.
func TestNestedFieldPaths(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		parentPath := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "parent")
		fieldName := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "field")
		value := "" // Empty to trigger validation error

		result := validation.NestedField(parentPath, fieldName, value, validation.Required())

		if result.IsValid() {
			t.Fatal("should have validation error for empty required field")
		}

		errors := result.Errors()
		if len(errors) != 1 {
			t.Fatalf("expected 1 error, got %d", len(errors))
		}

		expectedPath := parentPath + "." + fieldName
		if errors[0].Path != expectedPath {
			t.Fatalf("expected path %q, got %q", expectedPath, errors[0].Path)
		}
		if errors[0].Field != fieldName {
			t.Fatalf("expected field %q, got %q", fieldName, errors[0].Field)
		}
	})
}

// TestFieldValidation verifies Field function tracks field correctly.
func TestFieldValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldName := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "fieldName")
		value := rapid.IntRange(-50, 50).Draw(t, "value")

		result := validation.Field(fieldName, value, validation.Positive[int]())

		if value > 0 {
			if !result.IsValid() {
				t.Fatalf("positive value %d should pass validation", value)
			}
		} else {
			if result.IsValid() {
				t.Fatalf("non-positive value %d should fail validation", value)
			}
			errors := result.Errors()
			if len(errors) != 1 {
				t.Fatalf("expected 1 error, got %d", len(errors))
			}
			if errors[0].Field != fieldName {
				t.Fatalf("expected field %q, got %q", fieldName, errors[0].Field)
			}
		}
	})
}

// TestResultMerge verifies Result.Merge combines errors correctly.
func TestResultMerge(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numErrors1 := rapid.IntRange(0, 5).Draw(t, "numErrors1")
		numErrors2 := rapid.IntRange(0, 5).Draw(t, "numErrors2")

		result1 := validation.NewResult()
		for i := 0; i < numErrors1; i++ {
			result1.AddFieldError("field1", "error", "code")
		}

		result2 := validation.NewResult()
		for i := 0; i < numErrors2; i++ {
			result2.AddFieldError("field2", "error", "code")
		}

		result1.Merge(result2)

		expectedTotal := numErrors1 + numErrors2
		actualTotal := len(result1.Errors())
		if actualTotal != expectedTotal {
			t.Fatalf("expected %d errors after merge, got %d", expectedTotal, actualTotal)
		}
	})
}

// TestErrorMapGrouping verifies ErrorMap groups errors by field.
func TestErrorMapGrouping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		numFields := rapid.IntRange(1, 5).Draw(t, "numFields")
		errorsPerField := rapid.IntRange(1, 3).Draw(t, "errorsPerField")

		result := validation.NewResult()
		for f := 0; f < numFields; f++ {
			fieldName := rapid.StringMatching(`field[0-9]`).Draw(t, "fieldName")
			for e := 0; e < errorsPerField; e++ {
				result.AddFieldError(fieldName, "error message", "code")
			}
		}

		errorMap := result.ErrorMap()

		// Verify all errors are in the map
		totalInMap := 0
		for _, msgs := range errorMap {
			totalInMap += len(msgs)
		}

		if totalInMap != len(result.Errors()) {
			t.Fatalf("error map has %d errors, result has %d", totalInMap, len(result.Errors()))
		}
	})
}

// TestStringValidators verifies string validators work correctly.
func TestStringValidators(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.String().Draw(t, "value")
		minLen := rapid.IntRange(0, 10).Draw(t, "minLen")
		maxLen := rapid.IntRange(minLen, minLen+20).Draw(t, "maxLen")

		// MinLength
		minResult := validation.MinLength(minLen)(value)
		if len(value) < minLen {
			if minResult == nil {
				t.Fatalf("MinLength(%d) should fail for %q (len=%d)", minLen, value, len(value))
			}
		} else {
			if minResult != nil {
				t.Fatalf("MinLength(%d) should pass for %q (len=%d)", minLen, value, len(value))
			}
		}

		// MaxLength
		maxResult := validation.MaxLength(maxLen)(value)
		if len(value) > maxLen {
			if maxResult == nil {
				t.Fatalf("MaxLength(%d) should fail for %q (len=%d)", maxLen, value, len(value))
			}
		} else {
			if maxResult != nil {
				t.Fatalf("MaxLength(%d) should pass for %q (len=%d)", maxLen, value, len(value))
			}
		}
	})
}

// TestNumericValidators verifies numeric validators work correctly.
func TestNumericValidators(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(-100, 100).Draw(t, "value")
		min := rapid.IntRange(-50, 0).Draw(t, "min")
		max := rapid.IntRange(0, 50).Draw(t, "max")

		// InRange
		rangeResult := validation.InRange(min, max)(value)
		if value < min || value > max {
			if rangeResult == nil {
				t.Fatalf("InRange(%d,%d) should fail for %d", min, max, value)
			}
		} else {
			if rangeResult != nil {
				t.Fatalf("InRange(%d,%d) should pass for %d", min, max, value)
			}
		}

		// Positive
		posResult := validation.Positive[int]()(value)
		if value <= 0 {
			if posResult == nil {
				t.Fatalf("Positive should fail for %d", value)
			}
		} else {
			if posResult != nil {
				t.Fatalf("Positive should pass for %d", value)
			}
		}
	})
}

// TestCollectionValidators verifies collection validators work correctly.
func TestCollectionValidators(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := rapid.IntRange(0, 10).Draw(t, "size")
		slice := make([]int, size)
		for i := range slice {
			slice[i] = rapid.Int().Draw(t, "element")
		}

		minSize := rapid.IntRange(0, 5).Draw(t, "minSize")
		maxSize := rapid.IntRange(5, 15).Draw(t, "maxSize")

		// MinSize
		minResult := validation.MinSize[int](minSize)(slice)
		if len(slice) < minSize {
			if minResult == nil {
				t.Fatalf("MinSize(%d) should fail for slice of len %d", minSize, len(slice))
			}
		} else {
			if minResult != nil {
				t.Fatalf("MinSize(%d) should pass for slice of len %d", minSize, len(slice))
			}
		}

		// MaxSize
		maxResult := validation.MaxSize[int](maxSize)(slice)
		if len(slice) > maxSize {
			if maxResult == nil {
				t.Fatalf("MaxSize(%d) should fail for slice of len %d", maxSize, len(slice))
			}
		} else {
			if maxResult != nil {
				t.Fatalf("MaxSize(%d) should pass for slice of len %d", maxSize, len(slice))
			}
		}
	})
}

// TestUniqueElements verifies UniqueElements validator.
func TestUniqueElements(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate slice with potential duplicates
		size := rapid.IntRange(0, 10).Draw(t, "size")
		slice := make([]int, size)
		for i := range slice {
			slice[i] = rapid.IntRange(0, 5).Draw(t, "element") // Small range to encourage duplicates
		}

		result := validation.UniqueElements[int]()(slice)

		// Check for actual duplicates
		seen := make(map[int]bool)
		hasDuplicates := false
		for _, v := range slice {
			if seen[v] {
				hasDuplicates = true
				break
			}
			seen[v] = true
		}

		if hasDuplicates {
			if result == nil {
				t.Fatal("UniqueElements should fail for slice with duplicates")
			}
		} else {
			if result != nil {
				t.Fatal("UniqueElements should pass for slice without duplicates")
			}
		}
	})
}

// TestCustomValidator verifies Custom validator works correctly.
func TestCustomValidator(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(0, 100).Draw(t, "value")

		// Custom: must be even
		isEven := validation.Custom(func(v int) bool {
			return v%2 == 0
		}, "must be even", "even")

		result := isEven(value)

		if value%2 == 0 {
			if result != nil {
				t.Fatalf("Custom(even) should pass for %d", value)
			}
		} else {
			if result == nil {
				t.Fatalf("Custom(even) should fail for %d", value)
			}
		}
	})
}
