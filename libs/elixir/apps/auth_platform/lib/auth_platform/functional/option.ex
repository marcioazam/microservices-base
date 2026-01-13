defmodule AuthPlatform.Functional.Option do
  @moduledoc """
  Type-safe optional value handling.

  The Option type represents a value that may or may not be present.
  It uses `{:some, value}` for present values and `:none` for absent values.

  This is an alternative to using `nil` that makes the absence of a value explicit
  and provides safe operations for working with optional values.

  ## Usage

      alias AuthPlatform.Functional.Option

      # Creating options
      Option.some(42)           # {:some, 42}
      Option.none()             # :none
      Option.from_nullable(nil) # :none
      Option.from_nullable(42)  # {:some, 42}

      # Transforming options
      {:some, 5}
      |> Option.map(&(&1 * 2))     # {:some, 10}
      |> Option.flat_map(&safe_divide(100, &1))

      # Pattern matching with match/3
      option
      |> Option.match(
        fn value -> "Got: \#{value}" end,
        fn -> "Nothing" end
      )

      # Unwrapping
      Option.unwrap!({:some, 42})        # 42
      Option.unwrap_or({:none}, 0)       # 0

  """

  @type t(a) :: {:some, a} | :none
  @type t() :: t(any())

  # ============================================================================
  # Constructors
  # ============================================================================

  @doc """
  Creates an option containing the given value.

  ## Examples

      iex> AuthPlatform.Functional.Option.some(42)
      {:some, 42}

      iex> AuthPlatform.Functional.Option.some("hello")
      {:some, "hello"}

  """
  @spec some(value) :: {:some, value} when value: any()
  def some(value), do: {:some, value}

  @doc """
  Creates an empty option (no value present).

  ## Examples

      iex> AuthPlatform.Functional.Option.none()
      :none

  """
  @spec none() :: :none
  def none, do: :none

  @doc """
  Creates an option from a nullable value.

  Returns `:none` if the value is `nil`, otherwise `{:some, value}`.

  ## Examples

      iex> AuthPlatform.Functional.Option.from_nullable(nil)
      :none

      iex> AuthPlatform.Functional.Option.from_nullable(42)
      {:some, 42}

      iex> AuthPlatform.Functional.Option.from_nullable(false)
      {:some, false}

  """
  @spec from_nullable(value | nil) :: t(value) when value: any()
  def from_nullable(nil), do: :none
  def from_nullable(value), do: {:some, value}

  # ============================================================================
  # Transformations
  # ============================================================================

  @doc """
  Applies a function to the value inside a some option.

  If the option is none, returns none unchanged.

  ## Examples

      iex> AuthPlatform.Functional.Option.map({:some, 5}, &(&1 * 2))
      {:some, 10}

      iex> AuthPlatform.Functional.Option.map(:none, &(&1 * 2))
      :none

  """
  @spec map(t(a), (a -> b)) :: t(b) when a: any(), b: any()
  def map({:some, value}, fun) when is_function(fun, 1), do: {:some, fun.(value)}
  def map(:none, _fun), do: :none

  @doc """
  Applies a function that returns an option to the value inside a some option.

  Also known as `bind` or `chain`. Useful for sequencing operations that may return none.

  ## Examples

      iex> safe_head = fn
      ...>   [] -> :none
      ...>   [h | _] -> {:some, h}
      ...> end
      iex> AuthPlatform.Functional.Option.flat_map({:some, [1, 2, 3]}, safe_head)
      {:some, 1}

      iex> AuthPlatform.Functional.Option.flat_map(:none, fn _ -> {:some, 1} end)
      :none

  """
  @spec flat_map(t(a), (a -> t(b))) :: t(b) when a: any(), b: any()
  def flat_map({:some, value}, fun) when is_function(fun, 1), do: fun.(value)
  def flat_map(:none, _fun), do: :none

  @doc """
  Pattern matches on an option, applying the appropriate function.

  ## Examples

      iex> AuthPlatform.Functional.Option.match(
      ...>   {:some, 42},
      ...>   fn v -> "Got \#{v}" end,
      ...>   fn -> "Nothing" end
      ...> )
      "Got 42"

      iex> AuthPlatform.Functional.Option.match(
      ...>   :none,
      ...>   fn v -> "Got \#{v}" end,
      ...>   fn -> "Nothing" end
      ...> )
      "Nothing"

  """
  @spec match(t(a), (a -> b), (() -> b)) :: b when a: any(), b: any()
  def match({:some, value}, on_some, _on_none) when is_function(on_some, 1), do: on_some.(value)
  def match(:none, _on_some, on_none) when is_function(on_none, 0), do: on_none.()

  @doc """
  Filters an option based on a predicate.

  Returns the option unchanged if it's some and the predicate returns true.
  Returns none if the option is none or the predicate returns false.

  ## Examples

      iex> AuthPlatform.Functional.Option.filter({:some, 5}, &(&1 > 3))
      {:some, 5}

      iex> AuthPlatform.Functional.Option.filter({:some, 2}, &(&1 > 3))
      :none

      iex> AuthPlatform.Functional.Option.filter(:none, &(&1 > 3))
      :none

  """
  @spec filter(t(a), (a -> boolean())) :: t(a) when a: any()
  def filter({:some, value}, predicate) when is_function(predicate, 1) do
    if predicate.(value), do: {:some, value}, else: :none
  end

  def filter(:none, _predicate), do: :none

  # ============================================================================
  # Unwrapping
  # ============================================================================

  @doc """
  Extracts the value from a some option, raising if it's none.

  ## Examples

      iex> AuthPlatform.Functional.Option.unwrap!({:some, 42})
      42

  ## Raises

      AuthPlatform.Functional.Option.unwrap!(:none)
      # ** (RuntimeError) Unwrap on none

  """
  @spec unwrap!(t(a)) :: a when a: any()
  def unwrap!({:some, value}), do: value
  def unwrap!(:none), do: raise("Unwrap on none")

  @doc """
  Extracts the value from a some option, or returns the default if it's none.

  ## Examples

      iex> AuthPlatform.Functional.Option.unwrap_or({:some, 42}, 0)
      42

      iex> AuthPlatform.Functional.Option.unwrap_or(:none, 0)
      0

  """
  @spec unwrap_or(t(a), a) :: a when a: any()
  def unwrap_or({:some, value}, _default), do: value
  def unwrap_or(:none, default), do: default

  @doc """
  Extracts the value from a some option, or computes a default using the given function.

  ## Examples

      iex> AuthPlatform.Functional.Option.unwrap_or_else({:some, 42}, fn -> 0 end)
      42

      iex> AuthPlatform.Functional.Option.unwrap_or_else(:none, fn -> 0 end)
      0

  """
  @spec unwrap_or_else(t(a), (() -> a)) :: a when a: any()
  def unwrap_or_else({:some, value}, _fun), do: value
  def unwrap_or_else(:none, fun) when is_function(fun, 0), do: fun.()

  # ============================================================================
  # Predicates
  # ============================================================================

  @doc """
  Returns true if the option is some.

  ## Examples

      iex> AuthPlatform.Functional.Option.is_some?({:some, 42})
      true

      iex> AuthPlatform.Functional.Option.is_some?(:none)
      false

  """
  @spec is_some?(t(any())) :: boolean()
  def is_some?({:some, _}), do: true
  def is_some?(:none), do: false

  @doc """
  Returns true if the option is none.

  ## Examples

      iex> AuthPlatform.Functional.Option.is_none?(:none)
      true

      iex> AuthPlatform.Functional.Option.is_none?({:some, 42})
      false

  """
  @spec is_none?(t(any())) :: boolean()
  def is_none?(opt), do: not is_some?(opt)

  # ============================================================================
  # Utilities
  # ============================================================================

  @doc """
  Converts an option to a result.

  Returns `{:ok, value}` for some, or `{:error, error}` for none.

  ## Examples

      iex> AuthPlatform.Functional.Option.to_result({:some, 42}, :not_found)
      {:ok, 42}

      iex> AuthPlatform.Functional.Option.to_result(:none, :not_found)
      {:error, :not_found}

  """
  @spec to_result(t(a), e) :: AuthPlatform.Functional.Result.t(a, e) when a: any(), e: any()
  def to_result({:some, value}, _error), do: {:ok, value}
  def to_result(:none, error), do: {:error, error}

  @doc """
  Returns the first some option, or none if all are none.

  ## Examples

      iex> AuthPlatform.Functional.Option.or_else({:some, 1}, {:some, 2})
      {:some, 1}

      iex> AuthPlatform.Functional.Option.or_else(:none, {:some, 2})
      {:some, 2}

      iex> AuthPlatform.Functional.Option.or_else(:none, :none)
      :none

  """
  @spec or_else(t(a), t(a)) :: t(a) when a: any()
  def or_else({:some, _} = opt, _other), do: opt
  def or_else(:none, other), do: other

  @doc """
  Zips two options together.

  Returns some with a tuple if both are some, otherwise none.

  ## Examples

      iex> AuthPlatform.Functional.Option.zip({:some, 1}, {:some, 2})
      {:some, {1, 2}}

      iex> AuthPlatform.Functional.Option.zip({:some, 1}, :none)
      :none

      iex> AuthPlatform.Functional.Option.zip(:none, {:some, 2})
      :none

  """
  @spec zip(t(a), t(b)) :: t({a, b}) when a: any(), b: any()
  def zip({:some, a}, {:some, b}), do: {:some, {a, b}}
  def zip(_, _), do: :none
end
