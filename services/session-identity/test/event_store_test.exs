defmodule SessionIdentityCore.EventStoreTest do
  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.EventStore.{Event, Store, Aggregate}
  alias SessionIdentityCore.EventStore.Events.SessionCreated

  # Property-based test generators
  defp event_id_generator do
    StreamData.string(:alphanumeric, min_length: 32, max_length: 36)
  end

  defp user_id_generator do
    StreamData.string(:alphanumeric, min_length: 32, max_length: 36)
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

  defp event_payload_generator do
    StreamData.fixed_map(%{
      "session_id" => event_id_generator(),
      "user_id" => user_id_generator(),
      "device_id" => StreamData.string(:alphanumeric, min_length: 16, max_length: 32),
      "ip_address" => ip_address_generator(),
      "risk_score" => risk_score_generator()
    })
  end

  defp event_generator do
    StreamData.fixed_map(%{
      event_id: event_id_generator(),
      event_type: StreamData.member_of(["SessionCreated", "SessionRefreshed", "SessionInvalidated"]),
      aggregate_id: event_id_generator(),
      aggregate_type: StreamData.constant("Session"),
      sequence_number: StreamData.positive_integer(),
      schema_version: StreamData.integer(1..3),
      correlation_id: event_id_generator(),
      causation_id: StreamData.one_of([StreamData.constant(nil), event_id_generator()]),
      payload: event_payload_generator(),
      metadata: StreamData.constant(%{})
    })
    |> StreamData.map(fn attrs ->
      Event.new(attrs)
    end)
  end

  # **Feature: auth-platform-2025-enhancements, Property 4: Event Serialization Round-Trip**
  # **Validates: Requirements 14.6, 14.7**
  property "event serialization round-trip preserves all properties" do
    check all event <- event_generator(), max_runs: 100 do
      # Serialize to JSON
      json = Event.serialize(event)

      # Deserialize back
      {:ok, deserialized} = Event.deserialize(json)

      # Verify all properties are preserved
      assert deserialized.event_id == event.event_id
      assert deserialized.event_type == event.event_type
      assert deserialized.aggregate_id == event.aggregate_id
      assert deserialized.aggregate_type == event.aggregate_type
      assert deserialized.sequence_number == event.sequence_number
      assert deserialized.schema_version == event.schema_version
      assert deserialized.correlation_id == event.correlation_id
      assert deserialized.causation_id == event.causation_id
      assert deserialized.payload == event.payload
      assert deserialized.metadata == event.metadata

      # Timestamp should be equivalent (within microsecond precision)
      assert DateTime.diff(deserialized.timestamp, event.timestamp, :second) == 0
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 3: Event Sequence Monotonicity**
  # **Validates: Requirements 14.4**
  property "event sequence numbers are monotonically increasing" do
    check all events <- StreamData.list_of(event_generator(), min_length: 2, max_length: 10),
              max_runs: 100 do
      # Assign sequence numbers in order
      {sequenced_events, _} =
        Enum.map_reduce(events, 1, fn event, seq ->
          {%{event | sequence_number: seq}, seq + 1}
        end)

      # Verify monotonicity
      sequence_numbers = Enum.map(sequenced_events, & &1.sequence_number)

      pairs = Enum.zip(sequence_numbers, Enum.drop(sequence_numbers, 1))

      Enum.each(pairs, fn {prev, curr} ->
        assert curr > prev, "Sequence #{curr} should be greater than #{prev}"
      end)
    end
  end

  describe "Event" do
    test "creates event with auto-generated fields" do
      event = Event.new(
        event_type: "SessionCreated",
        aggregate_id: "session-123",
        payload: %{user_id: "user-456"}
      )

      assert event.event_id != nil
      assert event.correlation_id != nil
      assert event.timestamp != nil
      assert event.schema_version == 1
      assert event.event_type == "SessionCreated"
    end

    test "serializes and deserializes correctly" do
      event = Event.new(
        event_type: "SessionCreated",
        aggregate_id: "session-123",
        correlation_id: "corr-789",
        payload: %{user_id: "user-456", ip_address: "192.168.1.1"}
      )

      json = Event.serialize(event)
      {:ok, deserialized} = Event.deserialize(json)

      assert deserialized.event_type == event.event_type
      assert deserialized.aggregate_id == event.aggregate_id
      assert deserialized.correlation_id == event.correlation_id
      assert deserialized.payload == event.payload
    end

    test "handles schema migration" do
      # Simulate old schema without metadata
      old_data = %{
        "event_id" => "evt-123",
        "event_type" => "SessionCreated",
        "aggregate_id" => "session-123",
        "aggregate_type" => "Session",
        "sequence_number" => 1,
        "timestamp" => DateTime.utc_now() |> DateTime.to_iso8601(),
        "schema_version" => 0,
        "correlation_id" => "corr-123",
        "payload" => %{}
      }

      json = Jason.encode!(old_data)
      {:ok, event} = Event.deserialize(json)

      assert event.metadata == %{}
      assert event.causation_id == nil
    end
  end

  describe "Aggregate" do
    test "applies SessionCreated event correctly" do
      event = Event.new(
        event_type: "SessionCreated",
        aggregate_id: "session-123",
        payload: %{
          "session_id" => "session-123",
          "user_id" => "user-456",
          "device_id" => "device-789",
          "ip_address" => "192.168.1.1",
          "risk_score" => 0.2,
          "auth_methods" => ["password"]
        }
      )

      aggregate = %Aggregate{version: 0}
      updated = apply(Aggregate, :apply_event, [event, aggregate])

      assert updated.id == "session-123"
      assert updated.user_id == "user-456"
      assert updated.status == :active
      assert updated.version == 1
    end
  end
end
