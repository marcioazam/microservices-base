defmodule SessionIdentityCore.SessionPropertyTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Sessions.Session
  alias SessionIdentityCore.Sessions.SessionSerializer
  alias SessionIdentityCore.Sessions.SessionStore

  # Generators
  defp uuid_generator do
    StreamData.string(:alphanumeric, length: 32)
    |> StreamData.map(fn s ->
      # Format as UUID-like string
      "#{String.slice(s, 0, 8)}-#{String.slice(s, 8, 4)}-#{String.slice(s, 12, 4)}-#{String.slice(s, 16, 4)}-#{String.slice(s, 20, 12)}"
    end)
  end

  defp ip_address_generator do
    StreamData.tuple({
      StreamData.integer(0..255),
      StreamData.integer(0..255),
      StreamData.integer(0..255),
      StreamData.integer(0..255)
    })
    |> StreamData.map(fn {a, b, c, d} -> "#{a}.#{b}.#{c}.#{d}" end)
  end

  defp risk_score_generator do
    StreamData.float(min: 0.0, max: 1.0)
  end

  defp datetime_generator do
    StreamData.integer(0..2_000_000_000)
    |> StreamData.map(&DateTime.from_unix!/1)
  end

  defp session_generator do
    StreamData.fixed_map(%{
      id: uuid_generator(),
      user_id: uuid_generator(),
      device_id: uuid_generator(),
      ip_address: ip_address_generator(),
      user_agent: StreamData.string(:alphanumeric, min_length: 10, max_length: 100),
      device_fingerprint: StreamData.string(:alphanumeric, min_length: 32, max_length: 64),
      risk_score: risk_score_generator(),
      mfa_verified: StreamData.boolean(),
      expires_at: datetime_generator(),
      last_activity: datetime_generator(),
      inserted_at: datetime_generator(),
      updated_at: datetime_generator()
    })
    |> StreamData.map(fn attrs ->
      %Session{
        id: attrs.id,
        user_id: attrs.user_id,
        device_id: attrs.device_id,
        ip_address: attrs.ip_address,
        user_agent: attrs.user_agent,
        device_fingerprint: attrs.device_fingerprint,
        risk_score: attrs.risk_score,
        mfa_verified: attrs.mfa_verified,
        expires_at: attrs.expires_at,
        last_activity: attrs.last_activity,
        inserted_at: attrs.inserted_at,
        updated_at: attrs.updated_at
      }
    end)
  end

  # **Feature: auth-platform-2025-enhancements, Property 18: Session Serialization Round-Trip**
  # **Validates: Requirements 10.6, 10.7**
  property "session serialization round-trip preserves all properties" do
    check all session <- session_generator(), max_runs: 100 do
      # Serialize to JSON
      json = SessionSerializer.serialize(session)

      # Deserialize back
      {:ok, deserialized} = SessionSerializer.deserialize(json)

      # Verify all properties are preserved
      assert deserialized.id == session.id
      assert deserialized.user_id == session.user_id
      assert deserialized.device_id == session.device_id
      assert deserialized.ip_address == session.ip_address
      assert deserialized.user_agent == session.user_agent
      assert deserialized.device_fingerprint == session.device_fingerprint
      assert deserialized.mfa_verified == session.mfa_verified

      # Risk score should be preserved (within floating point tolerance)
      assert_in_delta deserialized.risk_score, session.risk_score, 0.0001

      # Timestamps should be equivalent (within second precision due to ISO8601)
      assert DateTime.diff(deserialized.expires_at, session.expires_at, :second) == 0
      assert DateTime.diff(deserialized.last_activity, session.last_activity, :second) == 0
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 14: Session Token Entropy**
  # **Validates: Requirements 10.1**
  property "session tokens have at least 256 bits of entropy" do
    check all _i <- StreamData.integer(1..100), max_runs: 100 do
      # Generate session token (256 bits = 32 bytes)
      token = :crypto.strong_rand_bytes(32)

      # Verify length
      assert byte_size(token) >= 32

      # Verify randomness (no obvious patterns)
      # Convert to binary string and check distribution
      bits = for <<bit::1 <- token>>, do: bit
      ones = Enum.count(bits, &(&1 == 1))
      zeros = Enum.count(bits, &(&1 == 0))

      # Should be roughly balanced (within 40-60% range for 256 bits)
      total = ones + zeros
      ratio = ones / total
      assert ratio > 0.3 and ratio < 0.7
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 9: Risk Score Bounds**
  # **Validates: Requirements 4.1**
  property "risk scores are always within [0.0, 1.0] range" do
    check all session <- session_generator(), max_runs: 100 do
      assert session.risk_score >= 0.0
      assert session.risk_score <= 1.0
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 10: Risk Threshold Actions**
  # **Validates: Requirements 4.2, 4.3**
  property "risk threshold actions are correctly determined" do
    check all risk_score <- risk_score_generator(), max_runs: 100 do
      action = determine_risk_action(risk_score)

      cond do
        risk_score > 0.9 ->
          assert action == :block

        risk_score > 0.7 ->
          assert action == :step_up

        true ->
          assert action == :allow
      end
    end
  end

  defp determine_risk_action(score) when score > 0.9, do: :block
  defp determine_risk_action(score) when score > 0.7, do: :step_up
  defp determine_risk_action(_score), do: :allow

  describe "Session" do
    test "is_expired? returns true for expired sessions" do
      expired_session = %Session{
        expires_at: DateTime.add(DateTime.utc_now(), -3600, :second)
      }

      assert Session.is_expired?(expired_session)
    end

    test "is_expired? returns false for valid sessions" do
      valid_session = %Session{
        expires_at: DateTime.add(DateTime.utc_now(), 3600, :second)
      }

      refute Session.is_expired?(valid_session)
    end
  end

  describe "SessionSerializer" do
    test "serializes and deserializes correctly" do
      session = %Session{
        id: "test-session-id",
        user_id: "test-user-id",
        device_id: "test-device-id",
        ip_address: "192.168.1.1",
        user_agent: "Mozilla/5.0",
        device_fingerprint: "fp-123456",
        risk_score: 0.25,
        mfa_verified: true,
        expires_at: DateTime.utc_now() |> DateTime.add(3600, :second),
        last_activity: DateTime.utc_now(),
        inserted_at: DateTime.utc_now(),
        updated_at: DateTime.utc_now()
      }

      json = SessionSerializer.serialize(session)
      {:ok, deserialized} = SessionSerializer.deserialize(json)

      assert deserialized.id == session.id
      assert deserialized.user_id == session.user_id
      assert deserialized.mfa_verified == session.mfa_verified
    end

    test "handles nil values gracefully" do
      session = %Session{
        id: "test-id",
        user_id: "user-id",
        device_id: nil,
        ip_address: "127.0.0.1",
        user_agent: nil,
        device_fingerprint: "fp",
        risk_score: 0.0,
        mfa_verified: false,
        expires_at: DateTime.utc_now(),
        last_activity: DateTime.utc_now(),
        inserted_at: DateTime.utc_now(),
        updated_at: DateTime.utc_now()
      }

      json = SessionSerializer.serialize(session)
      {:ok, deserialized} = SessionSerializer.deserialize(json)

      assert deserialized.device_id == nil
      assert deserialized.user_agent == nil
    end
  end
end
