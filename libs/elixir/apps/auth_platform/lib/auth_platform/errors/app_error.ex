defmodule AuthPlatform.Errors.AppError do
  @moduledoc """
  Structured application error with HTTP/gRPC mapping.

  AppError provides a consistent way to represent errors across the application
  with automatic mapping to HTTP status codes and gRPC codes.

  ## Usage

      alias AuthPlatform.Errors.AppError

      # Creating errors
      AppError.not_found("User")
      AppError.validation("Email is invalid")
      AppError.unauthorized("Invalid token")
      AppError.rate_limited()

      # Adding context
      error = AppError.not_found("User")
      |> AppError.with_details(%{user_id: "123"})
      |> AppError.with_correlation_id("req-abc-123")

      # Getting HTTP/gRPC codes
      AppError.http_status(error)  # 404
      AppError.grpc_code(error)    # 5 (NOT_FOUND)

      # Checking if retryable
      AppError.is_retryable?(error)  # false

      # Safe API response (PII redacted)
      AppError.to_api_response(error)

  """

  @type error_code ::
          :not_found
          | :validation
          | :unauthorized
          | :forbidden
          | :internal
          | :rate_limited
          | :timeout
          | :unavailable
          | :conflict
          | :bad_request

  @type t :: %__MODULE__{
          code: error_code(),
          message: String.t(),
          details: map(),
          correlation_id: String.t() | nil,
          cause: Exception.t() | nil,
          retryable: boolean()
        }

  @enforce_keys [:code, :message]
  defstruct [:code, :message, :correlation_id, :cause, details: %{}, retryable: false]

  # HTTP status code mapping
  @http_status_map %{
    not_found: 404,
    validation: 400,
    bad_request: 400,
    unauthorized: 401,
    forbidden: 403,
    internal: 500,
    rate_limited: 429,
    timeout: 504,
    unavailable: 503,
    conflict: 409
  }

  # gRPC status code mapping
  # See: https://grpc.github.io/grpc/core/md_doc_statuscodes.html
  @grpc_code_map %{
    not_found: 5,
    validation: 3,
    bad_request: 3,
    unauthorized: 16,
    forbidden: 7,
    internal: 13,
    rate_limited: 8,
    timeout: 4,
    unavailable: 14,
    conflict: 6
  }

  # PII fields to redact in API responses
  @pii_fields ~w(email phone ssn password token secret key credit_card)a

  # ============================================================================
  # Factory Functions
  # ============================================================================

  @doc """
  Creates a not found error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      iex> error.code
      :not_found
      iex> error.message
      "User not found"

  """
  @spec not_found(String.t()) :: t()
  def not_found(resource) when is_binary(resource) do
    %__MODULE__{code: :not_found, message: "#{resource} not found", retryable: false}
  end

  @doc """
  Creates a validation error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.validation("Email is invalid")
      iex> error.code
      :validation
      iex> error.message
      "Email is invalid"

  """
  @spec validation(String.t()) :: t()
  def validation(message) when is_binary(message) do
    %__MODULE__{code: :validation, message: message, retryable: false}
  end

  @doc """
  Creates a validation error with field details.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.validation_with_fields([{"email", "is invalid"}])
      iex> error.details
      %{fields: [{"email", "is invalid"}]}

  """
  @spec validation_with_fields([{String.t(), String.t()}]) :: t()
  def validation_with_fields(field_errors) when is_list(field_errors) do
    %__MODULE__{
      code: :validation,
      message: "Validation failed",
      details: %{fields: field_errors},
      retryable: false
    }
  end

  @doc """
  Creates an unauthorized error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.unauthorized("Invalid token")
      iex> error.code
      :unauthorized

  """
  @spec unauthorized(String.t()) :: t()
  def unauthorized(message) when is_binary(message) do
    %__MODULE__{code: :unauthorized, message: message, retryable: false}
  end

  @doc """
  Creates a forbidden error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.forbidden("Access denied")
      iex> error.code
      :forbidden

  """
  @spec forbidden(String.t()) :: t()
  def forbidden(message) when is_binary(message) do
    %__MODULE__{code: :forbidden, message: message, retryable: false}
  end

  @doc """
  Creates an internal error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.internal("Database connection failed")
      iex> error.code
      :internal

  """
  @spec internal(String.t()) :: t()
  def internal(message) when is_binary(message) do
    %__MODULE__{code: :internal, message: message, retryable: false}
  end

  @doc """
  Creates a rate limited error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.rate_limited()
      iex> error.code
      :rate_limited
      iex> error.retryable
      true

  """
  @spec rate_limited() :: t()
  def rate_limited do
    %__MODULE__{code: :rate_limited, message: "Rate limit exceeded", retryable: true}
  end

  @doc """
  Creates a timeout error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.timeout("Database query")
      iex> error.code
      :timeout
      iex> error.retryable
      true

  """
  @spec timeout(String.t()) :: t()
  def timeout(operation) when is_binary(operation) do
    %__MODULE__{code: :timeout, message: "#{operation} timed out", retryable: true}
  end

  @doc """
  Creates an unavailable error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.unavailable("Payment service")
      iex> error.code
      :unavailable
      iex> error.retryable
      true

  """
  @spec unavailable(String.t()) :: t()
  def unavailable(service) when is_binary(service) do
    %__MODULE__{code: :unavailable, message: "#{service} unavailable", retryable: true}
  end

  @doc """
  Creates a conflict error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.conflict("Resource already exists")
      iex> error.code
      :conflict

  """
  @spec conflict(String.t()) :: t()
  def conflict(message) when is_binary(message) do
    %__MODULE__{code: :conflict, message: message, retryable: false}
  end

  @doc """
  Creates a bad request error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.bad_request("Invalid JSON")
      iex> error.code
      :bad_request

  """
  @spec bad_request(String.t()) :: t()
  def bad_request(message) when is_binary(message) do
    %__MODULE__{code: :bad_request, message: message, retryable: false}
  end

  # ============================================================================
  # Modifiers
  # ============================================================================

  @doc """
  Adds details to an error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      ...> |> AuthPlatform.Errors.AppError.with_details(%{user_id: "123"})
      iex> error.details
      %{user_id: "123"}

  """
  @spec with_details(t(), map()) :: t()
  def with_details(%__MODULE__{} = error, details) when is_map(details) do
    %{error | details: Map.merge(error.details, details)}
  end

  @doc """
  Adds a correlation ID to an error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      ...> |> AuthPlatform.Errors.AppError.with_correlation_id("req-123")
      iex> error.correlation_id
      "req-123"

  """
  @spec with_correlation_id(t(), String.t()) :: t()
  def with_correlation_id(%__MODULE__{} = error, id) when is_binary(id) do
    %{error | correlation_id: id}
  end

  @doc """
  Adds a cause exception to an error.

  ## Examples

      iex> cause = %RuntimeError{message: "oops"}
      iex> error = AuthPlatform.Errors.AppError.internal("Failed")
      ...> |> AuthPlatform.Errors.AppError.with_cause(cause)
      iex> error.cause
      %RuntimeError{message: "oops"}

  """
  @spec with_cause(t(), Exception.t()) :: t()
  def with_cause(%__MODULE__{} = error, cause) do
    %{error | cause: cause}
  end

  # ============================================================================
  # Status Code Mapping
  # ============================================================================

  @doc """
  Returns the HTTP status code for an error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      iex> AuthPlatform.Errors.AppError.http_status(error)
      404

      iex> error = AuthPlatform.Errors.AppError.rate_limited()
      iex> AuthPlatform.Errors.AppError.http_status(error)
      429

  """
  @spec http_status(t()) :: pos_integer()
  def http_status(%__MODULE__{code: code}) do
    Map.get(@http_status_map, code, 500)
  end

  @doc """
  Returns the gRPC status code for an error.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      iex> AuthPlatform.Errors.AppError.grpc_code(error)
      5

      iex> error = AuthPlatform.Errors.AppError.unauthorized("Invalid")
      iex> AuthPlatform.Errors.AppError.grpc_code(error)
      16

  """
  @spec grpc_code(t()) :: non_neg_integer()
  def grpc_code(%__MODULE__{code: code}) do
    Map.get(@grpc_code_map, code, 13)
  end

  # ============================================================================
  # Classification
  # ============================================================================

  @doc """
  Returns true if the error is retryable.

  Retryable errors are transient failures that may succeed on retry:
  - rate_limited
  - timeout
  - unavailable

  ## Examples

      iex> AuthPlatform.Errors.AppError.is_retryable?(AuthPlatform.Errors.AppError.rate_limited())
      true

      iex> AuthPlatform.Errors.AppError.is_retryable?(AuthPlatform.Errors.AppError.not_found("User"))
      false

  """
  @spec is_retryable?(t()) :: boolean()
  def is_retryable?(%__MODULE__{retryable: retryable}), do: retryable

  # ============================================================================
  # API Response
  # ============================================================================

  @doc """
  Converts an error to a safe API response with PII redaction.

  Internal details and sensitive information are not exposed.

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      ...> |> AuthPlatform.Errors.AppError.with_correlation_id("req-123")
      iex> AuthPlatform.Errors.AppError.to_api_response(error)
      %{error: %{code: :not_found, message: "User not found", correlation_id: "req-123"}}

  """
  @spec to_api_response(t()) :: map()
  def to_api_response(%__MODULE__{} = error) do
    %{
      error: %{
        code: error.code,
        message: redact_pii(error.message),
        correlation_id: error.correlation_id
      }
    }
  end

  @doc """
  Converts an error to a detailed response (for internal logging).

  ## Examples

      iex> error = AuthPlatform.Errors.AppError.not_found("User")
      ...> |> AuthPlatform.Errors.AppError.with_details(%{user_id: "123"})
      iex> response = AuthPlatform.Errors.AppError.to_internal_response(error)
      iex> response.details
      %{user_id: "123"}

  """
  @spec to_internal_response(t()) :: map()
  def to_internal_response(%__MODULE__{} = error) do
    %{
      code: error.code,
      message: error.message,
      details: error.details,
      correlation_id: error.correlation_id,
      retryable: error.retryable,
      cause: format_cause(error.cause)
    }
  end

  # ============================================================================
  # Private Functions
  # ============================================================================

  defp redact_pii(message) when is_binary(message) do
    Enum.reduce(@pii_fields, message, fn field, acc ->
      field_str = Atom.to_string(field)
      # Simple pattern: field=value or field: value
      acc
      |> String.replace(~r/#{field_str}\s*[=:]\s*\S+/i, "#{field_str}=[REDACTED]")
    end)
  end

  defp format_cause(nil), do: nil

  defp format_cause(%{__struct__: struct, message: message}) do
    %{type: struct, message: message}
  end

  defp format_cause(cause), do: inspect(cause)
end
