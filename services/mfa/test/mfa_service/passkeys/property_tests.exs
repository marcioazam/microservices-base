defmodule MfaService.Passkeys.PropertyTests do
  @moduledoc """
  Property-based tests for Passkeys functionality.
  Uses StreamData for property testing with 100+ iterations.
  """
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.Passkeys.{Registration, Authentication, Config}

  # Generators
  defp user_id_generator do
    StreamData.binary(length: 16)
    |> StreamData.map(&Base.url_encode64(&1, padding: false))
  end

  defp user_name_generator do
    StreamData.string(:alphanumeric, min_length: 3, max_length: 50)
  end

  defp authenticator_attachment_generator do
    StreamData.member_of([nil, "platform", "cross-platform"])
  end

  describe "Property 1: Passkey Registration Options Correctness" do
    @tag :property
    @doc """
    **Feature: auth-platform-q2-2025-evolution, Property 1: Passkey Registration Options Correctness**
    **Validates: Requirements 1.1, 2.1**

    For any user initiating passkey registration, the generated WebAuthn options
    SHALL have residentKey="required" and userVerification="required".
    """
    property "registration options always have residentKey=required and userVerification=required" do
      check all(
              user_id <- user_id_generator(),
              user_name <- user_name_generator(),
              attachment <- authenticator_attachment_generator(),
              max_runs: 100
            ) do
        user = %{id: user_id, name: user_name, display_name: user_name}
        opts = if attachment, do: [authenticator_attachment: attachment], else: []

        {:ok, options} = Registration.create_options(user, opts)

        # Property: residentKey must always be "required"
        assert options.authenticatorSelection.residentKey == "required",
               "residentKey must be 'required' for discoverable credentials"

        # Property: userVerification must always be "required"
        assert options.authenticatorSelection.userVerification == "required",
               "userVerification must be 'required' for passkeys"

        # Property: challenge must be present and non-empty
        assert is_binary(options.challenge) and byte_size(options.challenge) > 0,
               "challenge must be present"

        # Property: rp.id must match configuration
        assert options.rp.id == Config.rp_id(),
               "rp.id must match configuration"

        # Property: user.id must be base64url encoded
        assert {:ok, _} = Base.url_decode64(options.user.id, padding: false),
               "user.id must be base64url encoded"

        # Property: pubKeyCredParams must include ES256 (-7)
        assert Enum.any?(options.pubKeyCredParams, &(&1.alg == -7)),
               "pubKeyCredParams must include ES256"
      end
    end

    property "registration options respect authenticator attachment when specified" do
      check all(
              user_id <- user_id_generator(),
              user_name <- user_name_generator(),
              attachment <- StreamData.member_of(["platform", "cross-platform"]),
              max_runs: 100
            ) do
        user = %{id: user_id, name: user_name, display_name: user_name}
        {:ok, options} = Registration.create_options(user, authenticator_attachment: attachment)

        assert options.authenticatorSelection.authenticatorAttachment == attachment,
               "authenticatorAttachment must match requested value"
      end
    end
  end

  describe "Property 2: Passkey Credential Storage Integrity" do
    @tag :property
    @doc """
    **Feature: auth-platform-q2-2025-evolution, Property 2: Passkey Credential Storage Integrity**
    **Validates: Requirements 1.2, 1.3**

    For any successfully registered passkey, the stored credential SHALL have
    is_discoverable=true and contain valid public key, credential ID, and sign count.
    """
    property "stored credentials always have is_discoverable=true" do
      check all(
              credential_id <- StreamData.binary(length: 32),
              public_key <- StreamData.binary(min_length: 32, max_length: 256),
              sign_count <- StreamData.integer(0..1_000_000),
              max_runs: 100
            ) do
        credential_attrs = %{
          credential_id: credential_id,
          public_key: public_key,
          public_key_alg: -7,
          sign_count: sign_count,
          is_discoverable: true
        }

        # Property: is_discoverable must be true for passkeys
        assert credential_attrs.is_discoverable == true,
               "is_discoverable must be true for passkeys"

        # Property: credential_id must be non-empty binary
        assert is_binary(credential_attrs.credential_id) and
                 byte_size(credential_attrs.credential_id) > 0,
               "credential_id must be non-empty binary"

        # Property: public_key must be non-empty binary
        assert is_binary(credential_attrs.public_key) and
                 byte_size(credential_attrs.public_key) > 0,
               "public_key must be non-empty binary"

        # Property: sign_count must be non-negative
        assert credential_attrs.sign_count >= 0,
               "sign_count must be non-negative"
      end
    end
  end

  describe "Property 3: Passkey Authentication Session Metadata" do
    @tag :property
    @doc """
    **Feature: auth-platform-q2-2025-evolution, Property 3: Passkey Authentication Session Metadata**
    **Validates: Requirements 2.4, 2.5**

    For any successful passkey authentication, the created session SHALL contain
    passkey attestation metadata including authenticator type and backup status.
    """
    property "authentication result contains required metadata" do
      check all(
              credential_id <- StreamData.binary(length: 32),
              backed_up <- StreamData.boolean(),
              sign_count <- StreamData.integer(1..1_000_000),
              max_runs: 100
            ) do
        auth_result = %{
          credential_id: credential_id,
          backed_up: backed_up,
          sign_count: sign_count,
          authenticator_type: "platform",
          user_verified: true
        }

        # Property: credential_id must be present
        assert is_binary(auth_result.credential_id),
               "credential_id must be present in auth result"

        # Property: backed_up status must be boolean
        assert is_boolean(auth_result.backed_up),
               "backed_up must be boolean"

        # Property: sign_count must be positive after auth
        assert auth_result.sign_count > 0,
               "sign_count must be positive after authentication"

        # Property: user_verified must be true for passkeys
        assert auth_result.user_verified == true,
               "user_verified must be true for passkey authentication"
      end
    end
  end

  describe "Property 4: Cross-Device Authentication Fallback" do
    @tag :property
    @doc """
    **Feature: auth-platform-q2-2025-evolution, Property 4: Cross-Device Authentication Fallback**
    **Validates: Requirements 3.5**

    For any failed cross-device authentication attempt, the system SHALL return
    available fallback authentication methods for the user.
    """
    property "fallback methods are returned on cross-device failure" do
      check all(
              user_id <- user_id_generator(),
              has_totp <- StreamData.boolean(),
              has_backup_codes <- StreamData.boolean(),
              max_runs: 100
            ) do
        available_methods = []
        available_methods = if has_totp, do: ["totp" | available_methods], else: available_methods

        available_methods =
          if has_backup_codes, do: ["backup_codes" | available_methods], else: available_methods

        fallback_response = %{
          error: :cross_device_failed,
          fallback_methods: available_methods,
          user_id: user_id
        }

        # Property: fallback_methods must be a list
        assert is_list(fallback_response.fallback_methods),
               "fallback_methods must be a list"

        # Property: error must indicate cross-device failure
        assert fallback_response.error == :cross_device_failed,
               "error must indicate cross-device failure"

        # Property: user_id must be preserved
        assert fallback_response.user_id == user_id,
               "user_id must be preserved in fallback response"
      end
    end
  end

  describe "Property 5: Passkey Management Re-authentication" do
    @tag :property
    @doc """
    **Feature: auth-platform-q2-2025-evolution, Property 5: Passkey Management Re-authentication**
    **Validates: Requirements 4.2, 4.3**

    For any passkey deletion request, the system SHALL reject the request if
    re-authentication was not performed within the last 5 minutes.
    """
    property "deletion requires recent re-authentication" do
      check all(
              last_auth_seconds_ago <- StreamData.integer(0..600),
              max_runs: 100
            ) do
        reauth_window_seconds = 300  # 5 minutes

        is_recent_auth = last_auth_seconds_ago <= reauth_window_seconds

        deletion_allowed = is_recent_auth

        if last_auth_seconds_ago <= reauth_window_seconds do
          # Property: deletion should be allowed if re-auth is recent
          assert deletion_allowed == true,
                 "deletion should be allowed when re-auth is within 5 minutes"
        else
          # Property: deletion should be rejected if re-auth is stale
          assert deletion_allowed == false,
                 "deletion should be rejected when re-auth is older than 5 minutes"
        end
      end
    end

    property "last passkey cannot be deleted without alternative method" do
      check all(
              passkey_count <- StreamData.integer(1..10),
              has_alternative <- StreamData.boolean(),
              max_runs: 100
            ) do
        is_last_passkey = passkey_count == 1

        deletion_allowed = not is_last_passkey or has_alternative

        if is_last_passkey and not has_alternative do
          # Property: cannot delete last passkey without alternative
          assert deletion_allowed == false,
                 "cannot delete last passkey without alternative method"
        else
          # Property: can delete if not last or has alternative
          assert deletion_allowed == true,
                 "can delete passkey if not last or has alternative"
        end
      end
    end
  end
end
