defmodule SessionIdentityCore.Events.EventPropertyTest do
  @moduledoc """
  Property tests for event structure correctness.
  
  Property 12: Event Structure Correctness
  Property 13: Event Replay Consistency (partial - requires store)
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Events.Event
  alias SessionIdentityCore.Shared.DateTime, as: DT
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 12: Event Structure Correctness" do
    property "events have non-null correlation_id" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated", "SessionMfaVerified"]),
              aggregate_id <- Generators.uuid(),
              sequence <- integer(1..1000),
              max_runs: @iterations
            ) do
        event = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: sequence,
          payload: %{}
        })

        assert event.correlation_id != nil
        assert is_binary(event.correlation_id)
        assert String.length(event.correlation_id) > 0
      end
    end

    property "events have non-null event_id" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated"]),
              aggregate_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        event = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: 1,
          payload: %{}
        })

        assert event.event_id != nil
        assert is_binary(event.event_id)
      end
    end

    property "serialized timestamp is ISO 8601 UTC format" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated"]),
              aggregate_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        event = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: 1,
          payload: %{}
        })

        map = Event.to_map(event)
        timestamp_str = map["timestamp"]

        # Verify ISO 8601 format
        assert String.contains?(timestamp_str, "T")
        assert String.ends_with?(timestamp_str, "Z")

        # Verify it can be parsed back
        parsed = DT.from_iso8601(timestamp_str)
        assert parsed != nil
      end
    end

    property "event round-trip preserves all fields" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated", "SessionMfaVerified"]),
              aggregate_id <- Generators.uuid(),
              sequence <- integer(1..1000),
              correlation_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        original = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: sequence,
          correlation_id: correlation_id,
          payload: %{"key" => "value"}
        })

        map = Event.to_map(original)
        {:ok, restored} = Event.from_map(map)

        assert restored.event_id == original.event_id
        assert restored.event_type == original.event_type
        assert restored.aggregate_id == original.aggregate_id
        assert restored.aggregate_type == original.aggregate_type
        assert restored.sequence_number == original.sequence_number
        assert restored.correlation_id == original.correlation_id
        assert restored.payload == original.payload
      end
    end

    property "valid? returns true for complete events" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated"]),
              aggregate_id <- Generators.uuid(),
              sequence <- integer(1..1000),
              max_runs: @iterations
            ) do
        event = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: sequence,
          payload: %{}
        })

        assert Event.valid?(event) == true
      end
    end

    property "schema_version is set to current version" do
      check all(
              event_type <- member_of(["SessionCreated", "SessionInvalidated"]),
              aggregate_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        event = Event.new(%{
          event_type: event_type,
          aggregate_id: aggregate_id,
          aggregate_type: "Session",
          sequence_number: 1,
          payload: %{}
        })

        assert event.schema_version == Event.current_schema_version()
      end
    end
  end
end
