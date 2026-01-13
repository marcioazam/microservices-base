defmodule AuthPlatform.Validation do
  @moduledoc """
  Composable validation with error accumulation.

  This module provides a functional approach to validation that:
  - Accumulates all errors instead of failing on the first one
  - Composes validators using `all/1`, `any/1`, and `not_/1`
  - Tracks field paths for nested validation

  ## Usage

      alias AuthPlatform.Validation

      # Single field validation
      Validation.validate_field("email", "test@example.com", [
        Validation.required(),
        Validation.matches_regex(~r/@/)
      ])

      # Multiple field validation with accumulation
      Validation.validate_all([
        Validation.validate_field("name", params["name"], [Validation.required()]),
        Validation.validate_field("age", params["age"], [Validation.positive()])
      ])

      # Composing validators
      age_validator = Validation.all([
        Validation.positive(),
        Validation.in_range(0, 150)
      ])

  ## Validation Result

  All validators return `{:ok, value}` on success or `{:errors, [{field, message}]}` on failure.

  """

  @type validation_error :: {String.t(), String.t()}
  @type validation_result(a) :: {:ok, a} | {:errors, [validation_error()]}
  @type validator(a) :: (a -> validation_result(a))

  # ============================================================================
  # Core Validation Functions
  # ============================================================================

  @doc """
  Validates all results and accumulates errors.

  Returns `{:ok, [values]}` if all validations pass, or `{:errors, all_errors}` if any fail.

  ## Examples

      iex> AuthPlatform.Validation.validate_all([{:ok, 1}, {:ok, 2}])
      {:ok, [1, 2]}

      iex> AuthPlatform.Validation.validate_all([{:ok, 1}, {:errors, [{"x", "bad"}]}])
      {:errors, [{"x", "bad"}]}

  """
  @spec validate_all([validation_result(any())]) :: validation_result([any()])
  def validate_all(results) when is_list(results) do
    {values, errors} =
      Enum.reduce(results, {[], []}, fn
        {:ok, value}, {vals, errs} -> {[value | vals], errs}
        {:errors, errs}, {vals, acc_errs} -> {vals, errs ++ acc_errs}
      end)

    case errors do
      [] -> {:ok, Enum.reverse(values)}
      _ -> {:errors, errors}
    end
  end

  @doc """
  Validates a field with a list of validators.

  Returns `{:ok, value}` if all validators pass, or `{:errors, [{field, message}]}` with all failures.

  ## Examples

      iex> AuthPlatform.Validation.validate_field("name", "John", [
      ...>   AuthPlatform.Validation.required(),
      ...>   AuthPlatform.Validation.min_length(2)
      ...> ])
      {:ok, "John"}

      iex> AuthPlatform.Validation.validate_field("name", "", [
      ...>   AuthPlatform.Validation.required()
      ...> ])
      {:errors, [{"name", "is required"}]}

  """
  @spec validate_field(String.t(), any(), [validator(any())]) :: validation_result(any())
  def validate_field(field, value, validators) when is_binary(field) and is_list(validators) do
    errors =
      validators
      |> Enum.flat_map(fn validator ->
        case validator.(value) do
          {:ok, _} -> []
          {:errors, errs} -> Enum.map(errs, fn {_, msg} -> {field, msg} end)
        end
      end)

    case errors do
      [] -> {:ok, value}
      _ -> {:errors, errors}
    end
  end

  # ============================================================================
  # String Validators
  # ============================================================================

  @doc """
  Validates that a value is present (not nil or empty string).

  ## Examples

      iex> AuthPlatform.Validation.required().("hello")
      {:ok, "hello"}

      iex> AuthPlatform.Validation.required().("")
      {:errors, [{"", "is required"}]}

      iex> AuthPlatform.Validation.required().(nil)
      {:errors, [{"", "is required"}]}

  """
  @spec required() :: validator(any())
  def required do
    fn
      nil -> {:errors, [{"", "is required"}]}
      "" -> {:errors, [{"", "is required"}]}
      value when is_binary(value) -> {:ok, value}
      value -> {:ok, value}
    end
  end

  @doc """
  Validates that a string has at least `min` characters.

  ## Examples

      iex> AuthPlatform.Validation.min_length(3).("hello")
      {:ok, "hello"}

      iex> AuthPlatform.Validation.min_length(3).("hi")
      {:errors, [{"", "must be at least 3 characters"}]}

  """
  @spec min_length(pos_integer()) :: validator(String.t())
  def min_length(min) when is_integer(min) and min > 0 do
    fn value when is_binary(value) ->
      if String.length(value) >= min,
        do: {:ok, value},
        else: {:errors, [{"", "must be at least #{min} characters"}]}
    end
  end

  @doc """
  Validates that a string has at most `max` characters.

  ## Examples

      iex> AuthPlatform.Validation.max_length(5).("hi")
      {:ok, "hi"}

      iex> AuthPlatform.Validation.max_length(3).("hello")
      {:errors, [{"", "must be at most 3 characters"}]}

  """
  @spec max_length(pos_integer()) :: validator(String.t())
  def max_length(max) when is_integer(max) and max > 0 do
    fn value when is_binary(value) ->
      if String.length(value) <= max,
        do: {:ok, value},
        else: {:errors, [{"", "must be at most #{max} characters"}]}
    end
  end

  @doc """
  Validates that a string matches a regex pattern.

  ## Examples

      iex> AuthPlatform.Validation.matches_regex(~r/^[a-z]+$/).("hello")
      {:ok, "hello"}

      iex> AuthPlatform.Validation.matches_regex(~r/^[a-z]+$/).("Hello123")
      {:errors, [{"", "does not match required format"}]}

  """
  @spec matches_regex(Regex.t()) :: validator(String.t())
  def matches_regex(%Regex{} = regex) do
    fn value when is_binary(value) ->
      if Regex.match?(regex, value),
        do: {:ok, value},
        else: {:errors, [{"", "does not match required format"}]}
    end
  end

  @doc """
  Validates that a value is one of the allowed values.

  ## Examples

      iex> AuthPlatform.Validation.one_of(["a", "b", "c"]).("b")
      {:ok, "b"}

      iex> AuthPlatform.Validation.one_of(["a", "b", "c"]).("d")
      {:errors, [{"", "must be one of: a, b, c"}]}

  """
  @spec one_of([any()]) :: validator(any())
  def one_of(allowed) when is_list(allowed) do
    fn value ->
      if value in allowed,
        do: {:ok, value},
        else: {:errors, [{"", "must be one of: #{Enum.join(allowed, ", ")}"}]}
    end
  end

  # ============================================================================
  # Numeric Validators
  # ============================================================================

  @doc """
  Validates that a number is positive (> 0).

  ## Examples

      iex> AuthPlatform.Validation.positive().(5)
      {:ok, 5}

      iex> AuthPlatform.Validation.positive().(0)
      {:errors, [{"", "must be positive"}]}

      iex> AuthPlatform.Validation.positive().(-1)
      {:errors, [{"", "must be positive"}]}

  """
  @spec positive() :: validator(number())
  def positive do
    fn value when is_number(value) ->
      if value > 0,
        do: {:ok, value},
        else: {:errors, [{"", "must be positive"}]}
    end
  end

  @doc """
  Validates that a number is non-negative (>= 0).

  ## Examples

      iex> AuthPlatform.Validation.non_negative().(0)
      {:ok, 0}

      iex> AuthPlatform.Validation.non_negative().(-1)
      {:errors, [{"", "must be non-negative"}]}

  """
  @spec non_negative() :: validator(number())
  def non_negative do
    fn value when is_number(value) ->
      if value >= 0,
        do: {:ok, value},
        else: {:errors, [{"", "must be non-negative"}]}
    end
  end

  @doc """
  Validates that a number is within a range (inclusive).

  ## Examples

      iex> AuthPlatform.Validation.in_range(1, 10).(5)
      {:ok, 5}

      iex> AuthPlatform.Validation.in_range(1, 10).(0)
      {:errors, [{"", "must be between 1 and 10"}]}

  """
  @spec in_range(number(), number()) :: validator(number())
  def in_range(min, max) when is_number(min) and is_number(max) do
    fn value when is_number(value) ->
      if value >= min and value <= max,
        do: {:ok, value},
        else: {:errors, [{"", "must be between #{min} and #{max}"}]}
    end
  end

  @doc """
  Validates that a number is at least `min`.

  ## Examples

      iex> AuthPlatform.Validation.min(5).(10)
      {:ok, 10}

      iex> AuthPlatform.Validation.min(5).(3)
      {:errors, [{"", "must be at least 5"}]}

  """
  @spec min(number()) :: validator(number())
  def min(min_val) when is_number(min_val) do
    fn value when is_number(value) ->
      if value >= min_val,
        do: {:ok, value},
        else: {:errors, [{"", "must be at least #{min_val}"}]}
    end
  end

  @doc """
  Validates that a number is at most `max`.

  ## Examples

      iex> AuthPlatform.Validation.max(10).(5)
      {:ok, 5}

      iex> AuthPlatform.Validation.max(10).(15)
      {:errors, [{"", "must be at most 10"}]}

  """
  @spec max(number()) :: validator(number())
  def max(max_val) when is_number(max_val) do
    fn value when is_number(value) ->
      if value <= max_val,
        do: {:ok, value},
        else: {:errors, [{"", "must be at most #{max_val}"}]}
    end
  end

  # ============================================================================
  # Collection Validators
  # ============================================================================

  @doc """
  Validates that a collection has at least `min` elements.

  ## Examples

      iex> AuthPlatform.Validation.min_size(2).([1, 2, 3])
      {:ok, [1, 2, 3]}

      iex> AuthPlatform.Validation.min_size(2).([1])
      {:errors, [{"", "must have at least 2 elements"}]}

  """
  @spec min_size(pos_integer()) :: validator(Enumerable.t())
  def min_size(min) when is_integer(min) and min >= 0 do
    fn value ->
      if Enum.count(value) >= min,
        do: {:ok, value},
        else: {:errors, [{"", "must have at least #{min} elements"}]}
    end
  end

  @doc """
  Validates that a collection has at most `max` elements.

  ## Examples

      iex> AuthPlatform.Validation.max_size(3).([1, 2])
      {:ok, [1, 2]}

      iex> AuthPlatform.Validation.max_size(2).([1, 2, 3])
      {:errors, [{"", "must have at most 2 elements"}]}

  """
  @spec max_size(pos_integer()) :: validator(Enumerable.t())
  def max_size(max) when is_integer(max) and max >= 0 do
    fn value ->
      if Enum.count(value) <= max,
        do: {:ok, value},
        else: {:errors, [{"", "must have at most #{max} elements"}]}
    end
  end

  @doc """
  Validates that a collection has unique elements.

  ## Examples

      iex> AuthPlatform.Validation.unique_elements().([1, 2, 3])
      {:ok, [1, 2, 3]}

      iex> AuthPlatform.Validation.unique_elements().([1, 2, 2])
      {:errors, [{"", "must have unique elements"}]}

  """
  @spec unique_elements() :: validator(Enumerable.t())
  def unique_elements do
    fn value ->
      list = Enum.to_list(value)

      if length(list) == length(Enum.uniq(list)),
        do: {:ok, value},
        else: {:errors, [{"", "must have unique elements"}]}
    end
  end

  # ============================================================================
  # Composition
  # ============================================================================

  @doc """
  Combines validators with AND logic - all must pass.

  ## Examples

      iex> validator = AuthPlatform.Validation.all([
      ...>   AuthPlatform.Validation.min_length(2),
      ...>   AuthPlatform.Validation.max_length(10)
      ...> ])
      iex> validator.("hello")
      {:ok, "hello"}

      iex> validator = AuthPlatform.Validation.all([
      ...>   AuthPlatform.Validation.min_length(10),
      ...>   AuthPlatform.Validation.max_length(5)
      ...> ])
      iex> validator.("hello")
      {:errors, [{"", "must be at least 10 characters"}]}

  """
  @spec all([validator(a)]) :: validator(a) when a: any()
  def all(validators) when is_list(validators) do
    fn value ->
      errors =
        validators
        |> Enum.flat_map(fn v ->
          case v.(value) do
            {:ok, _} -> []
            {:errors, errs} -> errs
          end
        end)

      case errors do
        [] -> {:ok, value}
        _ -> {:errors, errors}
      end
    end
  end

  @doc """
  Combines validators with OR logic - at least one must pass.

  ## Examples

      iex> validator = AuthPlatform.Validation.any([
      ...>   AuthPlatform.Validation.one_of(["a"]),
      ...>   AuthPlatform.Validation.one_of(["b"])
      ...> ])
      iex> validator.("a")
      {:ok, "a"}

      iex> validator = AuthPlatform.Validation.any([
      ...>   AuthPlatform.Validation.one_of(["a"]),
      ...>   AuthPlatform.Validation.one_of(["b"])
      ...> ])
      iex> validator.("c")
      {:errors, [{"", "none of the validators passed"}]}

  """
  @spec any([validator(a)]) :: validator(a) when a: any()
  def any(validators) when is_list(validators) do
    fn value ->
      passed =
        Enum.any?(validators, fn v ->
          case v.(value) do
            {:ok, _} -> true
            {:errors, _} -> false
          end
        end)

      if passed,
        do: {:ok, value},
        else: {:errors, [{"", "none of the validators passed"}]}
    end
  end

  @doc """
  Negates a validator - passes if the inner validator fails.

  ## Examples

      iex> validator = AuthPlatform.Validation.not_(AuthPlatform.Validation.one_of(["admin"]))
      iex> validator.("user")
      {:ok, "user"}

      iex> validator = AuthPlatform.Validation.not_(AuthPlatform.Validation.one_of(["admin"]))
      iex> validator.("admin")
      {:errors, [{"", "validation should have failed"}]}

  """
  @spec not_(validator(a)) :: validator(a) when a: any()
  def not_(validator) when is_function(validator, 1) do
    fn value ->
      case validator.(value) do
        {:ok, _} -> {:errors, [{"", "validation should have failed"}]}
        {:errors, _} -> {:ok, value}
      end
    end
  end
end
