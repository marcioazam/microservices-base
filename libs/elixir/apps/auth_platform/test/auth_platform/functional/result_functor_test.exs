defmodule AuthPlatform.Functional.ResultFunctorTest do
  @moduledoc """
  Property tests for Result functor laws.

  **Property 2: Functor Law Compliance**
  - Identity: map(value, id) == value
  - Composition: map(map(value, f), g) == map(value, fn x -> g.(f.(x)) end)
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Functional.Result

  # Identity function
  defp id(x), do: x

  # Simple pure functions for composition testing
  defp double(x) when is_number(x), do: x * 2
  defp double(x), do: x

  defp increment(x) when is_number(x), do: x + 1
  defp increment(x), do: x

  describe "Property 2: Functor Law Compliance - Identity" do
    @tag property: true
    @tag validates: "Requirements 1.5"
    property "map(ok(v), id) == ok(v) for any value" do
      check all value <- term() do
        result = Result.ok(value)
        assert Result.map(result, &id/1) == result
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.5"
    property "map(error(e), id) == error(e) for any error" do
      check all reason <- term() do
        result = Result.error(reason)
        assert Result.map(result, &id/1) == result
      end
    end
  end

  describe "Property 2: Functor Law Compliance - Composition" do
    @tag property: true
    @tag validates: "Requirements 1.5"
    property "map(map(ok(n), f), g) == map(ok(n), f >>> g) for numeric values" do
      check all n <- integer() do
        result = Result.ok(n)

        # map twice
        mapped_twice = result |> Result.map(&double/1) |> Result.map(&increment/1)

        # map once with composed function
        composed = fn x -> increment(double(x)) end
        mapped_composed = Result.map(result, composed)

        assert mapped_twice == mapped_composed
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.5"
    property "composition law holds for error results" do
      check all reason <- term() do
        result = Result.error(reason)

        # Both should return the error unchanged
        mapped_twice = result |> Result.map(&double/1) |> Result.map(&increment/1)
        composed = fn x -> increment(double(x)) end
        mapped_composed = Result.map(result, composed)

        assert mapped_twice == result
        assert mapped_composed == result
        assert mapped_twice == mapped_composed
      end
    end
  end

  describe "Monad Laws (flat_map)" do
    @tag property: true
    @tag validates: "Requirements 1.5"
    property "left identity: flat_map(ok(a), f) == f(a)" do
      check all n <- integer() do
        f = fn x -> Result.ok(x * 2) end

        left = Result.flat_map(Result.ok(n), f)
        right = f.(n)

        assert left == right
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.5"
    property "right identity: flat_map(m, ok) == m" do
      check all n <- integer() do
        m = Result.ok(n)

        assert Result.flat_map(m, &Result.ok/1) == m
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.5"
    property "associativity: flat_map(flat_map(m, f), g) == flat_map(m, fn x -> flat_map(f(x), g) end)" do
      check all n <- integer() do
        m = Result.ok(n)
        f = fn x -> Result.ok(x * 2) end
        g = fn x -> Result.ok(x + 1) end

        # (m >>= f) >>= g
        left = m |> Result.flat_map(f) |> Result.flat_map(g)

        # m >>= (\x -> f(x) >>= g)
        right = Result.flat_map(m, fn x -> Result.flat_map(f.(x), g) end)

        assert left == right
      end
    end
  end
end
