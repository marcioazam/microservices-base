defmodule MfaService.Crypto.ObservabilityPropertiesTest do
  @moduledoc """
  Property-based tests for observability features.
  Validates telemetry emission and correlation_id presence in logs.
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias MfaService.Crypto.{Telemetry, Logger}

  import ExUnit.CaptureLog

  # Generators

  defp operation_generator do
    member_of([:encrypt, :decrypt, :health_check, :get_key, :rotate_key])
  end

  defp correlation_id_generator do
    gen all uuid <- binary(length: 16) do
      Base.encode16(uuid, case: :lower)
      |> String.slice(0, 8)
      |> then(&"corr-#{&1}")
    end
  end

  defp duration_generator do
    integer(1..5000)
  end

  defp result_generator do
    one_of([
      constant({:ok, %{data: "test"}}),
      constant({:error, :timeout}),
      constant({:error, :service_unavailable})
    ])
  end

  describe "Property 15: Telemetry for RPC Calls" do
    @tag :property
    property "telemetry is emitted for all RPC calls" do
      # Attach a test handler to capture events
      test_pid = self()
      handler_id = "test-handler-#{:rand.uniform(10000)}"
      
      :telemetry.attach(
        handler_id,
        [:mfa_service, :crypto, :rpc, :call],
        fn event, measurements, metadata, _config ->
          send(test_pid, {:telemetry_event, event, measurements, metadata})
        end,
        nil
      )

      check all operation <- operation_generator(),
                result <- result_generator(),
                duration_ms <- duration_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        # Emit telemetry
        Telemetry.emit_rpc_call(operation, result, duration_ms, correlation_id)
        
        # Verify event was received
        assert_receive {:telemetry_event, event, measurements, metadata}, 1000
        
        # Verify event structure
        assert event == [:mfa_service, :crypto, :rpc, :call]
        assert measurements.duration_ms == duration_ms
        assert metadata.operation == operation
        assert metadata.correlation_id == correlation_id
        
        expected_status = case result do
          {:ok, _} -> :success
          {:error, _} -> :failure
        end
        assert metadata.status == expected_status
      end

      :telemetry.detach(handler_id)
    end

    @tag :property
    property "telemetry includes timestamp" do
      test_pid = self()
      handler_id = "test-handler-ts-#{:rand.uniform(10000)}"
      
      :telemetry.attach(
        handler_id,
        [:mfa_service, :crypto, :rpc, :call],
        fn _event, _measurements, metadata, _config ->
          send(test_pid, {:metadata, metadata})
        end,
        nil
      )

      check all operation <- operation_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 50 do
        
        before = DateTime.utc_now()
        Telemetry.emit_rpc_call(operation, {:ok, %{}}, 100, correlation_id)
        after_call = DateTime.utc_now()
        
        assert_receive {:metadata, metadata}, 1000
        
        assert Map.has_key?(metadata, :timestamp)
        assert DateTime.compare(metadata.timestamp, before) in [:gt, :eq]
        assert DateTime.compare(metadata.timestamp, after_call) in [:lt, :eq]
      end

      :telemetry.detach(handler_id)
    end
  end

  describe "Property 6: Correlation ID in Logs" do
    @tag :property
    property "correlation_id is present in all log entries" do
      check all correlation_id <- correlation_id_generator(),
                message <- string(:alphanumeric, min_length: 5, max_length: 50),
                max_runs: 100 do
        
        log = capture_log(fn ->
          Logger.info(message, correlation_id, [])
        end)
        
        assert log =~ correlation_id
      end
    end

    @tag :property
    property "correlation_id is present in error logs" do
      check all correlation_id <- correlation_id_generator(),
                operation <- operation_generator(),
                max_runs: 50 do
        
        log = capture_log(fn ->
          Logger.log_operation_failure(operation, correlation_id, "test error", [])
        end)
        
        assert log =~ correlation_id
        assert log =~ to_string(operation)
      end
    end

    @tag :property
    property "correlation_id is present in operation logs" do
      check all correlation_id <- correlation_id_generator(),
                operation <- operation_generator(),
                duration_ms <- duration_generator(),
                max_runs: 50 do
        
        start_log = capture_log(fn ->
          Logger.log_operation_start(operation, correlation_id, [])
        end)
        
        complete_log = capture_log(fn ->
          Logger.log_operation_complete(operation, correlation_id, duration_ms, [])
        end)
        
        assert start_log =~ correlation_id
        assert complete_log =~ correlation_id
      end
    end
  end

  describe "Circuit breaker telemetry" do
    @tag :property
    property "circuit breaker state changes emit telemetry" do
      test_pid = self()
      handler_id = "test-cb-handler-#{:rand.uniform(10000)}"
      
      :telemetry.attach(
        handler_id,
        [:mfa_service, :crypto, :circuit_breaker, :state_change],
        fn event, measurements, metadata, _config ->
          send(test_pid, {:cb_event, event, measurements, metadata})
        end,
        nil
      )

      check all state <- member_of([:closed, :half_open, :open]),
                max_runs: 30 do
        
        Telemetry.emit_circuit_breaker_state_change(state)
        
        assert_receive {:cb_event, event, _measurements, metadata}, 1000
        
        assert event == [:mfa_service, :crypto, :circuit_breaker, :state_change]
        assert metadata.new_state == state
      end

      :telemetry.detach(handler_id)
    end

    @tag :property
    property "circuit breaker rejections emit telemetry with correlation_id" do
      test_pid = self()
      handler_id = "test-cb-rej-#{:rand.uniform(10000)}"
      
      :telemetry.attach(
        handler_id,
        [:mfa_service, :crypto, :circuit_breaker, :rejection],
        fn event, _measurements, metadata, _config ->
          send(test_pid, {:rejection_event, event, metadata})
        end,
        nil
      )

      check all operation <- operation_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 50 do
        
        Telemetry.emit_circuit_breaker_rejection(operation, correlation_id)
        
        assert_receive {:rejection_event, event, metadata}, 1000
        
        assert event == [:mfa_service, :crypto, :circuit_breaker, :rejection]
        assert metadata.operation == operation
        assert metadata.correlation_id == correlation_id
      end

      :telemetry.detach(handler_id)
    end
  end

  describe "Failure telemetry" do
    @tag :property
    property "failures emit telemetry with error code" do
      test_pid = self()
      handler_id = "test-failure-#{:rand.uniform(10000)}"
      
      :telemetry.attach(
        handler_id,
        [:mfa_service, :crypto, :failure],
        fn event, _measurements, metadata, _config ->
          send(test_pid, {:failure_event, event, metadata})
        end,
        nil
      )

      error_codes = [:encryption_failed, :decryption_failed, :timeout, :service_unavailable]

      check all operation <- operation_generator(),
                error_code <- member_of(error_codes),
                correlation_id <- correlation_id_generator(),
                max_runs: 50 do
        
        Telemetry.emit_failure(operation, error_code, correlation_id)
        
        assert_receive {:failure_event, event, metadata}, 1000
        
        assert event == [:mfa_service, :crypto, :failure]
        assert metadata.operation == operation
        assert metadata.error_code == error_code
        assert metadata.correlation_id == correlation_id
      end

      :telemetry.detach(handler_id)
    end
  end
end
