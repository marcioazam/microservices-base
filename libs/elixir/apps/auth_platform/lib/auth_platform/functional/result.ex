defmodule AuthPlatform.Functional.Result do
  @moduledoc """
  Type-safe result handling for operations that may fail.

  The Result type represents the outcome of an operation that can either succeed
  with a value (`{:ok, value}`) or fail with an error (`{:error, reason}`).

  ## Usage

      alias AuthPlatform.Functional.Result

      # Creating results
      Result.ok(42)           # {:ok, 42}
      Result.error("failed")  # {:error, "failed"}

      # Transforming results
      {:ok, 5}
      |> Result.map(&(&1 * 2))     # {:ok, 10}
      |> Result.flat_map(&divide(100, &1))

      # Pattern matching with match/3
      result
      |> Result.match(
        fn value -> "Success: \#{value}" end,
        fn error -> "Error: \#{error}" end
      )

      # Unwrapping
      Result.unwrap!({:ok, 42})        # 42
      Result.unwrap_or({:error, _}, 0) # 0

  ## Wrapping exceptions

      Result.try_result do
        risky_operation()
      end
      # Returns {:ok, result} or {:error, exception}

  """

  @type t(ok, err) :: {:ok, ok} | {:error, err}
  @type t(ok) :: t(ok, any())
  @type t() :: t(any(), any())

  # ============================================================================
  # Constructors
  # ============================================================================

  @doc """
  Creates a successful result containing the given value.

  ## Examples

      iex> AuthPlatform.Functional.Result.ok(42)
      {:ok, 42}

      iex> AuthPlatform.Functional.Result.ok("hello")
      {:ok, "hello"}

  """
  @spec ok(value) :: {:ok, value} when value: any()
  def ok(value), do: {:ok, value}

  @doc """
  Creates a failed result containing the given error reason.

  ## Examples

      iex> AuthPlatform.Functional.Result.error("not found")
      {:error, "not found"}

      iex> AuthPlatform.Functional.Result.error(:timeout)
      {:error, :timeout}

  """
  @spec error(reason) :: {:error, reason} when reason: any()
  def error(reason), do: {:error, reason}

  # ============================================================================
  # Transformations
  # ============================================================================

  @doc """
  Applies a function to the value inside an ok result.

  If the result is an error, returns the error unchanged.

  ## Examples

      iex> AuthPlatform.Functional.Result.map({:ok, 5}, &(&1 * 2))
      {:ok, 10}

      iex> AuthPlatform.Functional.Result.map({:error, "fail"}, &(&1 * 2))
      {:error, "fail"}

  """
  @spec map(t(a, e), (a -> b)) :: t(b, e) when a: any(), b: any(), e: any()
  def map({:ok, value}, fun) when is_function(fun, 1), do: {:ok, fun.(value)}
  def map({:error, _} = err, _fun), do: err

  @doc """
  Applies a function that returns a result to the value inside an ok result.

  Also known as `bind` or `chain`. Useful for sequencing operations that may fail.

  ## Examples

      iex> divide = fn
      ...>   _, 0 -> {:error, :division_by_zero}
      ...>   a, b -> {:ok, div(a, b)}
      ...> end
      iex> AuthPlatform.Functional.Result.flat_map({:ok, 10}, &divide.(&1, 2))
      {:ok, 5}

      iex> AuthPlatform.Functional.Result.flat_map({:error, "fail"}, fn _ -> {:ok, 1} end)
      {:error, "fail"}

  """
  @spec flat_map(t(a, e), (a -> t(b, e))) :: t(b, e) when a: any(), b: any(), e: any()
  def flat_map({:ok, value}, fun) when is_function(fun, 1), do: fun.(value)
  def flat_map({:error, _} = err, _fun), do: err

  @doc """
  Pattern matches on a result, applying the appropriate function.

  ## Examples

      iex> AuthPlatform.Functional.Result.match(
      ...>   {:ok, 42},
      ...>   fn v -> "Got \#{v}" end,
      ...>   fn e -> "Error: \#{e}" end
      ...> )
      "Got 42"

      iex> AuthPlatform.Functional.Result.match(
      ...>   {:error, "oops"},
      ...>   fn v -> "Got \#{v}" end,
      ...>   fn e -> "Error: \#{e}" end
      ...> )
      "Error: oops"

  """
  @spec match(t(a, e), (a -> b), (e -> b)) :: b when a: any(), b: any(), e: any()
  def match({:ok, value}, on_ok, _on_error) when is_function(on_ok, 1), do: on_ok.(value)
  def match({:error, reason}, _on_ok, on_error) when is_function(on_error, 1), do: on_error.(reason)

  @doc """
  Applies a function to the error inside an error result.

  If the result is ok, returns the ok unchanged.

  ## Examples

      iex> AuthPlatform.Functional.Result.map_error({:error, "fail"}, &String.upcase/1)
      {:error, "FAIL"}

      iex> AuthPlatform.Functional.Result.map_error({:ok, 42}, &String.upcase/1)
      {:ok, 42}

  """
  @spec map_error(t(a, e1), (e1 -> e2)) :: t(a, e2) when a: any(), e1: any(), e2: any()
  def map_error({:ok, _} = ok, _fun), do: ok
  def map_error({:error, reason}, fun) when is_function(fun, 1), do: {:error, fun.(reason)}

  # ============================================================================
  # Unwrapping
  # ============================================================================

  @doc """
  Extracts the value from an ok result, raising if it's an error.

  ## Examples

      iex> AuthPlatform.Functional.Result.unwrap!({:ok, 42})
      42

  ## Raises

      AuthPlatform.Functional.Result.unwrap!({:error, "oops"})
      # ** (RuntimeError) Unwrap on error: "oops"

  """
  @spec unwrap!(t(a, any())) :: a when a: any()
  def unwrap!({:ok, value}), do: value
  def unwrap!({:error, reason}), do: raise("Unwrap on error: #{inspect(reason)}")

  @doc """
  Extracts the error from an error result, raising if it's ok.

  ## Examples

      iex> AuthPlatform.Functional.Result.unwrap_error!({:error, "oops"})
      "oops"

  ## Raises

      AuthPlatform.Functional.Result.unwrap_error!({:ok, 42})
      # ** (RuntimeError) Unwrap error on ok: 42

  """
  @spec unwrap_error!(t(any(), e)) :: e when e: any()
  def unwrap_error!({:error, reason}), do: reason
  def unwrap_error!({:ok, value}), do: raise("Unwrap error on ok: #{inspect(value)}")

  @doc """
  Extracts the value from an ok result, or returns the default if it's an error.

  ## Examples

      iex> AuthPlatform.Functional.Result.unwrap_or({:ok, 42}, 0)
      42

      iex> AuthPlatform.Functional.Result.unwrap_or({:error, "fail"}, 0)
      0

  """
  @spec unwrap_or(t(a, any()), a) :: a when a: any()
  def unwrap_or({:ok, value}, _default), do: value
  def unwrap_or({:error, _}, default), do: default

  @doc """
  Extracts the value from an ok result, or computes a default using the given function.

  ## Examples

      iex> AuthPlatform.Functional.Result.unwrap_or_else({:ok, 42}, fn _ -> 0 end)
      42

      iex> AuthPlatform.Functional.Result.unwrap_or_else({:error, "fail"}, fn e -> String.length(e) end)
      4

  """
  @spec unwrap_or_else(t(a, e), (e -> a)) :: a when a: any(), e: any()
  def unwrap_or_else({:ok, value}, _fun), do: value
  def unwrap_or_else({:error, reason}, fun) when is_function(fun, 1), do: fun.(reason)

  # ============================================================================
  # Predicates
  # ============================================================================

  @doc """
  Returns true if the result is ok.

  ## Examples

      iex> AuthPlatform.Functional.Result.is_ok?({:ok, 42})
      true

      iex> AuthPlatform.Functional.Result.is_ok?({:error, "fail"})
      false

  """
  @spec is_ok?(t(any(), any())) :: boolean()
  def is_ok?({:ok, _}), do: true
  def is_ok?({:error, _}), do: false

  @doc """
  Returns true if the result is an error.

  ## Examples

      iex> AuthPlatform.Functional.Result.is_error?({:error, "fail"})
      true

      iex> AuthPlatform.Functional.Result.is_error?({:ok, 42})
      false

  """
  @spec is_error?(t(any(), any())) :: boolean()
  def is_error?(result), do: not is_ok?(result)

  # ============================================================================
  # Utilities
  # ============================================================================

  @doc """
  Converts an ok result to an option (some), or returns none for errors.

  ## Examples

      iex> AuthPlatform.Functional.Result.to_option({:ok, 42})
      {:some, 42}

      iex> AuthPlatform.Functional.Result.to_option({:error, "fail"})
      :none

  """
  @spec to_option(t(a, any())) :: AuthPlatform.Functional.Option.t(a) when a: any()
  def to_option({:ok, value}), do: {:some, value}
  def to_option({:error, _}), do: :none

  @doc """
  Wraps a block of code that may raise an exception in a result.

  Returns `{:ok, value}` if the block succeeds, or `{:error, exception}` if it raises.

  ## Examples

      iex> AuthPlatform.Functional.Result.try_result do
      ...>   1 + 1
      ...> end
      {:ok, 2}

      iex> AuthPlatform.Functional.Result.try_result do
      ...>   raise "oops"
      ...> end
      {:error, %RuntimeError{message: "oops"}}

  """
  defmacro try_result(do: block) do
    quote do
      try do
        {:ok, unquote(block)}
      rescue
        e -> {:error, e}
      end
    end
  end

  @doc """
  Collects a list of results into a result of a list.

  If all results are ok, returns `{:ok, [values]}`.
  If any result is an error, returns the first error.

  ## Examples

      iex> AuthPlatform.Functional.Result.collect([{:ok, 1}, {:ok, 2}, {:ok, 3}])
      {:ok, [1, 2, 3]}

      iex> AuthPlatform.Functional.Result.collect([{:ok, 1}, {:error, "fail"}, {:ok, 3}])
      {:error, "fail"}

  """
  @spec collect([t(a, e)]) :: t([a], e) when a: any(), e: any()
  def collect(results) when is_list(results) do
    Enum.reduce_while(results, {:ok, []}, fn
      {:ok, value}, {:ok, acc} -> {:cont, {:ok, [value | acc]}}
      {:error, _} = err, _ -> {:halt, err}
    end)
    |> case do
      {:ok, values} -> {:ok, Enum.reverse(values)}
      error -> error
    end
  end
end
