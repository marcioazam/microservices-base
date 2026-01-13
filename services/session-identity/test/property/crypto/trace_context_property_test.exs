defmodule SessionIdentityCore.Crypto.TraceContextPropertyTest do
  @moduledoc """
  Property tests for W3C Trace Context propagation.
  
  **Property 2: Trace Context Propagation**
  **Validates: Requirements 1.3**
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.TraceContext

  # Generators

  defp hex_string(length) do
    gen all bytes <- binary(length: div(length, 2)) do
      Base.encode16(bytes, case: :lower)
    end
  end

  defp valid_traceparent do
    gen all trace_id <- hex_string(32),
            parent_id <- hex_string(16),
            trace_flags <- member_of(["00", "01"]) do
      "00-#{trace_id}-#{parent_id}-#{trace_flags}"
    end
  end

  defp valid_tracestate do
    gen all key <- string(:alphanumeric, min_length: 1, max_length: 10),
            value <- string(:alphanumeric, min_length: 1, max_length: 20) do
      "#{key}=#{value}"
    end
  end

  # Property Tests

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "traceparent round-trip through parse and format" do
    check all traceparent <- valid_traceparent(), max_runs: 100 do
      {:ok, parsed} = TraceContext.parse_traceparent(traceparent)
      
      assert parsed.version == "00"
      assert byte_size(parsed.trace_id) == 32
      assert byte_size(parsed.parent_id) == 16
      assert parsed.trace_flags in ["00", "01"]
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "build_metadata includes provided traceparent" do
    check all traceparent <- valid_traceparent(), max_runs: 100 do
      metadata = TraceContext.build_metadata(traceparent: traceparent)
      
      assert {"traceparent", ^traceparent} = List.keyfind(metadata, "traceparent", 0)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "build_metadata includes provided tracestate" do
    check all tracestate <- valid_tracestate(), max_runs: 100 do
      metadata = TraceContext.build_metadata(tracestate: tracestate)
      
      assert {"tracestate", ^tracestate} = List.keyfind(metadata, "tracestate", 0)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "build_metadata includes both traceparent and tracestate when provided" do
    check all traceparent <- valid_traceparent(),
              tracestate <- valid_tracestate(),
              max_runs: 100 do
      metadata = TraceContext.build_metadata(traceparent: traceparent, tracestate: tracestate)
      
      assert {"traceparent", ^traceparent} = List.keyfind(metadata, "traceparent", 0)
      assert {"tracestate", ^tracestate} = List.keyfind(metadata, "tracestate", 0)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "generated traceparent is valid W3C format" do
    check all _ <- constant(:ok), max_runs: 100 do
      traceparent = TraceContext.generate_traceparent()
      
      {:ok, parsed} = TraceContext.parse_traceparent(traceparent)
      
      assert parsed.version == "00"
      assert byte_size(parsed.trace_id) == 32
      assert byte_size(parsed.parent_id) == 16
      assert parsed.trace_flags == "01"
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.3"
  property "invalid traceparent format returns error" do
    check all invalid <- one_of([
              string(:alphanumeric, min_length: 1, max_length: 10),
              constant("invalid-format"),
              constant("00-short-id-01")
            ]),
            max_runs: 100 do
      assert {:error, :invalid_format} = TraceContext.parse_traceparent(invalid)
    end
  end
end
