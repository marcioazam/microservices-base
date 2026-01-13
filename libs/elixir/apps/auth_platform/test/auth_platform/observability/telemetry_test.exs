defmodule AuthPlatform.Observability.TelemetryTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Observability.Telemetry

  describe "event lists" do
    test "circuit_breaker_events returns expected events" do
      events = Telemetry.circuit_breaker_events()

      assert [:auth_platform, :circuit_breaker, :state_change] in events
      assert [:auth_platform, :circuit_breaker, :request_blocked] in events
    end

    test "retry_events returns expected events" do
      events = Telemetry.retry_events()

      assert [:auth_platform, :retry, :attempt] in events
      assert [:auth_platform, :retry, :exhausted] in events
    end

    test "rate_limiter_events returns expected events" do
      events = Telemetry.rate_limiter_events()

      assert [:auth_platform, :rate_limiter, :allowed] in events
      assert [:auth_platform, :rate_limiter, :rejected] in events
    end

    test "bulkhead_events returns expected events" do
      events = Telemetry.bulkhead_events()

      assert [:auth_platform, :bulkhead, :acquired] in events
      assert [:auth_platform, :bulkhead, :released] in events
      assert [:auth_platform, :bulkhead, :rejected] in events
      assert [:auth_platform, :bulkhead, :queued] in events
    end

    test "all_events returns all events" do
      all = Telemetry.all_events()

      assert length(all) == 10

      # Verify all event types are included
      assert Enum.all?(Telemetry.circuit_breaker_events(), &(&1 in all))
      assert Enum.all?(Telemetry.retry_events(), &(&1 in all))
      assert Enum.all?(Telemetry.rate_limiter_events(), &(&1 in all))
      assert Enum.all?(Telemetry.bulkhead_events(), &(&1 in all))
    end
  end

  describe "handler attachment" do
    setup do
      # Clean up any existing handlers
      :telemetry.detach("auth_platform_circuit_breaker")
      :telemetry.detach("auth_platform_retry")
      :telemetry.detach("auth_platform_rate_limiter")
      :telemetry.detach("auth_platform_bulkhead")
      :telemetry.detach("auth_platform_all")
      :ok
    end

    test "attach_circuit_breaker_handler attaches handler" do
      handler = fn _event, _measurements, _metadata -> :ok end
      assert :ok = Telemetry.attach_circuit_breaker_handler(handler)
    end

    test "attach_retry_handler attaches handler" do
      handler = fn _event, _measurements, _metadata -> :ok end
      assert :ok = Telemetry.attach_retry_handler(handler)
    end

    test "attach_rate_limiter_handler attaches handler" do
      handler = fn _event, _measurements, _metadata -> :ok end
      assert :ok = Telemetry.attach_rate_limiter_handler(handler)
    end

    test "attach_bulkhead_handler attaches handler" do
      handler = fn _event, _measurements, _metadata -> :ok end
      assert :ok = Telemetry.attach_bulkhead_handler(handler)
    end

    test "attach_all_handlers attaches handler for all events" do
      handler = fn _event, _measurements, _metadata -> :ok end
      assert :ok = Telemetry.attach_all_handlers(handler)
    end

    test "handler receives events" do
      test_pid = self()

      handler = fn event, measurements, metadata ->
        send(test_pid, {:telemetry_event, event, measurements, metadata})
      end

      Telemetry.attach_circuit_breaker_handler(handler)

      # Emit a test event
      :telemetry.execute(
        [:auth_platform, :circuit_breaker, :state_change],
        %{system_time: System.system_time()},
        %{name: :test, from: :closed, to: :open}
      )

      assert_receive {:telemetry_event, [:auth_platform, :circuit_breaker, :state_change], _, _}
    end

    test "detach_handler removes handler" do
      handler = fn _event, _measurements, _metadata -> :ok end
      Telemetry.attach_circuit_breaker_handler(handler)

      assert :ok = Telemetry.detach_handler("auth_platform_circuit_breaker")
    end
  end
end
