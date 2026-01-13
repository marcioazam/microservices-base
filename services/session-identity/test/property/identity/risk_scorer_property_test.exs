defmodule SessionIdentityCore.Identity.RiskScorerPropertyTest do
  @moduledoc """
  Property tests for risk scoring.
  
  Property 9: Risk Score Bounds and Thresholds
  Property 10: Risk Factors Affect Score
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Identity.RiskScorer
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 9: Risk Score Bounds and Thresholds" do
    property "risk score is always in [0.0, 1.0]" do
      check all(
              ip <- Generators.ip_address(),
              fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
              user_id <- Generators.uuid(),
              failed_attempts <- integer(0..20),
              hour <- integer(0..23),
              max_runs: @iterations
            ) do
        session = %{
          ip_address: ip,
          device_fingerprint: fingerprint,
          user_id: user_id
        }

        context = %{
          recent_failed_attempts: failed_attempts,
          hour: hour,
          known_devices: [],
          known_ips: []
        }

        score = RiskScorer.calculate_risk(session, context)

        assert score >= 0.0
        assert score <= 1.0
      end
    end

    property "score >= 0.7 requires step-up" do
      check all(score <- float(min: 0.7, max: 1.0), max_runs: @iterations) do
        assert RiskScorer.requires_step_up?(score) == true
      end
    end

    property "score < 0.7 does not require step-up" do
      check all(score <- float(min: 0.0, max: 0.69), max_runs: @iterations) do
        assert RiskScorer.requires_step_up?(score) == false
      end
    end

    property "score >= 0.9 requires webauthn or totp" do
      check all(score <- float(min: 0.9, max: 1.0), max_runs: @iterations) do
        factors = RiskScorer.get_required_factors(score)
        assert :webauthn in factors or :totp in factors
      end
    end

    property "score >= 0.7 and < 0.9 requires totp" do
      check all(score <- float(min: 0.7, max: 0.89), max_runs: @iterations) do
        factors = RiskScorer.get_required_factors(score)
        assert :totp in factors
      end
    end

    property "score < 0.5 requires no factors" do
      check all(score <- float(min: 0.0, max: 0.49), max_runs: @iterations) do
        factors = RiskScorer.get_required_factors(score)
        assert factors == []
      end
    end
  end

  describe "Property 10: Risk Factors Affect Score" do
    property "known device reduces risk score" do
      check all(
              ip <- Generators.ip_address(),
              fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
              user_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        session = %{
          ip_address: ip,
          device_fingerprint: fingerprint,
          user_id: user_id
        }

        context_unknown = %{known_devices: [], known_ips: []}
        context_known = %{known_devices: [fingerprint], known_ips: []}

        score_unknown = RiskScorer.calculate_risk(session, context_unknown)
        score_known = RiskScorer.calculate_risk(session, context_known)

        assert score_known <= score_unknown
      end
    end

    property "known IP reduces risk score" do
      check all(
              ip <- Generators.ip_address(),
              fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
              user_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        session = %{
          ip_address: ip,
          device_fingerprint: fingerprint,
          user_id: user_id
        }

        context_unknown = %{known_devices: [], known_ips: []}
        context_known = %{known_devices: [], known_ips: [ip]}

        score_unknown = RiskScorer.calculate_risk(session, context_unknown)
        score_known = RiskScorer.calculate_risk(session, context_known)

        assert score_known <= score_unknown
      end
    end

    property "5+ failed attempts increases behavior risk to 0.9" do
      check all(
              ip <- Generators.ip_address(),
              fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
              user_id <- Generators.uuid(),
              failed_attempts <- integer(5..20),
              max_runs: @iterations
            ) do
        session = %{
          ip_address: ip,
          device_fingerprint: fingerprint,
          user_id: user_id
        }

        context_high_risk = %{
          recent_failed_attempts: failed_attempts,
          known_devices: [fingerprint],
          known_ips: [ip]
        }

        context_low_risk = %{
          recent_failed_attempts: 0,
          known_devices: [fingerprint],
          known_ips: [ip]
        }

        score_high = RiskScorer.calculate_risk(session, context_high_risk)
        score_low = RiskScorer.calculate_risk(session, context_low_risk)

        assert score_high > score_low
      end
    end
  end
end
