defmodule SessionIdentityCore.OAuth.AuthorizationTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.{Authorization, IdToken}

  describe "OAuth Request Validation" do
    # **Feature: auth-microservices-platform, Property 13: OAuth Request Validation**
    # **Validates: Requirements 4.1, 4.2**
    property "validates client_id, redirect_uri, and requires PKCE for public clients" do
      check all client_id <- StreamData.string(:alphanumeric, min_length: 10, max_length: 50),
                redirect_uri <- StreamData.constant("https://example.com/callback"),
                code_challenge <- pkce_challenge_generator(),
                max_runs: 100 do
        
        client = %{
          client_id: client_id,
          client_type: "public",
          redirect_uris: [redirect_uri]
        }

        # Valid request with PKCE
        valid_params = %{
          client_id: client_id,
          redirect_uri: redirect_uri,
          code_challenge: code_challenge,
          code_challenge_method: "S256"
        }

        assert {:ok, _} = Authorization.validate_request(valid_params, client)

        # Request without PKCE for public client should fail
        invalid_params = %{
          client_id: client_id,
          redirect_uri: redirect_uri,
          code_challenge: nil
        }

        assert {:error, :pkce_required} = Authorization.validate_request(invalid_params, client)
      end
    end

    property "rejects invalid redirect_uri" do
      check all client_id <- StreamData.string(:alphanumeric, min_length: 10, max_length: 50),
                max_runs: 100 do
        
        client = %{
          client_id: client_id,
          client_type: "confidential",
          redirect_uris: ["https://allowed.com/callback"]
        }

        params = %{
          client_id: client_id,
          redirect_uri: "https://malicious.com/callback"
        }

        assert {:error, :invalid_redirect_uri} = Authorization.validate_request(params, client)
      end
    end

    property "rejects invalid client_id" do
      check all client_id <- StreamData.string(:alphanumeric, min_length: 10, max_length: 50),
                max_runs: 100 do
        
        client = %{
          client_id: "different-client-id",
          client_type: "confidential",
          redirect_uris: ["https://example.com/callback"]
        }

        params = %{
          client_id: client_id,
          redirect_uri: "https://example.com/callback"
        }

        assert {:error, :invalid_client} = Authorization.validate_request(params, client)
      end
    end
  end

  describe "OIDC Token Claims" do
    # **Feature: auth-microservices-platform, Property 15: OIDC Token Claims Completeness**
    # **Validates: Requirements 4.6**
    property "id_token contains all standard claims (sub, iss, aud, exp, iat)" do
      check all user_id <- uuid_generator(),
                issuer <- StreamData.string(:alphanumeric, min_length: 5, max_length: 50),
                audience <- StreamData.string(:alphanumeric, min_length: 5, max_length: 50),
                ttl <- StreamData.integer(300..86400),
                max_runs: 100 do
        
        {:ok, token} = IdToken.build(user_id, 
          issuer: issuer,
          audience: [audience],
          ttl: ttl
        )

        # Verify all required claims are present
        assert token.sub == user_id
        assert token.iss == issuer
        assert token.aud == [audience]
        assert token.exp != nil
        assert token.iat != nil
        assert token.exp > token.iat

        # Validate using the validation function
        assert IdToken.validate(token) == :ok
        assert IdToken.has_standard_claims?(token) == true
      end
    end

    property "id_token includes nonce when provided" do
      check all user_id <- uuid_generator(),
                nonce <- StreamData.string(:alphanumeric, min_length: 16, max_length: 64),
                max_runs: 100 do
        
        {:ok, token} = IdToken.build(user_id, nonce: nonce)

        assert token.nonce == nonce
        assert IdToken.has_nonce_if_provided?(token, nonce) == true
      end
    end

    property "id_token claims can be converted to map" do
      check all user_id <- uuid_generator(),
                max_runs: 100 do
        
        {:ok, token} = IdToken.build(user_id, 
          issuer: "test-issuer",
          audience: ["test-audience"]
        )

        claims = IdToken.to_claims(token)

        assert is_map(claims)
        assert claims.sub == user_id
        assert claims.iss == "test-issuer"
        assert claims.aud == ["test-audience"]
        assert is_integer(claims.exp)
        assert is_integer(claims.iat)
      end
    end
  end

  # Generators

  defp uuid_generator do
    StreamData.map(StreamData.constant(nil), fn _ -> Ecto.UUID.generate() end)
  end

  defp pkce_challenge_generator do
    # Generate a valid S256 challenge (43 chars, base64url)
    StreamData.map(StreamData.binary(length: 32), fn bytes ->
      :crypto.hash(:sha256, bytes)
      |> Base.url_encode64(padding: false)
    end)
  end
end
