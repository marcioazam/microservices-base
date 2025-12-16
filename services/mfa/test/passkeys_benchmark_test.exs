defmodule MfaService.Passkeys.BenchmarkTest do
  @moduledoc """
  Benchmark tests for Passkey operations.

  **Feature: auth-platform-q2-2025-evolution, Property 14: Passkey Latency SLO**
  **Validates: Requirements 12.1, 12.2**

  For any passkey registration or authentication operation under normal load,
  the operation SHALL complete within 200ms (registration) or 100ms (authentication) at p99 latency.
  """
  use ExUnit.Case, async: false

  alias MfaService.Passkeys.{Registration, Authentication, Config}

  @registration_p99_limit_ms 200
  @authentication_p99_limit_ms 100
  @sample_size 100

  describe "Property 14: Passkey Latency SLO" do
    @tag :benchmark
    test "registration options generation meets p99 < 200ms SLO" do
      latencies =
        1..@sample_size
        |> Enum.map(fn i ->
          user = %{
            id: "user-#{i}",
            name: "user#{i}@example.com",
            display_name: "User #{i}"
          }

          {time_us, _result} = :timer.tc(fn ->
            Registration.create_options(user, [])
          end)

          time_us / 1000  # Convert to ms
        end)
        |> Enum.sort()

      p99_index = round(@sample_size * 0.99) - 1
      p99_latency = Enum.at(latencies, p99_index)

      avg_latency = Enum.sum(latencies) / @sample_size
      min_latency = Enum.min(latencies)
      max_latency = Enum.max(latencies)

      IO.puts("\nRegistration Options Benchmark:")
      IO.puts("  Samples: #{@sample_size}")
      IO.puts("  Min: #{Float.round(min_latency, 2)}ms")
      IO.puts("  Avg: #{Float.round(avg_latency, 2)}ms")
      IO.puts("  Max: #{Float.round(max_latency, 2)}ms")
      IO.puts("  P99: #{Float.round(p99_latency, 2)}ms")
      IO.puts("  SLO: #{@registration_p99_limit_ms}ms")

      assert p99_latency < @registration_p99_limit_ms,
        "Registration p99 latency (#{Float.round(p99_latency, 2)}ms) exceeds SLO (#{@registration_p99_limit_ms}ms)"
    end

    @tag :benchmark
    test "authentication options generation meets p99 < 100ms SLO" do
      latencies =
        1..@sample_size
        |> Enum.map(fn i ->
          {time_us, _result} = :timer.tc(fn ->
            Authentication.create_options("user-#{i}", [])
          end)

          time_us / 1000  # Convert to ms
        end)
        |> Enum.sort()

      p99_index = round(@sample_size * 0.99) - 1
      p99_latency = Enum.at(latencies, p99_index)

      avg_latency = Enum.sum(latencies) / @sample_size
      min_latency = Enum.min(latencies)
      max_latency = Enum.max(latencies)

      IO.puts("\nAuthentication Options Benchmark:")
      IO.puts("  Samples: #{@sample_size}")
      IO.puts("  Min: #{Float.round(min_latency, 2)}ms")
      IO.puts("  Avg: #{Float.round(avg_latency, 2)}ms")
      IO.puts("  Max: #{Float.round(max_latency, 2)}ms")
      IO.puts("  P99: #{Float.round(p99_latency, 2)}ms")
      IO.puts("  SLO: #{@authentication_p99_limit_ms}ms")

      assert p99_latency < @authentication_p99_limit_ms,
        "Authentication p99 latency (#{Float.round(p99_latency, 2)}ms) exceeds SLO (#{@authentication_p99_limit_ms}ms)"
    end
  end
end
