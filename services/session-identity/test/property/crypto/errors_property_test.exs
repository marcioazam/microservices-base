defmodule SessionIdentityCore.Crypto.ErrorsPropertyTest do
  @moduledoc """
  Property tests for structured error handling.
  
  **Property 4: Structured Error Responses**
  **Validates: Requirements 1.5**
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.Errors

  # Generators

  defp grpc_status do
    member_of([
      :unavailable,
      :deadline_exceeded,
      :unauthenticated,
      :not_found,
      :invalid_argument,
      :permission_denied,
      :internal,
      :unknown
    ])
  end

  defp grpc_error do
    gen all status <- grpc_status(),
            message <- one_of([string(:alphanumeric, min_length: 1, max_length: 100), constant(nil)]) do
      %{status: status, message: message}
    end
  end

  defp error_details do
    one_of([
      string(:alphanumeric, min_length: 1, max_length: 100),
      constant(nil)
    ])
  end

  defp key_id do
    gen all namespace <- string(:alphanumeric, min_length: 1, max_length: 20),
            id <- string(:alphanumeric, min_length: 1, max_length: 20),
            version <- integer(0..100) do
      %{namespace: namespace, id: id, version: version}
    end
  end

  # Property Tests

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "all error constructors return valid structured errors" do
    check all details <- error_details(), max_runs: 100 do
      errors = [
        Errors.service_unavailable(details),
        Errors.operation_timeout(details),
        Errors.auth_failed(details),
        Errors.decryption_failed(details),
        Errors.encryption_failed(details),
        Errors.signing_failed(details),
        Errors.operation_failed(details)
      ]

      for error <- errors do
        assert Errors.valid_error?(error)
        assert is_atom(Errors.error_code(error))
        assert is_binary(Errors.error_message(error))
      end
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "key_not_found returns valid error with or without key_id" do
    check all key <- one_of([key_id(), constant(nil)]), max_runs: 100 do
      error = Errors.key_not_found(key)
      
      assert Errors.valid_error?(error)
      assert Errors.error_code(error) == :key_not_found
      assert is_binary(Errors.error_message(error))
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "from_grpc_error maps all gRPC statuses to valid errors" do
    check all grpc_err <- grpc_error(), max_runs: 100 do
      error = Errors.from_grpc_error(grpc_err)
      
      assert Errors.valid_error?(error)
      assert is_atom(Errors.error_code(error))
      assert is_binary(Errors.error_message(error))
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "gRPC unavailable status maps to service_unavailable error code" do
    check all message <- error_details(), max_runs: 100 do
      grpc_err = %{status: :unavailable, message: message}
      error = Errors.from_grpc_error(grpc_err)
      
      assert Errors.error_code(error) == :crypto_service_unavailable
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "gRPC deadline_exceeded status maps to timeout error code" do
    check all message <- error_details(), max_runs: 100 do
      grpc_err = %{status: :deadline_exceeded, message: message}
      error = Errors.from_grpc_error(grpc_err)
      
      assert Errors.error_code(error) == :crypto_operation_timeout
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "gRPC not_found status maps to key_not_found error code" do
    check all message <- error_details(), max_runs: 100 do
      grpc_err = %{status: :not_found, message: message}
      error = Errors.from_grpc_error(grpc_err)
      
      assert Errors.error_code(error) == :key_not_found
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "error messages include details when provided" do
    check all details <- string(:alphanumeric, min_length: 1, max_length: 50), max_runs: 100 do
      error = Errors.service_unavailable(details)
      
      assert String.contains?(Errors.error_message(error), details)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "valid_error? returns false for invalid structures" do
    check all invalid <- one_of([
              constant(%{error_code: "string", message: "msg"}),
              constant(%{error_code: :code}),
              constant(%{message: "msg"}),
              constant(nil),
              integer(),
              string(:alphanumeric)
            ]),
            max_runs: 100 do
      refute Errors.valid_error?(invalid)
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "aad_mismatch returns consistent error" do
    check all _ <- constant(:ok), max_runs: 100 do
      error = Errors.aad_mismatch()
      
      assert Errors.valid_error?(error)
      assert Errors.error_code(error) == :aad_mismatch
    end
  end

  @tag property: true
  @tag validates: "Requirements 1.5"
  property "signature_invalid returns consistent error" do
    check all _ <- constant(:ok), max_runs: 100 do
      error = Errors.signature_invalid()
      
      assert Errors.valid_error?(error)
      assert Errors.error_code(error) == :signature_invalid
    end
  end
end
