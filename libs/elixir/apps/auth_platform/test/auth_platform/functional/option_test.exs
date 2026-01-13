defmodule AuthPlatform.Functional.OptionTest do
  @moduledoc """
  Property and unit tests for Option type.

  **Property 1: Functional Types Round-Trip (Option)**
  **Property 2: Functor Law Compliance (Option)**
  **Property 3: Option Constructor Consistency**
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Functional.Option

  doctest AuthPlatform.Functional.Option

  # Identity function
  defp id(x), do: x

  # Simple pure functions for composition testing
  defp double(x) when is_number(x), do: x * 2
  defp double(x), do: x

  defp increment(x) when is_number(x), do: x + 1
  defp increment(x), do: x

  # ============================================================================
  # Property Tests
  # ============================================================================

  describe "Property 1: Functional Types Round-Trip (Option)" do
    @tag property: true
    @tag validates: "Requirements 1.2"
    property "some(v) |> unwrap!() returns v for any value" do
      check all value <- term() do
        option = Option.some(value)
        assert Option.unwrap!(option) == value
      end
    end
  end

  describe "Property 2: Functor Law Compliance (Option) - Identity" do
    @tag property: true
    @tag validates: "Requirements 1.6"
    property "map(some(v), id) == some(v) for any value" do
      check all value <- term() do
        option = Option.some(value)
        assert Option.map(option, &id/1) == option
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.6"
    property "map(none, id) == none" do
      assert Option.map(:none, &id/1) == :none
    end
  end

  describe "Property 2: Functor Law Compliance (Option) - Composition" do
    @tag property: true
    @tag validates: "Requirements 1.6"
    property "map(map(some(n), f), g) == map(some(n), f >>> g) for numeric values" do
      check all n <- integer() do
        option = Option.some(n)

        # map twice
        mapped_twice = option |> Option.map(&double/1) |> Option.map(&increment/1)

        # map once with composed function
        composed = fn x -> increment(double(x)) end
        mapped_composed = Option.map(option, composed)

        assert mapped_twice == mapped_composed
      end
    end
  end

  describe "Property 3: Option Constructor Consistency" do
    @tag property: true
    @tag validates: "Requirements 1.2"
    property "some(v) |> is_some?() returns true for any value" do
      check all value <- term() do
        option = Option.some(value)
        assert Option.is_some?(option) == true
        assert Option.is_none?(option) == false
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.2"
    property "none() |> is_none?() returns true" do
      assert Option.is_none?(Option.none()) == true
      assert Option.is_some?(Option.none()) == false
    end

    @tag property: true
    @tag validates: "Requirements 1.2"
    property "from_nullable(nil) == none()" do
      assert Option.from_nullable(nil) == Option.none()
    end

    @tag property: true
    @tag validates: "Requirements 1.2"
    property "from_nullable(v) == some(v) for non-nil values" do
      check all value <- term(), value != nil do
        assert Option.from_nullable(value) == Option.some(value)
      end
    end
  end

  describe "Monad Laws (flat_map)" do
    @tag property: true
    @tag validates: "Requirements 1.6"
    property "left identity: flat_map(some(a), f) == f(a)" do
      check all n <- integer() do
        f = fn x -> Option.some(x * 2) end

        left = Option.flat_map(Option.some(n), f)
        right = f.(n)

        assert left == right
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.6"
    property "right identity: flat_map(m, some) == m" do
      check all n <- integer() do
        m = Option.some(n)

        assert Option.flat_map(m, &Option.some/1) == m
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.6"
    property "associativity" do
      check all n <- integer() do
        m = Option.some(n)
        f = fn x -> Option.some(x * 2) end
        g = fn x -> Option.some(x + 1) end

        # (m >>= f) >>= g
        left = m |> Option.flat_map(f) |> Option.flat_map(g)

        # m >>= (\x -> f(x) >>= g)
        right = Option.flat_map(m, fn x -> Option.flat_map(f.(x), g) end)

        assert left == right
      end
    end
  end

  # ============================================================================
  # Unit Tests
  # ============================================================================

  describe "some/1" do
    test "creates a some tuple with the value" do
      assert Option.some(42) == {:some, 42}
      assert Option.some("hello") == {:some, "hello"}
      assert Option.some(nil) == {:some, nil}
    end
  end

  describe "none/0" do
    test "returns the none atom" do
      assert Option.none() == :none
    end
  end

  describe "from_nullable/1" do
    test "returns none for nil" do
      assert Option.from_nullable(nil) == :none
    end

    test "returns some for non-nil values" do
      assert Option.from_nullable(42) == {:some, 42}
      assert Option.from_nullable(false) == {:some, false}
      assert Option.from_nullable(0) == {:some, 0}
    end
  end

  describe "map/2" do
    test "applies function to some value" do
      assert Option.map({:some, 5}, &(&1 * 2)) == {:some, 10}
    end

    test "returns none unchanged" do
      assert Option.map(:none, &(&1 * 2)) == :none
    end
  end

  describe "flat_map/2" do
    test "chains some options" do
      safe_head = fn
        [] -> :none
        [h | _] -> {:some, h}
      end

      assert Option.flat_map({:some, [1, 2, 3]}, safe_head) == {:some, 1}
      assert Option.flat_map({:some, []}, safe_head) == :none
    end

    test "returns none unchanged" do
      assert Option.flat_map(:none, fn _ -> {:some, 1} end) == :none
    end
  end

  describe "match/3" do
    test "calls on_some for some option" do
      result = Option.match({:some, 42}, fn v -> v * 2 end, fn -> 0 end)
      assert result == 84
    end

    test "calls on_none for none option" do
      result = Option.match(:none, fn _ -> 0 end, fn -> "nothing" end)
      assert result == "nothing"
    end
  end

  describe "filter/2" do
    test "keeps some if predicate is true" do
      assert Option.filter({:some, 5}, &(&1 > 3)) == {:some, 5}
    end

    test "returns none if predicate is false" do
      assert Option.filter({:some, 2}, &(&1 > 3)) == :none
    end

    test "returns none for none input" do
      assert Option.filter(:none, &(&1 > 3)) == :none
    end
  end

  describe "unwrap!/1" do
    test "returns value for some" do
      assert Option.unwrap!({:some, 42}) == 42
    end

    test "raises for none" do
      assert_raise RuntimeError, ~r/Unwrap on none/, fn ->
        Option.unwrap!(:none)
      end
    end
  end

  describe "unwrap_or/2" do
    test "returns value for some" do
      assert Option.unwrap_or({:some, 42}, 0) == 42
    end

    test "returns default for none" do
      assert Option.unwrap_or(:none, 0) == 0
    end
  end

  describe "to_result/2" do
    test "converts some to ok" do
      assert Option.to_result({:some, 42}, :not_found) == {:ok, 42}
    end

    test "converts none to error" do
      assert Option.to_result(:none, :not_found) == {:error, :not_found}
    end
  end

  describe "or_else/2" do
    test "returns first some" do
      assert Option.or_else({:some, 1}, {:some, 2}) == {:some, 1}
    end

    test "returns second if first is none" do
      assert Option.or_else(:none, {:some, 2}) == {:some, 2}
    end

    test "returns none if both are none" do
      assert Option.or_else(:none, :none) == :none
    end
  end

  describe "zip/2" do
    test "zips two somes" do
      assert Option.zip({:some, 1}, {:some, 2}) == {:some, {1, 2}}
    end

    test "returns none if first is none" do
      assert Option.zip(:none, {:some, 2}) == :none
    end

    test "returns none if second is none" do
      assert Option.zip({:some, 1}, :none) == :none
    end
  end
end
