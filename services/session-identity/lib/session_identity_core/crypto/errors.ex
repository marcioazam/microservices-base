defmodule SessionIdentityCore.Crypto.Errors do
  @moduledoc """
  Structured error handling for crypto operations.
  
  All crypto errors contain:
  - error_code: atom identifying the error type
  - message: human-readable description
  
  Error codes are designed for programmatic handling and monitoring.
  """

  @type crypto_error :: %{
    error_code: atom(),
    message: String.t()
  }

  # Connection Errors

  @doc """
  Returns error when crypto-service is unavailable.
  """
  @spec service_unavailable(String.t() | nil) :: crypto_error()
  def service_unavailable(details \\ nil) do
    %{
      error_code: :crypto_service_unavailable,
      message: build_message("Crypto service is unavailable", details)
    }
  end

  @doc """
  Returns error when crypto operation times out.
  """
  @spec operation_timeout(String.t() | nil) :: crypto_error()
  def operation_timeout(details \\ nil) do
    %{
      error_code: :crypto_operation_timeout,
      message: build_message("Crypto operation timed out", details)
    }
  end

  # Authentication Errors

  @doc """
  Returns error when authentication to crypto-service fails.
  """
  @spec auth_failed(String.t() | nil) :: crypto_error()
  def auth_failed(details \\ nil) do
    %{
      error_code: :crypto_auth_failed,
      message: build_message("Authentication to crypto service failed", details)
    }
  end

  # Key Errors

  @doc """
  Returns error when requested key is not found.
  """
  @spec key_not_found(map() | nil) :: crypto_error()
  def key_not_found(key_id \\ nil) do
    details = if key_id, do: "key_id: #{inspect(key_id)}", else: nil
    %{
      error_code: :key_not_found,
      message: build_message("Encryption key not found", details)
    }
  end

  @doc """
  Returns error when key is in invalid state for operation.
  """
  @spec key_invalid_state(atom()) :: crypto_error()
  def key_invalid_state(state) do
    %{
      error_code: :key_invalid_state,
      message: "Key is in invalid state for this operation: #{state}"
    }
  end

  # Encryption/Decryption Errors

  @doc """
  Returns error when decryption fails.
  """
  @spec decryption_failed(String.t() | nil) :: crypto_error()
  def decryption_failed(details \\ nil) do
    %{
      error_code: :decryption_failed,
      message: build_message("Decryption failed", details)
    }
  end

  @doc """
  Returns error when AAD mismatch is detected.
  """
  @spec aad_mismatch() :: crypto_error()
  def aad_mismatch do
    %{
      error_code: :aad_mismatch,
      message: "Additional authenticated data mismatch - possible tampering detected"
    }
  end

  @doc """
  Returns error when encryption fails.
  """
  @spec encryption_failed(String.t() | nil) :: crypto_error()
  def encryption_failed(details \\ nil) do
    %{
      error_code: :encryption_failed,
      message: build_message("Encryption failed", details)
    }
  end

  # Signature Errors

  @doc """
  Returns error when signature verification fails.
  """
  @spec signature_invalid() :: crypto_error()
  def signature_invalid do
    %{
      error_code: :signature_invalid,
      message: "Signature verification failed"
    }
  end

  @doc """
  Returns error when signing fails.
  """
  @spec signing_failed(String.t() | nil) :: crypto_error()
  def signing_failed(details \\ nil) do
    %{
      error_code: :signing_failed,
      message: build_message("Signing operation failed", details)
    }
  end

  # Input Validation Errors

  @doc """
  Returns error for invalid argument.
  """
  @spec invalid_argument(String.t()) :: crypto_error()
  def invalid_argument(details) do
    %{
      error_code: :invalid_argument,
      message: "Invalid argument: #{details}"
    }
  end

  # Generic Errors

  @doc """
  Returns generic crypto operation error.
  """
  @spec operation_failed(String.t() | nil) :: crypto_error()
  def operation_failed(details \\ nil) do
    %{
      error_code: :crypto_operation_failed,
      message: build_message("Crypto operation failed", details)
    }
  end

  # Error Mapping

  @doc """
  Maps a gRPC error to a structured crypto error.
  """
  @spec from_grpc_error(term()) :: crypto_error()
  def from_grpc_error(%{status: status, message: message}) do
    case status do
      :unavailable -> service_unavailable(message)
      :deadline_exceeded -> operation_timeout(message)
      :unauthenticated -> auth_failed(message)
      :not_found -> key_not_found()
      :invalid_argument -> invalid_argument(message || "Unknown")
      :permission_denied -> auth_failed(message)
      _ -> operation_failed(message)
    end
  end

  def from_grpc_error(error) do
    operation_failed(inspect(error))
  end

  # Validation

  @doc """
  Checks if a value is a valid crypto error.
  """
  @spec valid_error?(term()) :: boolean()
  def valid_error?(%{error_code: code, message: msg}) 
      when is_atom(code) and is_binary(msg), do: true
  def valid_error?(_), do: false

  @doc """
  Extracts error code from a crypto error.
  """
  @spec error_code(crypto_error()) :: atom()
  def error_code(%{error_code: code}), do: code

  @doc """
  Extracts message from a crypto error.
  """
  @spec error_message(crypto_error()) :: String.t()
  def error_message(%{message: msg}), do: msg

  # Private

  defp build_message(base, nil), do: base
  defp build_message(base, details), do: "#{base}: #{details}"
end
