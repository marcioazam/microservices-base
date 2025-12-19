package validated

import (
	"errors"
	"testing"
)

func TestValidatedBasicOperations(t *testing.T) {
	t.Run("Valid creates valid result", func(t *testing.T) {
		v := Valid[string, int](42)
		if !v.IsValid() {
			t.Error("expected valid")
		}
		if v.GetValue() != 42 {
			t.Errorf("expected 42, got %d", v.GetValue())
		}
	})

	t.Run("Invalid creates invalid result", func(t *testing.T) {
		v := Invalid[string, int]("error1", "error2")
		if !v.IsInvalid() {
			t.Error("expected invalid")
		}
		if len(v.GetErrors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(v.GetErrors()))
		}
	})

	t.Run("GetValue panics on invalid", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic")
			}
		}()
		v := Invalid[string, int]("error")
		v.GetValue()
	})
}

func TestValidatedMap(t *testing.T) {
	t.Run("Map applies function to valid", func(t *testing.T) {
		v := Valid[string, int](21)
		mapped := Map(v, func(x int) int { return x * 2 })
		if !mapped.IsValid() || mapped.GetValue() != 42 {
			t.Error("expected 42")
		}
	})

	t.Run("Map preserves errors on invalid", func(t *testing.T) {
		v := Invalid[string, int]("error")
		mapped := Map(v, func(x int) int { return x * 2 })
		if !mapped.IsInvalid() {
			t.Error("expected invalid")
		}
		if len(mapped.GetErrors()) != 1 {
			t.Error("expected 1 error")
		}
	})
}

func TestValidatedCombine(t *testing.T) {
	t.Run("Combine valid values", func(t *testing.T) {
		v1 := Valid[string, int](10)
		v2 := Valid[string, int](20)
		combined := Combine(v1, v2, func(a, b int) int { return a + b })
		if !combined.IsValid() || combined.GetValue() != 30 {
			t.Error("expected 30")
		}
	})

	t.Run("Combine accumulates errors", func(t *testing.T) {
		v1 := Invalid[string, int]("error1")
		v2 := Invalid[string, int]("error2")
		combined := Combine(v1, v2, func(a, b int) int { return a + b })
		if !combined.IsInvalid() {
			t.Error("expected invalid")
		}
		if len(combined.GetErrors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(combined.GetErrors()))
		}
	})

	t.Run("Combine with one invalid", func(t *testing.T) {
		v1 := Valid[string, int](10)
		v2 := Invalid[string, int]("error")
		combined := Combine(v1, v2, func(a, b int) int { return a + b })
		if !combined.IsInvalid() {
			t.Error("expected invalid")
		}
	})
}

func TestValidatedCombine3(t *testing.T) {
	t.Run("Combine3 valid values", func(t *testing.T) {
		v1 := Valid[string, int](1)
		v2 := Valid[string, int](2)
		v3 := Valid[string, int](3)
		combined := Combine3(v1, v2, v3, func(a, b, c int) int { return a + b + c })
		if !combined.IsValid() || combined.GetValue() != 6 {
			t.Error("expected 6")
		}
	})

	t.Run("Combine3 accumulates all errors", func(t *testing.T) {
		v1 := Invalid[string, int]("e1")
		v2 := Invalid[string, int]("e2")
		v3 := Invalid[string, int]("e3")
		combined := Combine3(v1, v2, v3, func(a, b, c int) int { return a + b + c })
		if len(combined.GetErrors()) != 3 {
			t.Errorf("expected 3 errors, got %d", len(combined.GetErrors()))
		}
	})
}

func TestValidatedSequence(t *testing.T) {
	t.Run("Sequence all valid", func(t *testing.T) {
		vs := []Validated[string, int]{
			Valid[string, int](1),
			Valid[string, int](2),
			Valid[string, int](3),
		}
		result := Sequence(vs)
		if !result.IsValid() {
			t.Error("expected valid")
		}
		values := result.GetValue()
		if len(values) != 3 || values[0] != 1 || values[1] != 2 || values[2] != 3 {
			t.Error("unexpected values")
		}
	})

	t.Run("Sequence accumulates errors", func(t *testing.T) {
		vs := []Validated[string, int]{
			Valid[string, int](1),
			Invalid[string, int]("e1"),
			Invalid[string, int]("e2"),
		}
		result := Sequence(vs)
		if !result.IsInvalid() {
			t.Error("expected invalid")
		}
		if len(result.GetErrors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result.GetErrors()))
		}
	})
}

func TestValidatedTraverse(t *testing.T) {
	validate := func(x int) Validated[string, int] {
		if x < 0 {
			return Invalid[string, int]("negative")
		}
		return Valid[string, int](x * 2)
	}

	t.Run("Traverse all valid", func(t *testing.T) {
		result := Traverse([]int{1, 2, 3}, validate)
		if !result.IsValid() {
			t.Error("expected valid")
		}
		values := result.GetValue()
		if values[0] != 2 || values[1] != 4 || values[2] != 6 {
			t.Error("unexpected values")
		}
	})

	t.Run("Traverse with invalid", func(t *testing.T) {
		result := Traverse([]int{1, -2, -3}, validate)
		if !result.IsInvalid() {
			t.Error("expected invalid")
		}
		if len(result.GetErrors()) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result.GetErrors()))
		}
	})
}

func TestValidatedGetOrElse(t *testing.T) {
	t.Run("GetOrElse returns value when valid", func(t *testing.T) {
		v := Valid[string, int](42)
		if v.GetOrElse(0) != 42 {
			t.Error("expected 42")
		}
	})

	t.Run("GetOrElse returns default when invalid", func(t *testing.T) {
		v := Invalid[string, int]("error")
		if v.GetOrElse(99) != 99 {
			t.Error("expected 99")
		}
	})
}

func TestValidatedFold(t *testing.T) {
	t.Run("Fold calls onValid for valid", func(t *testing.T) {
		v := Valid[string, int](42)
		result := Fold(v,
			func(errs []string) string { return "invalid" },
			func(val int) string { return "valid" },
		)
		if result != "valid" {
			t.Error("expected valid")
		}
	})

	t.Run("Fold calls onInvalid for invalid", func(t *testing.T) {
		v := Invalid[string, int]("error")
		result := Fold(v,
			func(errs []string) string { return "invalid" },
			func(val int) string { return "valid" },
		)
		if result != "invalid" {
			t.Error("expected invalid")
		}
	})
}

func TestValidatedToResult(t *testing.T) {
	t.Run("ToResult converts valid", func(t *testing.T) {
		v := Valid[error, int](42)
		r := ToResult(v)
		if !r.IsOk() || r.Unwrap() != 42 {
			t.Error("expected Ok(42)")
		}
	})

	t.Run("ToResult converts invalid", func(t *testing.T) {
		err := errors.New("test error")
		v := Invalid[error, int](err)
		r := ToResult(v)
		if !r.IsErr() {
			t.Error("expected Err")
		}
	})
}
