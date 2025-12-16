defmodule SessionIdentityCore.OAuth21Test do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.OAuth21
  alias SessionIdentityCore.OAuth.PKCE

  # Generators for property tests
  defp code_verifier_generator do
    StreamData.string(:alphanumeric, min_length: 43, max_length: 128)
    |> StreamData.filter(fn s -> Regex.match?(~r/^[A-Za-z0-9\-._~]+$/, s) end)
  end

  defp valid_code_challenge_generator do
    code_verifier_generator()
    |> StreamData.map(&PKCE.compute_s256_challenge/1)
  end

  # **Feature: auth-platform-2025-enhancements, Property 19: PKCE Enforcement**
  # **Validates: Requirements 11.1**
  property "PKCE is required for all authorization requests" do
    check all client_id <- StreamData.string(:alphanumeric, min_length: 10, max_length: 32),
              redirect_uri <- StreamData.constant("https://example.com/callback"),
              max_runs: 100 do
      # Request without code_challenge should fail
      params_without_pkce = %{
        "response_type" => "code",
        "client_id" => client_id,
        "redirect_uri" => redirect_uri
      }

      result = OAuth21.validate_authorization_request(params_without_pkce)
      assert {:error, %{error: "invalid_request"}} = result
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 20: PKCE Validation**
  # **Validates: Requirements 11.2**
  property "PKCE validation uses SHA-256 correctly" do
    check all code_verifier <- code_verifier_generator(),
              max_runs: 100 do
      # Compute challenge
      code_challenge = PKCE.compute_s256_challenge(code_verifier)

      # Verification should succeed with correct verifier
      assert :ok = PKCE.verify(code_verifier, code_challenge, "S256")

      # Verification should fail with wrong verifier
      wrong_verifier = code_verifier <> "x"
      assert {:error, :invalid_code_verifier} = PKCE.verify(wrong_verifier, code_challenge, "S256")
    end
  end

  # **Feature: auth-platform-2025-enhancements, Property 21: Deprecated Flow Rejection**
  # **Validates: Requirements 11.3, 11.4**
  property "deprecated flows are rejected with appropriate errors" do
    check all client_id <- StreamData.string(:alphanumeric, min_length: 10, max_length: 32),
              max_runs: 100 do
      # Implicit grant should be rejected
      implicit_params = %{
        "response_type" => "token",
        "client_id" => client_id,
        "redirect_uri" => "https://example.com/callback"
      }

      result = OAuth21.validate_authorization_request(implicit_params)
      assert {:error, %{error: "unsupported_response_type"}} = result

      # ROPC grant should be rejected
      ropc_params = %{
        "grant_type" => "password",
        "username" => "user",
        "password" => "pass"
      }

      result = OAuth21.validate_token_request(ropc_params)
      assert {:error, %{error: "unsupported_grant_type"}} = result
    end
  end

  describe "OAuth 2.1 Authorization Request Validation" do
    test "rejects implicit grant flow" do
      params = %{
        "response_type" => "token",
        "client_id" => "test-client",
        "redirect_uri" => "https://example.com/callback"
      }

      assert {:error, %{error: "unsupported_response_type"}} =
               OAuth21.validate_authorization_request(params)
    end

    test "requires code_challenge parameter" do
      params = %{
        "response_type" => "code",
        "client_id" => "test-client",
        "redirect_uri" => "https://example.com/callback"
      }

      assert {:error, %{error: "invalid_request", error_description: desc}} =
               OAuth21.validate_authorization_request(params)

      assert desc =~ "PKCE is required"
    end

    test "rejects plain code_challenge_method" do
      params = %{
        "response_type" => "code",
        "client_id" => "test-client",
        "redirect_uri" => "https://example.com/callback",
        "code_challenge" => "test-challenge-43-chars-long-xxxxxxxxxxxxxxxxx",
        "code_challenge_method" => "plain"
      }

      assert {:error, %{error: "invalid_request", error_description: desc}} =
               OAuth21.validate_authorization_request(params)

      assert desc =~ "plain"
    end
  end

  describe "OAuth 2.1 Token Request Validation" do
    test "rejects resource owner password credentials grant" do
      params = %{
        "grant_type" => "password",
        "username" => "user@example.com",
        "password" => "secret"
      }

      assert {:error, %{error: "unsupported_grant_type"}} =
               OAuth21.validate_token_request(params)
    end

    test "requires code_verifier for authorization_code grant" do
      params = %{
        "grant_type" => "authorization_code",
        "code" => "auth-code-123",
        "redirect_uri" => "https://example.com/callback"
      }

      assert {:error, %{error: "invalid_request", error_description: desc}} =
               OAuth21.validate_token_request(params)

      assert desc =~ "code_verifier"
    end

    test "accepts valid authorization_code grant with PKCE" do
      code_verifier = String.duplicate("a", 43)

      params = %{
        "grant_type" => "authorization_code",
        "code" => "auth-code-123",
        "redirect_uri" => "https://example.com/callback",
        "code_verifier" => code_verifier
      }

      assert {:ok, _} = OAuth21.validate_token_request(params)
    end

    test "accepts refresh_token grant" do
      params = %{
        "grant_type" => "refresh_token",
        "refresh_token" => "refresh-token-123"
      }

      assert {:ok, _} = OAuth21.validate_token_request(params)
    end
  end

  describe "Deprecated flow detection" do
    test "identifies deprecated grant types" do
      assert OAuth21.deprecated_grant_type?("password")
      assert OAuth21.deprecated_grant_type?("implicit")
      refute OAuth21.deprecated_grant_type?("authorization_code")
      refute OAuth21.deprecated_grant_type?("refresh_token")
    end

    test "identifies deprecated response types" do
      assert OAuth21.deprecated_response_type?("token")
      refute OAuth21.deprecated_response_type?("code")
    end
  end
end
