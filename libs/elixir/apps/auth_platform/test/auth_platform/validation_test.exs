defmodule AuthPlatform.ValidationTest do
  @moduledoc """
  Property and unit tests for Validation module.

  **Property 6: Validation Error Accumulation**
  **Property 7: Validator Composition**
  **Property 8: String Validator Correctness**
  **Property 9: Numeric Validator Correctness**
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Validation

  doctest AuthPlatform.Validation

  # ============================================================================
  # Property Tests
  # ============================================================================

  describe "Property 6: Validation Error Accumulation" do
    @tag property: true
    @tag validates: "Requirements 3.6"
    property "validate_all accumulates all errors from N failing results" do
      check all n <- integer(1..5),
                ok_count <- integer(0..3) do
        # Create N error results
        error_results = for i <- 1..n, do: {:errors, [{"field#{i}", "error#{i}"}]}

        # Create some ok results
        ok_results = for i <- 1..ok_count, do: {:ok, i}

        # Combine them
        all_results = ok_results ++ error_results

        result = Validation.validate_all(all_results)

        # Should be errors with at least N errors
        assert {:errors, errors} = result
        assert length(errors) >= n
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.6"
    property "validate_all returns ok with all values when no errors" do
      check all values <- list_of(term(), min_length: 1, max_length: 10) do
        ok_results = Enum.map(values, &{:ok, &1})

        result = Validation.validate_all(ok_results)

        assert {:ok, collected} = result
        assert collected == values
      end
    end
  end

  describe "Property 7: Validator Composition" do
    @tag property: true
    @tag validates: "Requirements 3.5"
    property "all(validators) returns ok iff all validators pass" do
      check all n <- integer(1..100) do
        # All validators should pass for positive numbers
        validators = [
          Validation.positive(),
          Validation.min(1),
          Validation.max(1000)
        ]

        composed = Validation.all(validators)
        result = composed.(n)

        # All should pass for n in 1..100
        assert {:ok, ^n} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.5"
    property "all(validators) returns errors if any validator fails" do
      check all n <- integer(-100..-1) do
        validators = [
          Validation.positive(),
          Validation.min(1)
        ]

        composed = Validation.all(validators)
        result = composed.(n)

        # Should fail for negative numbers
        assert {:errors, errors} = result
        assert length(errors) >= 1
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.5"
    property "any(validators) returns ok if at least one validator passes" do
      check all n <- integer(1..100) do
        validators = [
          Validation.one_of([999]),
          Validation.positive()
        ]

        composed = Validation.any(validators)
        result = composed.(n)

        # positive() should pass
        assert {:ok, ^n} = result
      end
    end
  end

  describe "Property 8: String Validator Correctness" do
    @tag property: true
    @tag validates: "Requirements 3.2"
    property "required() returns ok for non-empty strings" do
      check all s <- string(:alphanumeric, min_length: 1) do
        result = Validation.required().(s)
        assert {:ok, ^s} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.2"
    property "required() returns error for empty string and nil" do
      assert {:errors, _} = Validation.required().("")
      assert {:errors, _} = Validation.required().(nil)
    end

    @tag property: true
    @tag validates: "Requirements 3.2"
    property "min_length(n) returns ok for strings with length >= n" do
      check all min <- integer(1..10),
                extra <- integer(0..20) do
        s = String.duplicate("a", min + extra)
        result = Validation.min_length(min).(s)
        assert {:ok, ^s} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.2"
    property "min_length(n) returns error for strings with length < n" do
      check all min <- integer(2..10) do
        s = String.duplicate("a", min - 1)
        result = Validation.min_length(min).(s)
        assert {:errors, _} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.2"
    property "max_length(n) returns ok for strings with length <= n" do
      check all max <- integer(1..20),
                len <- integer(0..max) do
        s = String.duplicate("a", len)
        result = Validation.max_length(max).(s)
        assert {:ok, ^s} = result
      end
    end
  end

  describe "Property 9: Numeric Validator Correctness" do
    @tag property: true
    @tag validates: "Requirements 3.3"
    property "positive() returns ok for n > 0" do
      check all n <- positive_integer() do
        result = Validation.positive().(n)
        assert {:ok, ^n} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.3"
    property "positive() returns error for n <= 0" do
      check all n <- integer(-100..0) do
        result = Validation.positive().(n)
        assert {:errors, _} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.3"
    property "in_range(min, max) returns ok for min <= n <= max" do
      check all min <- integer(-100..0),
                max <- integer(1..100),
                n <- integer(min..max) do
        result = Validation.in_range(min, max).(n)
        assert {:ok, ^n} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.3"
    property "in_range(min, max) returns error for n outside range" do
      check all min <- integer(0..10),
                max <- integer(11..20),
                n <- one_of([integer(-100..(min - 1)), integer((max + 1)..100)]) do
        result = Validation.in_range(min, max).(n)
        assert {:errors, _} = result
      end
    end

    @tag property: true
    @tag validates: "Requirements 3.3"
    property "non_negative() returns ok for n >= 0" do
      check all n <- non_negative_integer() do
        result = Validation.non_negative().(n)
        assert {:ok, ^n} = result
      end
    end
  end

  # ============================================================================
  # Unit Tests
  # ============================================================================

  describe "validate_field/3" do
    test "returns ok when all validators pass" do
      result =
        Validation.validate_field("name", "John", [
          Validation.required(),
          Validation.min_length(2)
        ])

      assert {:ok, "John"} = result
    end

    test "returns errors with field name" do
      result =
        Validation.validate_field("name", "", [
          Validation.required()
        ])

      assert {:errors, [{"name", "is required"}]} = result
    end

    test "accumulates multiple errors" do
      result =
        Validation.validate_field("name", "a", [
          Validation.min_length(5),
          Validation.max_length(0)
        ])

      assert {:errors, errors} = result
      assert length(errors) == 2
    end
  end

  describe "matches_regex/1" do
    test "returns ok for matching string" do
      validator = Validation.matches_regex(~r/^[a-z]+$/)
      assert {:ok, "hello"} = validator.("hello")
    end

    test "returns error for non-matching string" do
      validator = Validation.matches_regex(~r/^[a-z]+$/)
      assert {:errors, _} = validator.("Hello123")
    end
  end

  describe "one_of/1" do
    test "returns ok for allowed value" do
      validator = Validation.one_of(["a", "b", "c"])
      assert {:ok, "b"} = validator.("b")
    end

    test "returns error for disallowed value" do
      validator = Validation.one_of(["a", "b", "c"])
      assert {:errors, _} = validator.("d")
    end
  end

  describe "collection validators" do
    test "min_size/1 validates minimum size" do
      assert {:ok, [1, 2, 3]} = Validation.min_size(2).([1, 2, 3])
      assert {:errors, _} = Validation.min_size(5).([1, 2])
    end

    test "max_size/1 validates maximum size" do
      assert {:ok, [1, 2]} = Validation.max_size(3).([1, 2])
      assert {:errors, _} = Validation.max_size(2).([1, 2, 3])
    end

    test "unique_elements/0 validates uniqueness" do
      assert {:ok, [1, 2, 3]} = Validation.unique_elements().([1, 2, 3])
      assert {:errors, _} = Validation.unique_elements().([1, 2, 2])
    end
  end

  describe "not_/1" do
    test "negates validator" do
      validator = Validation.not_(Validation.one_of(["admin"]))
      assert {:ok, "user"} = validator.("user")
      assert {:errors, _} = validator.("admin")
    end
  end
end
