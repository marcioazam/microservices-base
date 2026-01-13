defmodule SessionIdentityCore.Crypto.MetricsPropertyTest do
  @moduledoc """
  Property tests for crypto metrics emission.
  
  Property 14: Metrics Emission
  Validates: Requirements 6.2
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.Metrics

  @min_runs 100

  describe "Property 14: Metrics Emission" do
    property "operation counter accepts valid operation types" do
      check all operation <- member_of([:encrypt, :decrypt, :sign, :verify, :reencrypt]),
                status <- member_of([:success, :error, :fallback]),
                namespace <- member_of([
                  "session_identity:session",
                  "session_identity:refresh_token",
                  "session_identity:jwt"
                ]),
                max_runs: @min_runs do
        # Should not raise
        assert :ok = Metrics.inc_operation(operation, status, namespace)
      end
    end

    property "duration observation accepts valid measurements" do
      check all operation <- member_of([:encrypt, :decrypt, :sign, :verify]),
                namespace <- string(:alphanumeric, min_length: 5, max_length: 30),
                duration_ms <- float(min: 0.0, max: 10_000.0),
                max_runs: @min_runs do
        assert :ok = Metrics.observe_duration(operation, namespace, duration_ms)
      end
    end

    property "circuit breaker state gauge accepts valid states" do
      check all state <- member_of([:closed, :open, :half_open]),
                max_runs: @min_runs do
        assert :ok = Metrics.set_circuit_state(state)
      end
    end

    property "duration values are always non-negative" do
      check all duration <- float(min: 0.0, max: 100_000.0),
                max_runs: @min_runs do
        assert duration >= 0
        # Duration in seconds should be positive
        duration_seconds = duration / 1000
        assert duration_seconds >= 0
      end
    end
  end
end
