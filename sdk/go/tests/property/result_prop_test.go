package property_test

import (
	"errors"
	"testing"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestResultMapFlatMapCorrectness tests Property 30: Result Map/FlatMap Correctness
// **Feature: go-sdk-modernization, Property 30: Result Map/FlatMap Correctness**
// **Validates: Requirements 14.3**
func TestResultMapFlatMapCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		isOk := rapid.Bool().Draw(t, "isOk")

		var r authplatform.Result[int]
		if isOk {
			r = authplatform.Ok(value)
		} else {
			r = authplatform.Err[int](errors.New("test error"))
		}

		// Test Map
		doubled := authplatform.Map(r, func(v int) int { return v * 2 })

		if isOk {
			// Property: Map on Ok transforms the value
			if !doubled.IsOk() {
				t.Fatal("Map on Ok should return Ok")
			}
			if doubled.Unwrap() != value*2 {
				t.Fatalf("Map should transform value: expected %d, got %d", value*2, doubled.Unwrap())
			}
		} else {
			// Property: Map on Err preserves the error
			if !doubled.IsErr() {
				t.Fatal("Map on Err should return Err")
			}
		}

		// Test FlatMap
		flatMapped := authplatform.FlatMap(r, func(v int) authplatform.Result[string] {
			if v > 0 {
				return authplatform.Ok("positive")
			}
			return authplatform.Err[string](errors.New("not positive"))
		})

		if isOk {
			// Property: FlatMap on Ok chains correctly
			if value > 0 {
				if !flatMapped.IsOk() {
					t.Fatal("FlatMap should return Ok for positive value")
				}
				if flatMapped.Unwrap() != "positive" {
					t.Fatal("FlatMap should return correct value")
				}
			} else {
				if !flatMapped.IsErr() {
					t.Fatal("FlatMap should return Err for non-positive value")
				}
			}
		} else {
			// Property: FlatMap on Err preserves the original error
			if !flatMapped.IsErr() {
				t.Fatal("FlatMap on Err should return Err")
			}
		}
	})
}

// TestResultErrorPreservation tests Property 31: Result Error Preservation
// **Feature: go-sdk-modernization, Property 31: Result Error Preservation**
// **Validates: Requirements 14.4**
func TestResultErrorPreservation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		errMsg := rapid.StringMatching(`[a-zA-Z0-9 ]+`).Draw(t, "errMsg")
		if errMsg == "" {
			errMsg = "test error"
		}

		originalErr := errors.New(errMsg)
		r := authplatform.Err[int](originalErr)

		// Property: Error() returns the original error
		if r.Error() != originalErr {
			t.Fatal("Error() should return the original error")
		}

		// Property: UnwrapErr() returns the original error
		if r.UnwrapErr() != originalErr {
			t.Fatal("UnwrapErr() should return the original error")
		}

		// Property: Error is preserved through Map
		mapped := authplatform.Map(r, func(v int) string { return "transformed" })
		if mapped.Error() != originalErr {
			t.Fatal("Error should be preserved through Map")
		}

		// Property: Error is preserved through FlatMap
		flatMapped := authplatform.FlatMap(r, func(v int) authplatform.Result[string] {
			return authplatform.Ok("should not reach")
		})
		if flatMapped.Error() != originalErr {
			t.Fatal("Error should be preserved through FlatMap")
		}

		// Property: Error is preserved through MapErr transformation
		transformedErr := errors.New("transformed: " + errMsg)
		mappedErr := authplatform.MapErr(r, func(e error) error { return transformedErr })
		if mappedErr.Error() != transformedErr {
			t.Fatal("MapErr should transform the error")
		}
	})
}

// TestResultMatchCorrectness tests Property 32: Result Match Correctness
// **Feature: go-sdk-modernization, Property 32: Result Match Correctness**
// **Validates: Requirements 14.5**
func TestResultMatchCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		isOk := rapid.Bool().Draw(t, "isOk")

		var r authplatform.Result[int]
		if isOk {
			r = authplatform.Ok(value)
		} else {
			r = authplatform.Err[int](errors.New("test error"))
		}

		okCalled := false
		errCalled := false
		var receivedValue int
		var receivedErr error

		r.Match(
			func(v int) {
				okCalled = true
				receivedValue = v
			},
			func(e error) {
				errCalled = true
				receivedErr = e
			},
		)

		if isOk {
			// Property: Match calls onOk for Ok results
			if !okCalled {
				t.Fatal("Match should call onOk for Ok result")
			}
			if errCalled {
				t.Fatal("Match should not call onErr for Ok result")
			}
			if receivedValue != value {
				t.Fatalf("Match should pass correct value: expected %d, got %d", value, receivedValue)
			}
		} else {
			// Property: Match calls onErr for Err results
			if okCalled {
				t.Fatal("Match should not call onOk for Err result")
			}
			if !errCalled {
				t.Fatal("Match should call onErr for Err result")
			}
			if receivedErr == nil {
				t.Fatal("Match should pass the error to onErr")
			}
		}
	})
}

// TestOptionMapFlatMapCorrectness tests Option Map/FlatMap operations
func TestOptionMapFlatMapCorrectness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		isSome := rapid.Bool().Draw(t, "isSome")

		var o authplatform.Option[int]
		if isSome {
			o = authplatform.Some(value)
		} else {
			o = authplatform.None[int]()
		}

		// Test MapOption
		doubled := authplatform.MapOption(o, func(v int) int { return v * 2 })

		if isSome {
			// Property: MapOption on Some transforms the value
			if !doubled.IsSome() {
				t.Fatal("MapOption on Some should return Some")
			}
			if doubled.Unwrap() != value*2 {
				t.Fatalf("MapOption should transform value: expected %d, got %d", value*2, doubled.Unwrap())
			}
		} else {
			// Property: MapOption on None returns None
			if !doubled.IsNone() {
				t.Fatal("MapOption on None should return None")
			}
		}

		// Test FlatMapOption
		flatMapped := authplatform.FlatMapOption(o, func(v int) authplatform.Option[string] {
			if v > 0 {
				return authplatform.Some("positive")
			}
			return authplatform.None[string]()
		})

		if isSome {
			if value > 0 {
				if !flatMapped.IsSome() {
					t.Fatal("FlatMapOption should return Some for positive value")
				}
			} else {
				if !flatMapped.IsNone() {
					t.Fatal("FlatMapOption should return None for non-positive value")
				}
			}
		} else {
			if !flatMapped.IsNone() {
				t.Fatal("FlatMapOption on None should return None")
			}
		}
	})
}

// TestResultOptionConversion tests conversion between Result and Option
func TestResultOptionConversion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")

		// Test OkOr: Option to Result
		someOpt := authplatform.Some(value)
		noneOpt := authplatform.None[int]()
		testErr := errors.New("test error")

		someResult := authplatform.OkOr(someOpt, testErr)
		noneResult := authplatform.OkOr(noneOpt, testErr)

		// Property: OkOr on Some returns Ok
		if !someResult.IsOk() {
			t.Fatal("OkOr on Some should return Ok")
		}
		if someResult.Unwrap() != value {
			t.Fatal("OkOr should preserve the value")
		}

		// Property: OkOr on None returns Err
		if !noneResult.IsErr() {
			t.Fatal("OkOr on None should return Err")
		}
		if noneResult.Error() != testErr {
			t.Fatal("OkOr should use the provided error")
		}

		// Test ToOption: Result to Option
		okResult := authplatform.Ok(value)
		errResult := authplatform.Err[int](testErr)

		okOption := authplatform.ToOption(okResult)
		errOption := authplatform.ToOption(errResult)

		// Property: ToOption on Ok returns Some
		if !okOption.IsSome() {
			t.Fatal("ToOption on Ok should return Some")
		}
		if okOption.Unwrap() != value {
			t.Fatal("ToOption should preserve the value")
		}

		// Property: ToOption on Err returns None
		if !errOption.IsNone() {
			t.Fatal("ToOption on Err should return None")
		}
	})
}

// TestResultUnwrapOrDefault tests UnwrapOr behavior
func TestResultUnwrapOrDefault(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.Int().Draw(t, "value")
		defaultValue := rapid.Int().Draw(t, "defaultValue")
		isOk := rapid.Bool().Draw(t, "isOk")

		var r authplatform.Result[int]
		if isOk {
			r = authplatform.Ok(value)
		} else {
			r = authplatform.Err[int](errors.New("test error"))
		}

		result := r.UnwrapOr(defaultValue)

		if isOk {
			// Property: UnwrapOr on Ok returns the value
			if result != value {
				t.Fatalf("UnwrapOr on Ok should return value: expected %d, got %d", value, result)
			}
		} else {
			// Property: UnwrapOr on Err returns the default
			if result != defaultValue {
				t.Fatalf("UnwrapOr on Err should return default: expected %d, got %d", defaultValue, result)
			}
		}
	})
}
