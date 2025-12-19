package utils_test

import (
	"testing"

	"github.com/authcorp/libs/go/src/utils"
	"pgregory.net/rapid"
)

// Property 9: Validation Error Accumulation
// Multiple validation errors are accumulated, not short-circuited.
func TestProperty_ValidationErrorAccumulation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create validator with multiple failing rules
		// Use Required and MinLength - empty string fails both
		validator := utils.NewValidator[string]("field").
			AddRule(utils.Required()).
			AddRule(utils.MinLength(10))

		// Empty string fails both rules
		result := validator.Validate("")

		if result.IsValid() {
			t.Fatalf("Should be invalid")
		}

		// Should have accumulated both errors
		errors := result.Errors()
		if len(errors) < 2 {
			t.Fatalf("Expected at least 2 errors, got %d", len(errors))
		}
	})
}

// Property 10: Validator And Composition
// And-composed validators accumulate errors from both.
func TestProperty_ValidatorAndComposition(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		v1 := utils.NewValidator[string]("field").AddRule(utils.Required())
		v2 := utils.NewValidator[string]("field").AddRule(utils.MinLength(5))

		combined := v1.And(v2)

		// Empty string fails both
		result := combined.Validate("")

		if result.IsValid() {
			t.Fatalf("Should be invalid")
		}

		errors := result.Errors()
		if len(errors) < 2 {
			t.Fatalf("Expected at least 2 errors from combined validator")
		}
	})
}

// Property 11: Validation Path Tracking
// Validation errors include field path.
func TestProperty_ValidationPathTracking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fieldName := rapid.StringMatching(`[a-z]+`).Draw(t, "fieldName")

		validator := utils.NewValidator[string](fieldName).
			AddRule(utils.Required())

		result := validator.Validate("")

		if result.IsValid() {
			t.Fatalf("Should be invalid")
		}

		errors := result.Errors()
		if len(errors) == 0 {
			t.Fatalf("Expected at least one error")
		}

		if errors[0].Field != fieldName {
			t.Fatalf("Expected field %s, got %s", fieldName, errors[0].Field)
		}
	})
}

// Property 12: Validation Result Exclusivity
// Validation is either valid or has errors, never both.
func TestProperty_ValidationResultExclusivity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.String().Draw(t, "value")

		validator := utils.NewValidator[string]("field").
			AddRule(utils.MinLength(5))

		result := validator.Validate(value)

		hasErrors := len(result.Errors()) > 0
		isValid := result.IsValid()

		// XOR: exactly one must be true
		if isValid == hasErrors {
			t.Fatalf("Validation must be either valid or have errors, not both or neither")
		}
	})
}

// Property: Valid values pass validation
func TestProperty_ValidValuesPasses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate string that satisfies constraints
		value := rapid.StringMatching(`[a-zA-Z]{5,10}`).Draw(t, "value")

		validator := utils.NewValidator[string]("field").
			AddRule(utils.MinLength(5)).
			AddRule(utils.MaxLength(10))

		result := validator.Validate(value)

		if !result.IsValid() {
			t.Fatalf("Valid value should pass: %s, errors: %v", value, result.ErrorMessages())
		}
	})
}

// Property: Range validation
func TestProperty_RangeValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		min := rapid.IntRange(0, 50).Draw(t, "min")
		max := rapid.IntRange(51, 100).Draw(t, "max")
		value := rapid.IntRange(min, max).Draw(t, "value")

		validator := utils.NewValidator[int]("field").
			AddRule(utils.Range(min, max))

		result := validator.Validate(value)

		if !result.IsValid() {
			t.Fatalf("Value %d should be in range [%d, %d]", value, min, max)
		}
	})
}

// Property: OneOf validation
func TestProperty_OneOfValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		allowed := []string{"a", "b", "c"}
		value := rapid.SampledFrom(allowed).Draw(t, "value")

		validator := utils.NewValidator[string]("field").
			AddRule(utils.OneOf(allowed...))

		result := validator.Validate(value)

		if !result.IsValid() {
			t.Fatalf("Value %s should be in allowed list", value)
		}
	})
}

// Property: Custom rule validation
func TestProperty_CustomRuleValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		isEven := utils.Custom(
			func(v int) bool { return v%2 == 0 },
			"must be even",
			"even",
		)

		validator := utils.NewValidator[int]("field").AddRule(isEven)
		result := validator.Validate(value)

		expectedValid := value%2 == 0
		if result.IsValid() != expectedValid {
			t.Fatalf("Expected valid=%v for value %d", expectedValid, value)
		}
	})
}
