defmodule SessionIdentityCore.Sessions.SessionSerializerPropertyTest do
  @moduledoc """
  Property tests for session serialization round-trip guarantee.
  
  Property 1: Session Serialization Round-Trip
  For any valid Session struct, serializing to JSON and then deserializing
  back SHALL produce an equivalent Session struct with all fields preserved.
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Sessions.{Session, SessionSerializer}
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 1: Session Serialization Round-Trip" do
    property "serialize then deserialize produces equivalent session" do
      check all(session <- Generators.session(), max_runs: @iterations) do
        json = SessionSerializer.serialize(session)
        {:ok, deserialized} = SessionSerializer.deserialize(json)

        assert deserialized.id == session.id
        assert deserialized.user_id == session.user_id
        assert deserialized.device_id == session.device_id
        assert deserialized.ip_address == session.ip_address
        assert deserialized.user_agent == session.user_agent
        assert deserialized.device_fingerprint == session.device_fingerprint
        assert deserialized.risk_score == session.risk_score
        assert deserialized.mfa_verified == session.mfa_verified

        # DateTime comparison (truncate to seconds for comparison)
        assert DateTime.truncate(deserialized.expires_at, :second) ==
                 DateTime.truncate(session.expires_at, :second)

        assert DateTime.truncate(deserialized.last_activity, :second) ==
                 DateTime.truncate(session.last_activity, :second)
      end
    end

    property "serialized JSON is valid JSON" do
      check all(session <- Generators.session(), max_runs: @iterations) do
        json = SessionSerializer.serialize(session)
        assert {:ok, _} = Jason.decode(json)
      end
    end

    property "to_serializable_map contains all required fields" do
      check all(session <- Generators.session(), max_runs: @iterations) do
        map = SessionSerializer.to_serializable_map(session)

        assert Map.has_key?(map, "id")
        assert Map.has_key?(map, "user_id")
        assert Map.has_key?(map, "ip_address")
        assert Map.has_key?(map, "device_fingerprint")
        assert Map.has_key?(map, "risk_score")
        assert Map.has_key?(map, "mfa_verified")
        assert Map.has_key?(map, "expires_at")
        assert Map.has_key?(map, "last_activity")
      end
    end
  end
end
