defmodule MfaService.Benchmark.LatencyBenchmarkTest do
  @moduledoc """
  Benchmark tests for MFA Service latency SLOs.
  Validates p99 latency requirements per spec.

  ## SLO Requirements
  - Registration options: < 200ms p99
  - Authentication options: < 100ms p99
  - TOTP validation: < 50ms p99
  - WebAuthn assertion: < 150ms p99
  """

  use ExUnit.Case, async: false

  alias MfaService.TOTP.{Generator, Validator}
  alias MfaService.Passkeys.{Registration, Authentication}
  alias MfaService.WebAuthn.Authentication, as: WebAuthnAuth

  @moduletag :benchmark
  @iterations 1000

  describe "Registration Options Latency (SLO: < 200ms p99)" do
    @tag timeout: 300_000
    test "p99 latency is under 200ms" do
      user = %{
        id: "benchmark-user-#{System.unique_integer()}",
        name: "benchmark@example.com",
        display_name: "Benchmark User"
      }

      latencies = measure_latencies(@iterations, fn ->
        Registration.create_options(user)
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  Registration Options Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")
      IO.puts("    SLO: < 200ms")

      assert p99 < 200, "P99 latency #{p99}ms exceeds SLO of 200ms"
    end
  end

  describe "Authentication Options Latency (SLO: < 100ms p99)" do
    @tag timeout: 300_000
    test "p99 latency is under 100ms" do
      latencies = measure_latencies(@iterations, fn ->
        Authentication.create_options(nil)
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  Authentication Options Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")
      IO.puts("    SLO: < 100ms")

      assert p99 < 100, "P99 latency #{p99}ms exceeds SLO of 100ms"
    end
  end

  describe "TOTP Validation Latency (SLO: < 50ms p99)" do
    @tag timeout: 300_000
    test "p99 latency is under 50ms" do
      secret = Generator.generate_secret()
      code = Validator.generate_code(secret)

      latencies = measure_latencies(@iterations, fn ->
        Validator.validate(code, secret)
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  TOTP Validation Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")
      IO.puts("    SLO: < 50ms")

      assert p99 < 50, "P99 latency #{p99}ms exceeds SLO of 50ms"
    end
  end

  describe "WebAuthn Begin Authentication Latency (SLO: < 150ms p99)" do
    @tag timeout: 300_000
    test "p99 latency is under 150ms" do
      user_id = "benchmark-user-#{System.unique_integer()}"
      credentials = [
        %{credential_id: :crypto.strong_rand_bytes(32), transports: ["internal"]}
      ]

      latencies = measure_latencies(@iterations, fn ->
        WebAuthnAuth.begin_authentication(user_id, credentials)
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  WebAuthn Begin Authentication Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")
      IO.puts("    SLO: < 150ms")

      assert p99 < 150, "P99 latency #{p99}ms exceeds SLO of 150ms"
    end
  end

  describe "Challenge Generation Latency" do
    @tag timeout: 300_000
    test "challenge generation is fast" do
      latencies = measure_latencies(@iterations, fn ->
        MfaService.Challenge.generate()
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  Challenge Generation Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")

      # Challenge generation should be very fast (< 10ms)
      assert p99 < 10, "P99 latency #{p99}ms is too high for challenge generation"
    end
  end

  describe "Device Fingerprint Computation Latency" do
    @tag timeout: 300_000
    test "fingerprint computation is fast" do
      attrs = %{
        user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
        accept_language: "en-US,en;q=0.9",
        timezone: "America/New_York",
        screen_resolution: "1920x1080",
        platform: "Win32"
      }

      latencies = measure_latencies(@iterations, fn ->
        MfaService.Device.Fingerprint.compute(attrs)
      end)

      p99 = percentile(latencies, 99)
      avg = Enum.sum(latencies) / length(latencies)

      IO.puts("\n  Device Fingerprint Computation Latency:")
      IO.puts("    Iterations: #{@iterations}")
      IO.puts("    Average: #{Float.round(avg, 2)}ms")
      IO.puts("    P99: #{Float.round(p99, 2)}ms")

      # Fingerprint computation should be fast (< 20ms)
      assert p99 < 20, "P99 latency #{p99}ms is too high for fingerprint computation"
    end
  end

  # Helper functions

  defp measure_latencies(iterations, fun) do
    # Warm up
    for _ <- 1..100, do: fun.()

    # Measure
    for _ <- 1..iterations do
      start = System.monotonic_time(:microsecond)
      fun.()
      stop = System.monotonic_time(:microsecond)
      (stop - start) / 1000  # Convert to milliseconds
    end
  end

  defp percentile(latencies, p) do
    sorted = Enum.sort(latencies)
    index = ceil(length(sorted) * p / 100) - 1
    Enum.at(sorted, max(0, index))
  end
end
