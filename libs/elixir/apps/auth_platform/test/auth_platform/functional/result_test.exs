defmodule AuthPlatform.Functional.ResultTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Functional.Result

  doctest AuthPlatform.Functional.Result

  # ============================================================================
  # Property Tests
  # ============================================================================

  describe "Property 1: Functional Types Round-Trip" do
    @tag property: true
    @tag validates: "Requirements 1.3, 1.4"
    property "ok(v) |> unwrap!() returns v for any value" do
      check all value <- term() do
        result = Result.ok(value)
        assert Result.unwrap!(result) == value
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.3, 1.4"
    property "error(e) |> unwrap_error!() returns e for any error" do
      check all reason <- term() do
        result = Result.error(reason)
        assert Result.unwrap_error!(result) == reason
      end
    end
  end

  describe "Property 3: Result Constructor Consistency" do
    @tag property: true
    @tag validates: "Requirements 1.1"
    property "ok(v) |> is_ok?() returns true for any value" do
      check all value <- term() do
        result = Result.ok(value)
        assert Result.is_ok?(result) == true
        assert Result.is_error?(result) == false
      end
    end

    @tag property: true
    @tag validates: "Requirements 1.1"
    property "error(e) |> is_error?() returns true for any error" do
      check all reason <- term() do
        result = Result.error(reason)
        assert Result.is_error?(result) == true
        assert Result.is_ok?(result) == false
      end
    end
  end

  # ============================================================================
  # Unit Tests
  # ============================================================================

  describe "ok/1" do
    test "creates an ok tuple with the value" do
      assert Result.ok(42) == {:ok, 42}
      assert Result.ok("hello") == {:ok, "hello"}
      assert Result.ok(nil) == {:ok, nil}
    end
  end

  describe "error/1" do
    test "creates an error tuple with the reason" do
      assert Result.error("not found") == {:error, "not found"}
      assert Result.error(:timeout) == {:error, :timeout}
    end
  end

  describe "map/2" do
    test "applies function to ok value" do
      assert Result.map({:ok, 5}, &(&1 * 2)) == {:ok, 10}
    end

    test "returns error unchanged" do
      assert Result.map({:error, "fail"}, &(&1 * 2)) == {:error, "fail"}
    end
  end

  describe "flat_map/2" do
    test "chains ok results" do
      divide = fn
        _, 0 -> {:error, :division_by_zero}
        a, b -> {:ok, div(a, b)}
      end

      assert Result.flat_map({:ok, 10}, &divide.(&1, 2)) == {:ok, 5}
      assert Result.flat_map({:ok, 10}, &divide.(&1, 0)) == {:error, :division_by_zero}
    end

    test "returns error unchanged" do
      assert Result.flat_map({:error, "fail"}, fn _ -> {:ok, 1} end) == {:error, "fail"}
    end
  end

  describe "match/3" do
    test "calls on_ok for ok result" do
      result = Result.match({:ok, 42}, fn v -> v * 2 end, fn _ -> 0 end)
      assert result == 84
    end

    test "calls on_error for error result" do
      result = Result.match({:error, "oops"}, fn _ -> 0 end, fn e -> String.length(e) end)
      assert result == 4
    end
  end

  describe "unwrap!/1" do
    test "returns value for ok" do
      assert Result.unwrap!({:ok, 42}) == 42
    end

    test "raises for error" do
      assert_raise RuntimeError, ~r/Unwrap on error/, fn ->
        Result.unwrap!({:error, "oops"})
      end
    end
  end

  describe "unwrap_or/2" do
    test "returns value for ok" do
      assert Result.unwrap_or({:ok, 42}, 0) == 42
    end

    test "returns default for error" do
      assert Result.unwrap_or({:error, "fail"}, 0) == 0
    end
  end

  describe "collect/1" do
    test "collects all ok values" do
      assert Result.collect([{:ok, 1}, {:ok, 2}, {:ok, 3}]) == {:ok, [1, 2, 3]}
    end

    test "returns first error" do
      assert Result.collect([{:ok, 1}, {:error, "fail"}, {:ok, 3}]) == {:error, "fail"}
    end

    test "handles empty list" do
      assert Result.collect([]) == {:ok, []}
    end
  end

  describe "try_result/1" do
    test "wraps successful computation" do
      result = Result.try_result do
        1 + 1
      end

      assert result == {:ok, 2}
    end

    test "wraps raised exception" do
      result = Result.try_result do
        raise "oops"
      end

      assert {:error, %RuntimeError{message: "oops"}} = result
    end
  end

  describe "to_option/1" do
    test "converts ok to some" do
      assert Result.to_option({:ok, 42}) == {:some, 42}
    end

    test "converts error to none" do
      assert Result.to_option({:error, "fail"}) == :none
    end
  end
end
