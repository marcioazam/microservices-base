defmodule SessionIdentityCore.CAEP.EmitterPropertyTest do
  @moduledoc """
  Property tests for CAEP event format.
  
  Property 14: CAEP Event Format
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.CAEP.Emitter
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 14: CAEP Event Format" do
    property "session-revoked events have required SSF fields" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :admin_termination, :security_violation]),
              admin_id <- one_of([constant(nil), Generators.uuid()]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, admin_id)

        # Required SSF fields
        assert Map.has_key?(event, "event_type")
        assert Map.has_key?(event, "subject")
        assert Map.has_key?(event, "event_timestamp")
        assert Map.has_key?(event, "reason_admin")

        assert event["event_type"] == "session-revoked"
      end
    end

    property "subject has required format fields" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :admin_termination]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, nil)

        subject = event["subject"]

        assert Map.has_key?(subject, "format")
        assert Map.has_key?(subject, "iss")
        assert Map.has_key?(subject, "sub")

        assert subject["sub"] == user_id
      end
    end

    property "event_timestamp is ISO 8601 format" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :admin_termination]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, nil)

        timestamp = event["event_timestamp"]

        # Verify ISO 8601 format
        assert String.contains?(timestamp, "T")
        assert String.ends_with?(timestamp, "Z")
      end
    end

    property "reason_admin contains reason field" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :admin_termination, :security_violation]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, nil)

        reason_admin = event["reason_admin"]

        assert Map.has_key?(reason_admin, "reason")
        assert reason_admin["reason"] == Atom.to_string(reason)
      end
    end

    property "admin_id is included when provided" do
      check all(
              user_id <- Generators.uuid(),
              admin_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, :admin_termination, admin_id)

        reason_admin = event["reason_admin"]

        assert Map.has_key?(reason_admin, "admin_id")
        assert reason_admin["admin_id"] == admin_id
      end
    end

    property "admin_id is not included when nil" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :security_violation]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, nil)

        reason_admin = event["reason_admin"]

        refute Map.has_key?(reason_admin, "admin_id")
      end
    end

    property "txn (transaction ID) is always present" do
      check all(
              user_id <- Generators.uuid(),
              reason <- member_of([:user_logout, :admin_termination]),
              max_runs: @iterations
            ) do
        session = %{user_id: user_id}
        event = Emitter.build_event("session-revoked", session, reason, nil)

        assert Map.has_key?(event, "txn")
        assert is_binary(event["txn"])
        assert String.length(event["txn"]) > 0
      end
    end
  end
end
