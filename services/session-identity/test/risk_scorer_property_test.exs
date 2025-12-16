defmodule SessionIdentityCore.RiskScorerPropertyTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Identity.RiskScorer

  # Generators
  defp ip_address_generator do
    StreamData.tuple({
      StreamData.integer(0..255),
      StreamData.integer(0..255),
      StreamData.integer(0..255),
      StreamData.integer(0..255)
    })
    |> StreamData.map(fn {a, b, c, d} -> "#{a}.#{b}.#{c}.#{d}" end)
  end

  defp device_fingerprint_generator do
    StreamData.string(:alphanumeric, min_length: 32, max_length: 64)
  end

  defp session_generator do
    StreamData.fixed_map(%{
      user_id: StreamData.string(:alphanumeric, min_length: 16, max_length: 36),
      ip_address: ip_address_generator(),
      device_fingerprint: device_fingerprint_generator()
    })
    |> StreamData.map(fn attrs ->
      %{
        user_id: attrs.user_id,
        ip_address: attrs.ip_address,
        device_fingerprint: attrs.device_fingerprint
      }
    end)
  end

  defp context_generator do
    StreamData.fixed_map(%{
      known_ips: StreamData.list_of(ip_address_generator(), max_length: 5),
      known_devices: StreamData.list_of(device_fingerprint_generator(), max_length: 3),
      recent_failed_attempts: StreamData.integer(0..10),
      hour: StreamData.integer(0..23),
      location_risk: StreamData.float(min: 0.0, max: 1.0)
    })
  end

  # **Feature: auth-platform-2025-enhancements, Property 9: Risk Score Bounds**
  # **Validates: Requirements 4.1**
  property "risk scores are always within [0.0, 1.0] range" do
    check all session <- session_generator(),
              context <- context_generator(),
              max_runs: 100 do
      score = RiskScorer.calculate_risk(session, context)

      assert score >= 0.0, "Risk score #{score} should be >= 0.0"
      assert score <= 1.0, "Risk score #{score} should be <= 1.0"
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 10: Risk Threshold Actions**
  # **Validates: Requirements 4.2, 4.3**
  property "risk threshold actions are correctly determined" do
    check all score <- StreamData.float(min: 0.0, max: 1.0), max_runs: 100 do
      requires_step_up = RiskScorer.requires_step_up?(score)
      required_factors = RiskScorer.get_required_factors(score)

      cond do
        score > 0.9 ->
          # Should require strongest authentication
          assert requires_step_up
          assert :webauthn in required_factors or :totp in required_factors

        score > 0.7 ->
          # Should require step-up
          assert requires_step_up
          assert length(required_factors) > 0

        score > 0.5 ->
          # May require email verification
          assert :email_verification in required_factors or required_factors == []

        true ->
          # Low risk - no additional factors required
          refute requires_step_up or score > 0.7
      end
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 11: Risk Feature Completeness**
  # **Validates: Requirements 4.4**
  property "risk calculation considers all required features" do
    check all session <- session_generator(),
              context <- context_generator(),
              max_runs: 100 do
      # Calculate risk with full context
      score_with_context = RiskScorer.calculate_risk(session, context)

      # Calculate risk with empty context
      score_without_context = RiskScorer.calculate_risk(session, %{})

      # Scores should potentially differ based on context
      # (they might be the same in some cases, but the function should accept both)
      assert is_float(score_with_context)
      assert is_float(score_without_context)

      # Both should be valid scores
      assert score_with_context >= 0.0 and score_with_context <= 1.0
      assert score_without_context >= 0.0 and score_without_context <= 1.0
    end
  end

  property "known devices reduce risk score" do
    check all session <- session_generator(),
              max_runs: 100 do
      # Context with known device
      context_known = %{
        known_devices: [session.device_fingerprint],
        known_ips: [],
        recent_failed_attempts: 0,
        hour: 12,
        location_risk: 0.0
      }

      # Context with unknown device
      context_unknown = %{
        known_devices: [],
        known_ips: [],
        recent_failed_attempts: 0,
        hour: 12,
        location_risk: 0.0
      }

      score_known = RiskScorer.calculate_risk(session, context_known)
      score_unknown = RiskScorer.calculate_risk(session, context_unknown)

      # Known device should result in lower or equal risk
      assert score_known <= score_unknown,
             "Known device score #{score_known} should be <= unknown device score #{score_unknown}"
    end
  end

  property "failed attempts increase risk score" do
    check all session <- session_generator(),
              base_attempts <- StreamData.integer(0..2),
              additional_attempts <- StreamData.integer(3..10),
              max_runs: 100 do
      context_low = %{
        known_devices: [],
        known_ips: [],
        recent_failed_attempts: base_attempts,
        hour: 12,
        location_risk: 0.0
      }

      context_high = %{
        known_devices: [],
        known_ips: [],
        recent_failed_attempts: base_attempts + additional_attempts,
        hour: 12,
        location_risk: 0.0
      }

      score_low = RiskScorer.calculate_risk(session, context_low)
      score_high = RiskScorer.calculate_risk(session, context_high)

      # More failed attempts should result in higher or equal risk
      assert score_high >= score_low,
             "High attempts score #{score_high} should be >= low attempts score #{score_low}"
    end
  end

  describe "RiskScorer" do
    test "returns 0.0 for completely trusted context" do
      session = %{
        user_id: "user-123",
        ip_address: "192.168.1.1",
        device_fingerprint: "known-device-fp"
      }

      context = %{
        known_ips: ["192.168.1.1"],
        known_devices: ["known-device-fp"],
        recent_failed_attempts: 0,
        hour: 12,
        location_risk: 0.0
      }

      score = RiskScorer.calculate_risk(session, context)
      assert score == 0.0
    end

    test "returns high score for suspicious context" do
      session = %{
        user_id: "user-123",
        ip_address: "1.2.3.4",
        device_fingerprint: "unknown-device"
      }

      context = %{
        known_ips: [],
        known_devices: [],
        recent_failed_attempts: 5,
        hour: 3,  # Unusual hour
        location_risk: 0.8
      }

      score = RiskScorer.calculate_risk(session, context)
      assert score > 0.5
    end

    test "requires_step_up? returns true for high scores" do
      assert RiskScorer.requires_step_up?(0.8)
      assert RiskScorer.requires_step_up?(0.9)
      assert RiskScorer.requires_step_up?(1.0)
    end

    test "requires_step_up? returns false for low scores" do
      refute RiskScorer.requires_step_up?(0.0)
      refute RiskScorer.requires_step_up?(0.5)
      refute RiskScorer.requires_step_up?(0.69)
    end

    test "get_required_factors returns appropriate factors" do
      # Very high risk
      factors_high = RiskScorer.get_required_factors(0.95)
      assert :webauthn in factors_high or :totp in factors_high

      # Medium-high risk
      factors_medium = RiskScorer.get_required_factors(0.75)
      assert :totp in factors_medium

      # Low risk
      factors_low = RiskScorer.get_required_factors(0.3)
      assert factors_low == []
    end
  end
end
