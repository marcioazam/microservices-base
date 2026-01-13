defmodule AuthPlatform.Errors.AppErrorTest do
  @moduledoc """
  Property and unit tests for AppError.

  **Property 4: Error Code Mapping Consistency**
  **Property 5: Retryable Error Classification**
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Errors.AppError

  doctest AuthPlatform.Errors.AppError

  # All error codes
  @all_error_codes [
    :not_found,
    :validation,
    :unauthorized,
    :forbidden,
    :internal,
    :rate_limited,
    :timeout,
    :unavailable,
    :conflict,
    :bad_request
  ]

  # Retryable error codes
  @retryable_codes [:rate_limited, :timeout, :unavailable]

  # Non-retryable error codes
  @non_retryable_codes [:not_found, :validation, :unauthorized, :forbidden, :internal, :conflict, :bad_request]

  # ============================================================================
  # Property Tests
  # ============================================================================

  describe "Property 4: Error Code Mapping Consistency" do
    @tag property: true
    @tag validates: "Requirements 2.4, 2.5"
    property "http_status returns valid HTTP status code (100-599) for all error codes" do
      check all code <- member_of(@all_error_codes) do
        error = create_error_for_code(code)
        status = AppError.http_status(error)

        assert is_integer(status)
        assert status >= 100 and status <= 599
      end
    end

    @tag property: true
    @tag validates: "Requirements 2.4, 2.5"
    property "grpc_code returns valid gRPC code (0-16) for all error codes" do
      check all code <- member_of(@all_error_codes) do
        error = create_error_for_code(code)
        grpc_code = AppError.grpc_code(error)

        assert is_integer(grpc_code)
        assert grpc_code >= 0 and grpc_code <= 16
      end
    end

    @tag property: true
    @tag validates: "Requirements 2.4, 2.5"
    property "same error code always maps to same HTTP status (deterministic)" do
      check all code <- member_of(@all_error_codes) do
        error1 = create_error_for_code(code)
        error2 = create_error_for_code(code)

        assert AppError.http_status(error1) == AppError.http_status(error2)
      end
    end

    @tag property: true
    @tag validates: "Requirements 2.4, 2.5"
    property "same error code always maps to same gRPC code (deterministic)" do
      check all code <- member_of(@all_error_codes) do
        error1 = create_error_for_code(code)
        error2 = create_error_for_code(code)

        assert AppError.grpc_code(error1) == AppError.grpc_code(error2)
      end
    end
  end

  describe "Property 5: Retryable Error Classification" do
    @tag property: true
    @tag validates: "Requirements 2.6"
    property "rate_limited, timeout, unavailable are always retryable" do
      check all code <- member_of(@retryable_codes) do
        error = create_error_for_code(code)
        assert AppError.is_retryable?(error) == true
      end
    end

    @tag property: true
    @tag validates: "Requirements 2.6"
    property "not_found, validation, unauthorized, internal are never retryable" do
      check all code <- member_of(@non_retryable_codes) do
        error = create_error_for_code(code)
        assert AppError.is_retryable?(error) == false
      end
    end
  end

  # ============================================================================
  # Unit Tests
  # ============================================================================

  describe "factory functions" do
    test "not_found/1 creates correct error" do
      error = AppError.not_found("User")
      assert error.code == :not_found
      assert error.message == "User not found"
      assert error.retryable == false
    end

    test "validation/1 creates correct error" do
      error = AppError.validation("Email is invalid")
      assert error.code == :validation
      assert error.message == "Email is invalid"
    end

    test "validation_with_fields/1 creates error with field details" do
      error = AppError.validation_with_fields([{"email", "is invalid"}, {"name", "is required"}])
      assert error.code == :validation
      assert error.details == %{fields: [{"email", "is invalid"}, {"name", "is required"}]}
    end

    test "unauthorized/1 creates correct error" do
      error = AppError.unauthorized("Invalid token")
      assert error.code == :unauthorized
      assert error.message == "Invalid token"
    end

    test "forbidden/1 creates correct error" do
      error = AppError.forbidden("Access denied")
      assert error.code == :forbidden
    end

    test "internal/1 creates correct error" do
      error = AppError.internal("Database error")
      assert error.code == :internal
    end

    test "rate_limited/0 creates retryable error" do
      error = AppError.rate_limited()
      assert error.code == :rate_limited
      assert error.retryable == true
    end

    test "timeout/1 creates retryable error" do
      error = AppError.timeout("Database query")
      assert error.code == :timeout
      assert error.message == "Database query timed out"
      assert error.retryable == true
    end

    test "unavailable/1 creates retryable error" do
      error = AppError.unavailable("Payment service")
      assert error.code == :unavailable
      assert error.message == "Payment service unavailable"
      assert error.retryable == true
    end

    test "conflict/1 creates correct error" do
      error = AppError.conflict("Resource exists")
      assert error.code == :conflict
    end

    test "bad_request/1 creates correct error" do
      error = AppError.bad_request("Invalid JSON")
      assert error.code == :bad_request
    end
  end

  describe "modifiers" do
    test "with_details/2 adds details" do
      error =
        AppError.not_found("User")
        |> AppError.with_details(%{user_id: "123"})

      assert error.details == %{user_id: "123"}
    end

    test "with_details/2 merges with existing details" do
      error =
        AppError.not_found("User")
        |> AppError.with_details(%{user_id: "123"})
        |> AppError.with_details(%{attempt: 1})

      assert error.details == %{user_id: "123", attempt: 1}
    end

    test "with_correlation_id/2 adds correlation ID" do
      error =
        AppError.not_found("User")
        |> AppError.with_correlation_id("req-123")

      assert error.correlation_id == "req-123"
    end

    test "with_cause/2 adds cause exception" do
      cause = %RuntimeError{message: "oops"}

      error =
        AppError.internal("Failed")
        |> AppError.with_cause(cause)

      assert error.cause == cause
    end
  end

  describe "http_status/1" do
    test "returns correct status codes" do
      assert AppError.http_status(AppError.not_found("X")) == 404
      assert AppError.http_status(AppError.validation("X")) == 400
      assert AppError.http_status(AppError.unauthorized("X")) == 401
      assert AppError.http_status(AppError.forbidden("X")) == 403
      assert AppError.http_status(AppError.internal("X")) == 500
      assert AppError.http_status(AppError.rate_limited()) == 429
      assert AppError.http_status(AppError.timeout("X")) == 504
      assert AppError.http_status(AppError.unavailable("X")) == 503
      assert AppError.http_status(AppError.conflict("X")) == 409
    end
  end

  describe "grpc_code/1" do
    test "returns correct gRPC codes" do
      assert AppError.grpc_code(AppError.not_found("X")) == 5
      assert AppError.grpc_code(AppError.validation("X")) == 3
      assert AppError.grpc_code(AppError.unauthorized("X")) == 16
      assert AppError.grpc_code(AppError.forbidden("X")) == 7
      assert AppError.grpc_code(AppError.internal("X")) == 13
      assert AppError.grpc_code(AppError.rate_limited()) == 8
      assert AppError.grpc_code(AppError.timeout("X")) == 4
      assert AppError.grpc_code(AppError.unavailable("X")) == 14
      assert AppError.grpc_code(AppError.conflict("X")) == 6
    end
  end

  describe "to_api_response/1" do
    test "returns safe response structure" do
      error =
        AppError.not_found("User")
        |> AppError.with_correlation_id("req-123")
        |> AppError.with_details(%{internal_id: "secret"})

      response = AppError.to_api_response(error)

      assert response == %{
               error: %{
                 code: :not_found,
                 message: "User not found",
                 correlation_id: "req-123"
               }
             }

      # Details should not be exposed
      refute Map.has_key?(response.error, :details)
    end
  end

  describe "to_internal_response/1" do
    test "returns detailed response" do
      error =
        AppError.not_found("User")
        |> AppError.with_details(%{user_id: "123"})
        |> AppError.with_correlation_id("req-123")

      response = AppError.to_internal_response(error)

      assert response.code == :not_found
      assert response.message == "User not found"
      assert response.details == %{user_id: "123"}
      assert response.correlation_id == "req-123"
      assert response.retryable == false
    end
  end

  # ============================================================================
  # Helpers
  # ============================================================================

  defp create_error_for_code(:not_found), do: AppError.not_found("Resource")
  defp create_error_for_code(:validation), do: AppError.validation("Invalid")
  defp create_error_for_code(:unauthorized), do: AppError.unauthorized("Unauthorized")
  defp create_error_for_code(:forbidden), do: AppError.forbidden("Forbidden")
  defp create_error_for_code(:internal), do: AppError.internal("Internal error")
  defp create_error_for_code(:rate_limited), do: AppError.rate_limited()
  defp create_error_for_code(:timeout), do: AppError.timeout("Operation")
  defp create_error_for_code(:unavailable), do: AppError.unavailable("Service")
  defp create_error_for_code(:conflict), do: AppError.conflict("Conflict")
  defp create_error_for_code(:bad_request), do: AppError.bad_request("Bad request")
end
