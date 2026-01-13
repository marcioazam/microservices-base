defmodule MfaService.Crypto.Error do
  @moduledoc """
  Error struct for Crypto Service operations.
  Provides user-safe error messages without exposing internal details.
  Supports error categorization, retryability, and telemetry integration.
  """

  alias MfaService.Crypto.Telemetry

  @type error_code :: 
    :encryption_failed | 
    :decryption_failed | 
    :key_not_found | 
    :key_generation_failed |
    :key_rotation_failed |
    :key_metadata_failed |
    :health_check_failed |
    :service_unavailable |
    :circuit_open |
    :timeout |
    :invalid_response |
    :migration_failed |
    :connection_failed |
    :authentication_failed

  @type t :: %__MODULE__{
    code: error_code(),
    message: String.t(),
    correlation_id: String.t() | nil,
    retryable: boolean(),
    category: atom()
  }

  @enforce_keys [:code, :message]
  defstruct [:code, :message, :correlation_id, retryable: false, category: :unknown]

  @error_messages %{
    encryption_failed: "Failed to encrypt data",
    decryption_failed: "Failed to decrypt data",
    key_not_found: "Encryption key not found",
    key_generation_failed: "Failed to generate encryption key",
    key_rotation_failed: "Failed to rotate encryption key",
    key_metadata_failed: "Failed to retrieve key metadata",
    health_check_failed: "Crypto service health check failed",
    service_unavailable: "Crypto service is unavailable",
    circuit_open: "Crypto service circuit breaker is open",
    timeout: "Crypto service request timed out",
    invalid_response: "Invalid response from crypto service",
    migration_failed: "Failed to migrate encrypted data",
    connection_failed: "Failed to connect to crypto service",
    authentication_failed: "Authentication with crypto service failed"
  }

  @retryable_errors [:service_unavailable, :timeout, :connection_failed]
  
  @error_categories %{
    encryption_failed: :crypto,
    decryption_failed: :crypto,
    key_not_found: :key_management,
    key_generation_failed: :key_management,
    key_rotation_failed: :key_management,
    key_metadata_failed: :key_management,
    health_check_failed: :connectivity,
    service_unavailable: :connectivity,
    circuit_open: :resilience,
    timeout: :connectivity,
    invalid_response: :protocol,
    migration_failed: :migration,
    connection_failed: :connectivity,
    authentication_failed: :security
  }

  @doc """
  Creates a new error with the given code and internal reason.
  The internal reason is NOT exposed in the error message.
  Emits telemetry for the error.
  """
  @spec new(error_code(), term(), String.t() | nil) :: t()
  def new(code, _internal_reason, correlation_id \\ nil) do
    error = %__MODULE__{
      code: code,
      message: Map.get(@error_messages, code, "An error occurred"),
      correlation_id: correlation_id,
      retryable: code in @retryable_errors,
      category: Map.get(@error_categories, code, :unknown)
    }
    
    # Emit telemetry for error tracking
    if correlation_id do
      Telemetry.emit_failure(:error, code, correlation_id)
    end
    
    error
  end

  @doc """
  Creates a new error without emitting telemetry.
  Use when telemetry is handled elsewhere.
  """
  @spec new_silent(error_code(), term(), String.t() | nil) :: t()
  def new_silent(code, _internal_reason, correlation_id \\ nil) do
    %__MODULE__{
      code: code,
      message: Map.get(@error_messages, code, "An error occurred"),
      correlation_id: correlation_id,
      retryable: code in @retryable_errors,
      category: Map.get(@error_categories, code, :unknown)
    }
  end

  @doc """
  Creates a circuit open error.
  """
  @spec circuit_open(String.t() | nil) :: t()
  def circuit_open(correlation_id \\ nil) do
    new_silent(:circuit_open, nil, correlation_id)
  end

  @doc """
  Creates a timeout error.
  """
  @spec timeout(String.t() | nil) :: t()
  def timeout(correlation_id \\ nil) do
    new_silent(:timeout, nil, correlation_id)
  end

  @doc """
  Creates a service unavailable error.
  """
  @spec service_unavailable(String.t() | nil) :: t()
  def service_unavailable(correlation_id \\ nil) do
    new_silent(:service_unavailable, nil, correlation_id)
  end

  @doc """
  Creates a connection failed error.
  """
  @spec connection_failed(String.t() | nil) :: t()
  def connection_failed(correlation_id \\ nil) do
    new_silent(:connection_failed, nil, correlation_id)
  end

  @doc """
  Creates an invalid response error.
  """
  @spec invalid_response(String.t() | nil) :: t()
  def invalid_response(correlation_id \\ nil) do
    new_silent(:invalid_response, nil, correlation_id)
  end

  @doc """
  Returns true if the error is retryable.
  """
  @spec retryable?(t()) :: boolean()
  def retryable?(%__MODULE__{retryable: retryable}), do: retryable

  @doc """
  Returns the error category.
  """
  @spec category(t()) :: atom()
  def category(%__MODULE__{category: category}), do: category

  @doc """
  Wraps a function result, converting errors to CryptoError.
  """
  @spec wrap(term(), error_code(), String.t() | nil) :: {:ok, term()} | {:error, t()}
  def wrap({:ok, result}, _code, _correlation_id), do: {:ok, result}
  def wrap({:error, reason}, code, correlation_id) do
    {:error, new_silent(code, reason, correlation_id)}
  end
  def wrap(:error, code, correlation_id) do
    {:error, new_silent(code, nil, correlation_id)}
  end

  @doc """
  Converts the error to a map for logging.
  Does not include internal details.
  """
  @spec to_log_map(t()) :: map()
  def to_log_map(%__MODULE__{} = error) do
    %{
      error_code: error.code,
      error_message: error.message,
      correlation_id: error.correlation_id,
      retryable: error.retryable
    }
  end
end

defimpl String.Chars, for: MfaService.Crypto.Error do
  def to_string(%MfaService.Crypto.Error{code: code, message: message}) do
    "[#{code}] #{message}"
  end
end
