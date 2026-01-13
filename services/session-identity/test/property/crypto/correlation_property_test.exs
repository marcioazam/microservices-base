defmodule SessionIdentityCore.Crypto.CorrelationPropertyTest do
  @moduledoc """
  Property tests for correlation ID inclusion in crypto operations.
  
  **Property 3: Correlation ID Inclusion**
  **Validates: Requirements 1.4**
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.Correlation

  # Generators

  defp valid_correlation_id do
    gen all bytes <- binary(length: 16) do
      Base.encode16(bytes, case: :lower)
    end
  end

  defp opts_with_correlation_id do
    gen all id <- valid_correlation_id(),
            other_opts <- list_of(tuple({atom(:alphanumeric), term()}), max_length: 3) do
      [{:correlation_id, id} | other_opts]
    end
  end

  defp opts_without_correlation_id do
    gen all other_opts <- list_of(tuple({atom(:alphanumeric), term()}), max_length: 3) do
      Keyword.delete(other_opts, :correlation_id)
    end
  end

  # Property Tests

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "get_or_generate returns provided correlation_id when present" do
    check all opts <- opts_with_correlation_id(), max_runs: 100 do
      expected_id = Keyword.get(opts, :correlation_id)
      result = Correlation.get_or_generate(opts)
      
      assert result == expected_id
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "get_or_generate generates new id when not provided" do
    check all opts <- opts_without_correlation_id(), max_runs: 100 do
      result = Correlation.get_or_generate(opts)
      
      assert is_binary(result)
      assert byte_size(result) == 32  # 16 bytes hex encoded
      assert Correlation.valid?(result)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "generated correlation_ids are unique" do
    check all _ <- constant(:ok), max_runs: 100 do
      ids = for _ <- 1..10, do: Correlation.generate()
      
      assert length(Enum.uniq(ids)) == 10
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "ensure_correlation_id always returns opts with valid correlation_id" do
    check all opts <- one_of([opts_with_correlation_id(), opts_without_correlation_id()]),
              max_runs: 100 do
      result_opts = Correlation.ensure_correlation_id(opts)
      correlation_id = Keyword.get(result_opts, :correlation_id)
      
      assert Correlation.valid?(correlation_id)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "valid? returns true for non-empty strings" do
    check all id <- string(:alphanumeric, min_length: 1, max_length: 64), max_runs: 100 do
      assert Correlation.valid?(id)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "valid? returns false for empty or non-string values" do
    check all invalid <- one_of([
              constant(""),
              constant(nil),
              integer(),
              list_of(integer())
            ]),
            max_runs: 100 do
      refute Correlation.valid?(invalid)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "extract_for_logging returns correlation_id when present" do
    check all opts <- opts_with_correlation_id(), max_runs: 100 do
      expected_id = Keyword.get(opts, :correlation_id)
      result = Correlation.extract_for_logging(opts)
      
      assert Keyword.get(result, :correlation_id) == expected_id
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.4"
  property "extract_for_logging returns empty list when not present" do
    check all opts <- opts_without_correlation_id(), max_runs: 100 do
      result = Correlation.extract_for_logging(opts)
      
      assert result == []
    end
  end
end
