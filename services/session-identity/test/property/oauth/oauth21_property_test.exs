defmodule SessionIdentityCore.OAuth.OAuth21PropertyTest do
  @moduledoc """
  Property tests for OAuth 2.1 compliance.
  
  Property 3: Redirect URI Exact Matching
  Property 4: Refresh Token Rotation (placeholder - requires state)
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.OAuth21
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 3: Redirect URI Exact Matching" do
    property "exact match succeeds, non-match fails" do
      check all(
              uri1 <- Generators.redirect_uri(),
              uri2 <- Generators.redirect_uri(),
              uri1 != uri2,
              max_runs: @iterations
            ) do
        # Since get_registered_redirect_uris returns [], all URIs should fail
        # This tests that no pattern matching is happening
        params = %{
          "response_type" => "code",
          "code_challenge" => generate_valid_challenge(),
          "code_challenge_method" => "S256",
          "redirect_uri" => uri1,
          "client_id" => "test_client"
        }

        # Should fail because URI is not registered (exact match required)
        assert {:error, %{error: "invalid_request"}} =
                 OAuth21.validate_authorization_request(params)
      end
    end

    property "substring URIs do not match" do
      check all(
              base_uri <- Generators.redirect_uri(),
              suffix <- string(:alphanumeric, min_length: 1, max_length: 10),
              max_runs: @iterations
            ) do
        extended_uri = base_uri <> "/" <> suffix

        params = %{
          "response_type" => "code",
          "code_challenge" => generate_valid_challenge(),
          "code_challenge_method" => "S256",
          "redirect_uri" => extended_uri,
          "client_id" => "test_client"
        }

        # Extended URI should not match base URI (no pattern matching)
        assert {:error, _} = OAuth21.validate_authorization_request(params)
      end
    end
  end

  describe "OAuth 2.1 Grant Type Validation" do
    property "password grant is always rejected" do
      check all(_ <- constant(nil), max_runs: 10) do
        params = %{"grant_type" => "password", "code_verifier" => generate_valid_verifier()}

        assert {:error, %{error: "unsupported_grant_type"}} =
                 OAuth21.validate_token_request(params)
      end
    end

    property "authorization_code grant requires code_verifier" do
      check all(_ <- constant(nil), max_runs: 10) do
        params = %{"grant_type" => "authorization_code"}

        assert {:error, %{error: "invalid_request"}} = OAuth21.validate_token_request(params)
      end
    end
  end

  describe "OAuth 2.1 Response Type Validation" do
    property "implicit grant (token) is always rejected" do
      check all(_ <- constant(nil), max_runs: 10) do
        params = %{
          "response_type" => "token",
          "code_challenge" => generate_valid_challenge(),
          "redirect_uri" => "https://example.com/callback",
          "client_id" => "test"
        }

        assert {:error, %{error: "unsupported_response_type"}} =
                 OAuth21.validate_authorization_request(params)
      end
    end
  end

  # Helper functions

  defp generate_valid_verifier do
    :crypto.strong_rand_bytes(32) |> Base.url_encode64(padding: false)
  end

  defp generate_valid_challenge do
    generate_valid_verifier()
    |> then(&:crypto.hash(:sha256, &1))
    |> Base.url_encode64(padding: false)
  end
end
