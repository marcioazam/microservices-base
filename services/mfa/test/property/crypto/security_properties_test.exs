defmodule MfaService.Crypto.SecurityPropertiesTest do
  @moduledoc """
  Property-based tests for security hardening.
  Validates log sanitization, error message sanitization, and response validation.
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Crypto.{Sanitizer, Logger, Error}

  import ExUnit.CaptureLog

  # Generators

  defp secret_generator do
    gen all length <- integer(32..64),
            bytes <- binary(length: length) do
      Base.encode64(bytes)
    end
  end

  defp totp_secret_generator do
    gen all length <- integer(16..32),
            bytes <- binary(length: length) do
      Base.encode32(bytes, padding: false)
    end
  end

  defp hex_key_generator do
    gen all bytes <- binary(length: 32) do
      Base.encode16(bytes, case: :lower)
    end
  end

  defp correlation_id_generator do
    gen all uuid <- binary(length: 16) do
      "corr-" <> Base.encode16(uuid, case: :lower) |> String.slice(0, 20)
    end
  end

  defp error_code_generator do
    member_of([
      :encryption_failed, :decryption_failed, :key_not_found,
      :service_unavailable, :timeout, :invalid_response
    ])
  end

  describe "Property 12: No Sensitive Data in Logs" do
    @tag :property
    property "base64 secrets are redacted from logs" do
      check all secret <- secret_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        log = capture_log(fn ->
          Logger.info("Processing secret: #{secret}", correlation_id, [])
        end)
        
        # Secret should be redacted
        refute log =~ secret
        assert log =~ "[REDACTED]" or not String.contains?(log, secret)
      end
    end

    @tag :property
    property "TOTP secrets are redacted from logs" do
      check all secret <- totp_secret_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        log = capture_log(fn ->
          Logger.info("TOTP secret: #{secret}", correlation_id, [])
        end)
        
        refute log =~ secret
      end
    end

    @tag :property
    property "hex keys are redacted from logs" do
      check all key <- hex_key_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        log = capture_log(fn ->
          Logger.info("Key: #{key}", correlation_id, [])
        end)
        
        refute log =~ key
      end
    end

    @tag :property
    property "sensitive metadata keys are redacted" do
      check all secret <- secret_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 50 do
        
        log = capture_log(fn ->
          Logger.info("Operation complete", correlation_id, [
            secret: secret,
            plaintext: "sensitive data",
            key: "encryption_key_value"
          ])
        end)
        
        refute log =~ secret
        refute log =~ "sensitive data"
        refute log =~ "encryption_key_value"
        assert log =~ "[REDACTED]"
      end
    end
  end

  describe "Property 13: No Internal Details in Errors" do
    @tag :property
    property "error messages do not contain stack traces" do
      check all error_code <- error_code_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        internal_reason = """
        ** (RuntimeError) something went wrong
            lib/mfa_service/crypto/client.ex:42: MfaService.Crypto.Client.encrypt/4
            lib/mfa_service/crypto/totp_encryptor.ex:15: MfaService.Crypto.TOTPEncryptor.encrypt_secret/2
        """
        
        error = Error.new(error_code, internal_reason, correlation_id)
        
        # Error message should not contain internal details
        refute error.message =~ "lib/"
        refute error.message =~ ".ex:"
        refute error.message =~ "RuntimeError"
        refute error.message =~ "stack trace"
      end
    end

    @tag :property
    property "error to_string does not expose internals" do
      check all error_code <- error_code_generator(),
                correlation_id <- correlation_id_generator(),
                max_runs: 100 do
        
        error = Error.new(error_code, "internal: database connection failed at line 42", correlation_id)
        error_string = to_string(error)
        
        refute error_string =~ "database connection"
        refute error_string =~ "line 42"
        refute error_string =~ "internal:"
      end
    end

    @tag :property
    property "sanitized error messages are user-safe" do
      check all message <- string(:printable, min_length: 10, max_length: 200),
                max_runs: 100 do
        
        # Add internal details to message
        internal_message = "#{message} at line 42 in lib/test.ex:123 ** (Error)"
        
        sanitized = Sanitizer.sanitize_error_message(internal_message)
        
        refute sanitized =~ "at line 42"
        refute sanitized =~ "lib/test.ex:123"
        refute sanitized =~ "** (Error)"
      end
    end
  end

  describe "Property 14: Response Validation" do
    @tag :property
    property "responses with missing required fields are rejected" do
      required_fields = [:key_id, :iv, :tag, :ciphertext]
      
      check all present_count <- integer(0..3),
                max_runs: 100 do
        
        # Create response with only some fields
        present_fields = Enum.take(required_fields, present_count)
        response = Map.new(present_fields, fn field -> {field, "value"} end)
        
        result = Sanitizer.validate_response(response, required_fields)
        
        if present_count == 4 do
          assert result == :ok
        else
          assert result == {:error, :invalid_response}
        end
      end
    end

    @tag :property
    property "responses with nil values are rejected" do
      check all field_to_nil <- member_of([:key_id, :iv, :tag, :ciphertext]),
                max_runs: 50 do
        
        response = %{
          key_id: "key-123",
          iv: "iv-value",
          tag: "tag-value",
          ciphertext: "ciphertext-value"
        }
        |> Map.put(field_to_nil, nil)
        
        result = Sanitizer.validate_response(response, [:key_id, :iv, :tag, :ciphertext])
        
        assert result == {:error, :invalid_response}
      end
    end

    @tag :property
    property "responses with wrong types are rejected" do
      type_specs = [key_id: :string, iv: :binary, tag: :binary, ciphertext: :binary]
      
      check all field <- member_of([:key_id, :iv, :tag, :ciphertext]),
                max_runs: 50 do
        
        response = %{
          key_id: "key-123",
          iv: "iv-value",
          tag: "tag-value",
          ciphertext: "ciphertext-value"
        }
        |> Map.put(field, 12345)  # Wrong type (integer instead of string/binary)
        
        result = Sanitizer.validate_response_types(response, type_specs)
        
        assert result == {:error, :invalid_response}
      end
    end

    @tag :property
    property "valid responses pass validation" do
      check all key_id <- string(:alphanumeric, min_length: 5, max_length: 20),
                iv <- binary(length: 12),
                tag <- binary(length: 16),
                ciphertext <- binary(min_length: 1, max_length: 100),
                max_runs: 100 do
        
        response = %{
          key_id: key_id,
          iv: iv,
          tag: tag,
          ciphertext: ciphertext
        }
        
        assert :ok == Sanitizer.validate_response(response, [:key_id, :iv, :tag, :ciphertext])
      end
    end
  end

  describe "Sanitizer edge cases" do
    @tag :property
    property "nested maps are sanitized" do
      check all secret <- secret_generator(),
                max_runs: 50 do
        
        nested = %{
          user: %{
            id: "user-123",
            secret: secret,
            profile: %{
              password: "hunter2"
            }
          }
        }
        
        sanitized = Sanitizer.sanitize_map(nested)
        
        assert sanitized.user.secret == "[REDACTED]"
        assert sanitized.user.profile.password == "[REDACTED]"
        assert sanitized.user.id == "user-123"
      end
    end

    @tag :property
    property "keyword lists are sanitized" do
      check all secret <- secret_generator(),
                max_runs: 50 do
        
        kw = [
          user_id: "user-123",
          secret: secret,
          token: "auth-token"
        ]
        
        sanitized = Sanitizer.sanitize_keyword(kw)
        
        assert Keyword.get(sanitized, :secret) == "[REDACTED]"
        assert Keyword.get(sanitized, :token) == "[REDACTED]"
        assert Keyword.get(sanitized, :user_id) == "user-123"
      end
    end
  end
end
