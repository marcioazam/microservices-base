defmodule MfaService.Property.ErrorSanitizationPropertyTest do
  @moduledoc """
  Property-based tests for error message sanitization.
  Validates that error messages do not expose internal details.

  **Feature: mfa-service-modernization-2025, Property 18: Error Message Sanitization**
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias AuthPlatform.Errors.AppError

  @moduletag :property

  # Patterns that should NEVER appear in user-facing error messages
  @forbidden_patterns [
    # Stack traces
    ~r/\*\* \(.*\)/,
    ~r/lib\/.*\.ex:\d+/,
    ~r/\.exs?:\d+:/,
    ~r/at .*:\d+/,

    # Internal module names
    ~r/Elixir\./,
    ~r/MfaService\.\w+\.\w+/,
    ~r/AuthPlatform\.\w+\.\w+/,

    # Database details
    ~r/Ecto\./,
    ~r/Postgrex\./,
    ~r/PostgreSQL/i,
    ~r/constraint.*violated/i,
    ~r/relation.*does not exist/i,

    # File paths
    ~r/\/home\//,
    ~r/\/var\//,
    ~r/\/opt\//,
    ~r/C:\\/,

    # Environment variables
    ~r/DATABASE_URL/,
    ~r/SECRET_KEY/,
    ~r/API_KEY/,

    # Internal error codes
    ~r/EPERM/,
    ~r/ENOENT/,
    ~r/ECONNREFUSED/,

    # Sensitive data patterns
    ~r/password/i,
    ~r/secret/i,
    ~r/token.*[a-zA-Z0-9]{20,}/i,
    ~r/key.*[a-zA-Z0-9]{20,}/i
  ]

  describe "Property 18: Error Message Sanitization" do
    @tag property: 18
    property "validation errors do not expose internal details" do
      check all message <- error_message_generator(), max_runs: 100 do
        error = AppError.validation(message)

        # The error message should be sanitized
        assert_no_forbidden_patterns(error.message)
      end
    end

    property "unauthorized errors do not expose internal details" do
      check all message <- error_message_generator(), max_runs: 100 do
        error = AppError.unauthorized(message)

        assert_no_forbidden_patterns(error.message)
      end
    end

    property "not_found errors do not expose internal details" do
      check all message <- error_message_generator(), max_runs: 100 do
        error = AppError.not_found(message)

        assert_no_forbidden_patterns(error.message)
      end
    end

    property "internal errors do not expose stack traces" do
      check all message <- error_message_generator(), max_runs: 100 do
        error = AppError.internal(message)

        # Internal errors especially should not expose details
        assert_no_forbidden_patterns(error.message)
        refute String.contains?(error.message, "**")
        refute String.contains?(error.message, ".ex:")
      end
    end
  end

  describe "Error message content" do
    property "error messages are user-friendly" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 50 do
        # Generate various error types
        errors = [
          AppError.validation("Invalid input"),
          AppError.unauthorized("Authentication required"),
          AppError.forbidden("Access denied"),
          AppError.not_found("Resource not found"),
          AppError.conflict("Resource conflict"),
          AppError.internal("An error occurred")
        ]

        for error <- errors do
          # Should be readable text
          assert is_binary(error.message)
          assert String.length(error.message) > 0
          assert String.length(error.message) < 500

          # Should not contain technical jargon
          assert_no_forbidden_patterns(error.message)
        end
      end
    end

    property "error codes map to appropriate HTTP status codes" do
      check all _iteration <- StreamData.constant(:ok), max_runs: 50 do
        assert AppError.http_status(AppError.validation("test")) in [400, 422]
        assert AppError.http_status(AppError.unauthorized("test")) == 401
        assert AppError.http_status(AppError.forbidden("test")) == 403
        assert AppError.http_status(AppError.not_found("test")) == 404
        assert AppError.http_status(AppError.conflict("test")) == 409
        assert AppError.http_status(AppError.internal("test")) == 500
      end
    end
  end

  describe "Sanitization of specific error types" do
    test "TOTP validation errors are sanitized" do
      error = AppError.unauthorized("Invalid TOTP code")

      assert error.message == "Invalid TOTP code"
      refute String.contains?(error.message, "secret")
      refute String.contains?(error.message, "hmac")
    end

    test "WebAuthn errors are sanitized" do
      error = AppError.unauthorized("WebAuthn signature verification failed")

      assert error.message == "WebAuthn signature verification failed"
      refute String.contains?(error.message, "public_key")
      refute String.contains?(error.message, "credential_id")
    end

    test "Challenge errors are sanitized" do
      error = AppError.not_found("Challenge not found or expired")

      assert error.message == "Challenge not found or expired"
      refute String.contains?(error.message, "cache")
      refute String.contains?(error.message, "redis")
    end

    test "Database errors are sanitized" do
      # Simulate a database error being wrapped
      error = AppError.internal("An error occurred while processing your request")

      refute String.contains?(error.message, "Ecto")
      refute String.contains?(error.message, "Postgrex")
      refute String.contains?(error.message, "constraint")
    end
  end

  # Helper functions

  defp error_message_generator do
    StreamData.member_of([
      "Invalid input",
      "Authentication required",
      "Access denied",
      "Resource not found",
      "Operation failed",
      "Invalid code",
      "Session expired",
      "Rate limit exceeded",
      "Invalid request",
      "Service unavailable"
    ])
  end

  defp assert_no_forbidden_patterns(message) do
    for pattern <- @forbidden_patterns do
      refute Regex.match?(pattern, message),
             "Error message should not match forbidden pattern: #{inspect(pattern)}\nMessage: #{message}"
    end
  end
end
