package validator

import "testing"

func TestValidator(t *testing.T) {
	t.Run("Rule validates", func(t *testing.T) {
		v := New[int]().Rule("positive", func(x int) bool { return x > 0 }, "must be positive")

		result := v.Validate(5)
		if !result.IsValid() {
			t.Error("expected valid")
		}

		result = v.Validate(-5)
		if result.IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("Multiple rules", func(t *testing.T) {
		v := New[int]().
			Rule("positive", func(x int) bool { return x > 0 }, "must be positive").
			Rule("even", func(x int) bool { return x%2 == 0 }, "must be even")

		result := v.Validate(4)
		if !result.IsValid() {
			t.Error("expected valid")
		}

		result = v.Validate(3)
		if result.IsValid() {
			t.Error("expected invalid")
		}
		if len(result.Errors) != 1 {
			t.Errorf("expected 1 error, got %d", len(result.Errors))
		}

		result = v.Validate(-3)
		if len(result.Errors) != 2 {
			t.Errorf("expected 2 errors, got %d", len(result.Errors))
		}
	})

	t.Run("And combines validators", func(t *testing.T) {
		v1 := New[int]().Rule("positive", func(x int) bool { return x > 0 }, "must be positive")
		v2 := New[int]().Rule("small", func(x int) bool { return x < 100 }, "must be small")

		v1.And(v2)
		result := v1.Validate(50)
		if !result.IsValid() {
			t.Error("expected valid")
		}

		result = v1.Validate(150)
		if result.IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("Messages returns all messages", func(t *testing.T) {
		v := New[int]().
			Rule("r1", func(x int) bool { return false }, "error 1").
			Rule("r2", func(x int) bool { return false }, "error 2")

		result := v.Validate(0)
		msgs := result.Messages()
		if len(msgs) != 2 {
			t.Errorf("expected 2 messages, got %d", len(msgs))
		}
	})
}

func TestFieldValidator(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	t.Run("Field validates nested field", func(t *testing.T) {
		nameValidator := MinLength(2)
		ageValidator := Min(0)

		personValidator := New[Person]()
		Field(personValidator, "name", func(p Person) string { return p.Name }, nameValidator)
		Field(personValidator, "age", func(p Person) int { return p.Age }, ageValidator)

		result := personValidator.Validate(Person{Name: "Al", Age: 25})
		if !result.IsValid() {
			t.Error("expected valid")
		}

		result = personValidator.Validate(Person{Name: "A", Age: 25})
		if result.IsValid() {
			t.Error("expected invalid")
		}
	})
}

func TestForEach(t *testing.T) {
	itemValidator := Min(0)
	sliceValidator := ForEach(itemValidator)

	result := sliceValidator.Validate([]int{1, 2, 3})
	if !result.IsValid() {
		t.Error("expected valid")
	}

	result = sliceValidator.Validate([]int{1, -2, 3})
	if result.IsValid() {
		t.Error("expected invalid")
	}
}

func TestBuiltInValidators(t *testing.T) {
	t.Run("Required", func(t *testing.T) {
		v := Required[string]()
		if !v.Validate("hello").IsValid() {
			t.Error("expected valid")
		}
		if v.Validate("").IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("MinLength", func(t *testing.T) {
		v := MinLength(3)
		if !v.Validate("hello").IsValid() {
			t.Error("expected valid")
		}
		if v.Validate("hi").IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("MaxLength", func(t *testing.T) {
		v := MaxLength(5)
		if !v.Validate("hello").IsValid() {
			t.Error("expected valid")
		}
		if v.Validate("hello world").IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("Min", func(t *testing.T) {
		v := Min(10)
		if !v.Validate(15).IsValid() {
			t.Error("expected valid")
		}
		if v.Validate(5).IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("Max", func(t *testing.T) {
		v := Max(10)
		if !v.Validate(5).IsValid() {
			t.Error("expected valid")
		}
		if v.Validate(15).IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("Range", func(t *testing.T) {
		v := Range(1, 10)
		if !v.Validate(5).IsValid() {
			t.Error("expected valid")
		}
		if v.Validate(15).IsValid() {
			t.Error("expected invalid")
		}
	})

	t.Run("OneOf", func(t *testing.T) {
		v := OneOf("a", "b", "c")
		if !v.Validate("b").IsValid() {
			t.Error("expected valid")
		}
		if v.Validate("d").IsValid() {
			t.Error("expected invalid")
		}
	})
}
