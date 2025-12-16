defmodule SessionIdentityCore.Sessions.SessionTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Sessions.Session
  alias SessionIdentityCore.Identity.RiskScorer

  describe "Session Record Completeness" do
    # **Feature: auth-microservices-platform, Property 9: Session Record Completeness**
    # **Validates: Requirements 3.1**
    property "created session contains all required fields" do
      check all user_id <- uuid_generator(),
                ip_address <- ip_address_generator(),
                fingerprint <- StreamData.string(:alphanumeric, min_length: 32, max_length: 64),
                user_agent <- StreamData.string(:printable, min_length: 10, max_length: 200),
                max_runs: 100 do
        
        attrs = %{
          user_id: user_id,
          ip_address: ip_address,
          device_fingerprint: fingerprint,
          user_agent: user_agent
        }

        changeset = Session.changeset(%Session{}, attrs)

        assert changeset.valid?
        
        session = Ecto.Changeset.apply_changes(changeset)

        # Verify all required fields are present
        assert session.user_id == user_id
        assert session.ip_address == ip_address
        assert session.device_fingerprint == fingerprint
        assert session.user_agent == user_agent
        
        # Verify defaults are set
        assert session.risk_score == 0.0
        assert session.mfa_verified == false
        assert session.expires_at != nil
        assert session.last_activity != nil
      end
    end

    property "session without required fields is invalid" do
      check all user_id <- uuid_generator(),
                max_runs: 100 do
        
        # Missing ip_address and device_fingerprint
        attrs = %{user_id: user_id}

        changeset = Session.changeset(%Session{}, attrs)

        refute changeset.valid?
        assert Keyword.has_key?(changeset.errors, :ip_address)
        assert Keyword.has_key?(changeset.errors, :device_fingerprint)
      end
    end
  end

  describe "Risk Scoring" do
    # **Feature: auth-microservices-platform, Property 12: Risk Scoring Triggers Step-Up**
    # **Validates: Requirements 3.5**
    property "high risk score triggers step-up authentication" do
      check all risk_score <- StreamData.float(min: 0.7, max: 1.0),
                max_runs: 100 do
        
        assert RiskScorer.requires_step_up?(risk_score) == true
      end
    end

    property "low risk score does not trigger step-up" do
      check all risk_score <- StreamData.float(min: 0.0, max: 0.69),
                max_runs: 100 do
        
        assert RiskScorer.requires_step_up?(risk_score) == false
      end
    end

    property "required factors increase with risk level" do
      check all low_risk <- StreamData.float(min: 0.0, max: 0.49),
                high_risk <- StreamData.float(min: 0.9, max: 1.0),
                max_runs: 100 do
        
        low_factors = RiskScorer.get_required_factors(low_risk)
        high_factors = RiskScorer.get_required_factors(high_risk)

        assert length(low_factors) <= length(high_factors)
      end
    end
  end

  describe "Session Expiration" do
    property "session is expired after expires_at" do
      check all hours_ago <- StreamData.integer(1..100),
                max_runs: 100 do
        
        expired_at = DateTime.utc_now() |> DateTime.add(-hours_ago * 3600, :second)
        
        session = %Session{
          id: Ecto.UUID.generate(),
          user_id: Ecto.UUID.generate(),
          ip_address: "127.0.0.1",
          device_fingerprint: "test",
          expires_at: expired_at,
          inserted_at: DateTime.utc_now(),
          last_activity: DateTime.utc_now()
        }

        assert Session.is_expired?(session) == true
      end
    end

    property "session is not expired before expires_at" do
      check all hours_future <- StreamData.integer(1..100),
                max_runs: 100 do
        
        expires_at = DateTime.utc_now() |> DateTime.add(hours_future * 3600, :second)
        
        session = %Session{
          id: Ecto.UUID.generate(),
          user_id: Ecto.UUID.generate(),
          ip_address: "127.0.0.1",
          device_fingerprint: "test",
          expires_at: expires_at,
          inserted_at: DateTime.utc_now(),
          last_activity: DateTime.utc_now()
        }

        assert Session.is_expired?(session) == false
      end
    end
  end

  # Generators

  defp uuid_generator do
    StreamData.map(StreamData.constant(nil), fn _ -> Ecto.UUID.generate() end)
  end

  defp ip_address_generator do
    StreamData.map(
      StreamData.tuple({
        StreamData.integer(0..255),
        StreamData.integer(0..255),
        StreamData.integer(0..255),
        StreamData.integer(0..255)
      }),
      fn {a, b, c, d} -> "#{a}.#{b}.#{c}.#{d}" end
    )
  end
end
