defmodule MfaService.Crypto.ErrorTest do
  @moduledoc """
  Unit tests for CryptoError struct.
  Tests error creation, categorization, and retryability.
  """

  use ExUnit.Case, async: true

  alias MfaService.Crypto.Error

  describe "new/3" do
    test "creates error with correct code" do
      error = Error.new_silent(:encryption_failed, "internal reason", "corr-123")
      
      assert error.code == :encryption_failed
      assert error.correlation_id == "corr-123"
    end

    test "does not expose internal reason in message" do
      internal_reason = "database connection failed at lib/crypto.ex:42"
      error = Error.new_silent(:encryption_failed, internal_reason, nil)
      
      refute error.message =~ "database"
      refute error.message =~ "lib/crypto.ex"
      assert error.message == "Failed to encrypt data"
    end

    test "sets retryable flag for retryable errors" do
      assert Error.new_silent(:timeout, nil, nil).retryable == true
      assert Error.new_silent(:service_unavailable, nil, nil).retryable == true
      assert Error.new_silent(:connection_failed, nil, nil).retryable == true
    end

    test "sets retryable flag to false for non-retryable errors" do
      assert Error.new_silent(:encryption_failed, nil, nil).retryable == false
      assert Error.new_silent(:decryption_failed, nil, nil).retryable == false
      assert Error.new_silent(:key_not_found, nil, nil).retryable == false
    end

    test "sets correct category for errors" do
      assert Error.new_silent(:encryption_failed, nil, nil).category == :crypto
      assert Error.new_silent(:key_not_found, nil, nil).category == :key_management
      assert Error.new_silent(:timeout, nil, nil).category == :connectivity
      assert Error.new_silent(:circuit_open, nil, nil).category == :resilience
    end
  end

  describe "convenience constructors" do
    test "circuit_open/1 creates circuit open error" do
      error = Error.circuit_open("corr-456")
      
      assert error.code == :circuit_open
      assert error.correlation_id == "corr-456"
      assert error.retryable == false
      assert error.category == :resilience
    end

    test "timeout/1 creates timeout error" do
      error = Error.timeout("corr-789")
      
      assert error.code == :timeout
      assert error.retryable == true
      assert error.category == :connectivity
    end

    test "service_unavailable/1 creates service unavailable error" do
      error = Error.service_unavailable()
      
      assert error.code == :service_unavailable
      assert error.retryable == true
    end

    test "connection_failed/1 creates connection failed error" do
      error = Error.connection_failed("corr-abc")
      
      assert error.code == :connection_failed
      assert error.retryable == true
      assert error.category == :connectivity
    end

    test "invalid_response/1 creates invalid response error" do
      error = Error.invalid_response()
      
      assert error.code == :invalid_response
      assert error.retryable == false
      assert error.category == :protocol
    end
  end

  describe "retryable?/1" do
    test "returns true for retryable errors" do
      assert Error.retryable?(Error.timeout())
      assert Error.retryable?(Error.service_unavailable())
      assert Error.retryable?(Error.connection_failed())
    end

    test "returns false for non-retryable errors" do
      refute Error.retryable?(Error.circuit_open())
      refute Error.retryable?(Error.invalid_response())
      refute Error.retryable?(Error.new_silent(:encryption_failed, nil, nil))
    end
  end

  describe "category/1" do
    test "returns correct category" do
      assert Error.category(Error.new_silent(:encryption_failed, nil, nil)) == :crypto
      assert Error.category(Error.new_silent(:key_rotation_failed, nil, nil)) == :key_management
      assert Error.category(Error.circuit_open()) == :resilience
    end
  end

  describe "wrap/3" do
    test "passes through ok results" do
      assert {:ok, "data"} = Error.wrap({:ok, "data"}, :encryption_failed, "corr")
    end

    test "wraps error tuples" do
      result = Error.wrap({:error, :some_reason}, :encryption_failed, "corr-123")
      
      assert {:error, %Error{code: :encryption_failed}} = result
    end

    test "wraps bare :error atom" do
      result = Error.wrap(:error, :decryption_failed, "corr-456")
      
      assert {:error, %Error{code: :decryption_failed}} = result
    end
  end

  describe "to_log_map/1" do
    test "returns map suitable for logging" do
      error = Error.new_silent(:encryption_failed, "internal", "corr-123")
      log_map = Error.to_log_map(error)
      
      assert log_map.error_code == :encryption_failed
      assert log_map.correlation_id == "corr-123"
      assert log_map.retryable == false
      assert is_binary(log_map.error_message)
    end

    test "does not include internal details" do
      error = Error.new_silent(:encryption_failed, "secret internal reason", "corr")
      log_map = Error.to_log_map(error)
      
      refute Map.has_key?(log_map, :internal_reason)
      refute log_map.error_message =~ "secret internal"
    end
  end

  describe "String.Chars implementation" do
    test "converts error to string" do
      error = Error.new_silent(:encryption_failed, nil, nil)
      string = to_string(error)
      
      assert string =~ "encryption_failed"
      assert string =~ "Failed to encrypt data"
    end

    test "string does not contain internal details" do
      error = Error.new_silent(:encryption_failed, "internal db error at line 42", nil)
      string = to_string(error)
      
      refute string =~ "internal"
      refute string =~ "line 42"
    end
  end

  describe "error scenarios" do
    test "encryption failure scenario" do
      error = Error.new_silent(:encryption_failed, 
        "GRPC error: connection refused to crypto-service:50051", 
        "req-123")
      
      assert error.code == :encryption_failed
      assert error.message == "Failed to encrypt data"
      refute error.message =~ "GRPC"
      refute error.message =~ "50051"
    end

    test "decryption failure scenario" do
      error = Error.new_silent(:decryption_failed,
        "AES-GCM authentication failed: tag mismatch",
        "req-456")
      
      assert error.code == :decryption_failed
      assert error.message == "Failed to decrypt data"
      refute error.message =~ "AES-GCM"
      refute error.message =~ "tag mismatch"
    end

    test "circuit breaker open scenario" do
      error = Error.circuit_open("req-789")
      
      assert error.code == :circuit_open
      assert error.message == "Crypto service circuit breaker is open"
      assert error.retryable == false
    end
  end
end
