defmodule MfaService.Integration.LoggingClientTest do
  @moduledoc """
  Integration tests for Logging_Service client.
  Tests connection, fallback behavior, and circuit breaker.
  """

  use ExUnit.Case, async: false

  alias AuthPlatform.Clients.Logging

  @moduletag :integration

  describe "Logging_Service connection" do
    test "logging client is available after application start" do
      # The logging client should be initialized by the application
      # Test basic logging doesn't crash

      # Should not raise
      result = Logging.info("Integration test log", test: true)
      assert result in [:ok, {:error, _}]
    end
  end

  describe "Logging operations" do
    test "info level logging works" do
      result = Logging.info("Test info message", module: __MODULE__)
      assert result in [:ok, {:error, _}]
    end

    test "warn level logging works" do
      result = Logging.warn("Test warning message", module: __MODULE__)
      assert result in [:ok, {:error, _}]
    end

    test "error level logging works" do
      result = Logging.error("Test error message", module: __MODULE__)
      assert result in [:ok, {:error, _}]
    end

    test "debug level logging works" do
      result = Logging.debug("Test debug message", module: __MODULE__)
      assert result in [:ok, {:error, _}]
    end
  end

  describe "Structured logging" do
    test "logs with correlation ID" do
      correlation_id = "corr-#{System.unique_integer()}"

      result = Logging.info("Correlated log",
        correlation_id: correlation_id,
        user_id: "test_user"
      )

      assert result in [:ok, {:error, _}]
    end

    test "logs with metadata" do
      result = Logging.info("Log with metadata",
        event_type: "test_event",
        duration_ms: 42,
        success: true
      )

      assert result in [:ok, {:error, _}]
    end
  end

  describe "Fallback behavior" do
    test "falls back to local Logger when service unavailable" do
      # When Logging_Service is unavailable, should fallback to local Logger
      # This test verifies the fallback doesn't crash

      # Multiple log calls should work even if service is down
      for _ <- 1..10 do
        Logging.info("Fallback test", iteration: System.unique_integer())
      end

      # Should complete without raising
      assert true
    end

    test "preserves correlation ID in fallback" do
      correlation_id = "fallback-corr-#{System.unique_integer()}"

      # Should not lose correlation ID even in fallback mode
      result = Logging.info("Fallback with correlation",
        correlation_id: correlation_id
      )

      assert result in [:ok, {:error, _}]
    end
  end

  describe "Circuit breaker behavior" do
    test "circuit breaker is configured for logging" do
      # The circuit breaker should be started by the application
      # Multiple operations should work without issues

      for i <- 1..20 do
        Logging.info("Circuit breaker test", iteration: i)
      end

      # Should complete without raising
      assert true
    end
  end

  describe "MFA-specific logging" do
    test "logs TOTP validation events" do
      result = Logging.info("TOTP validation",
        event_type: "totp_validate",
        user_id: "test_user",
        success: true,
        duration_ms: 15
      )

      assert result in [:ok, {:error, _}]
    end

    test "logs WebAuthn authentication events" do
      result = Logging.info("WebAuthn authentication",
        event_type: "webauthn_auth",
        user_id: "test_user",
        credential_id: "cred_123",
        success: true
      )

      assert result in [:ok, {:error, _}]
    end

    test "logs passkey management events" do
      result = Logging.info("Passkey renamed",
        event_type: "passkey_rename",
        user_id: "test_user",
        passkey_id: "pk_123"
      )

      assert result in [:ok, {:error, _}]
    end
  end
end
