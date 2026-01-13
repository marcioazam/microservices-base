defmodule SessionIdentityCore.Crypto.JWTSignerPropertyTest do
  @moduledoc """
  Property tests for JWT signing and verification.
  
  **Property 5: JWT Signing Round-Trip**
  **Validates: Requirements 2.1, 2.3**
  """

  use ExUnit.Case, async: false
  use ExUnitProperties

  alias SessionIdentityCore.Crypto.{JWTSigner, Fallback}

  # Note: These tests use the local fallback since crypto-service
  # may not be available in test environment

  setup do
    # Configure to use fallback for testing
    Application.put_env(:session_identity_core, :crypto_config, %{
      enabled: false,
      fallback_enabled: true,
      jwt_key_namespace: "session_identity:jwt",
      jwt_algorithm: :ecdsa_p256
    })
    
    # Set a consistent local JWT secret for testing
    Application.put_env(:session_identity_core, :local_jwt_secret, :crypto.strong_rand_bytes(32))
    
    :ok
  end

  # Generators

  defp jwt_claims do
    gen all sub <- string(:alphanumeric, min_length: 1, max_length: 50),
            iss <- string(:alphanumeric, min_length: 1, max_length: 50),
            aud <- string(:alphanumeric, min_length: 1, max_length: 50),
            extra_claims <- map_of(
              atom(:alphanumeric),
              one_of([string(:alphanumeric), integer(), boolean()]),
              max_length: 5
            ) do
      Map.merge(extra_claims, %{
        sub: sub,
        iss: iss,
        aud: aud
      })
    end
  end

  defp simple_claims do
    gen all sub <- string(:alphanumeric, min_length: 1, max_length: 20),
            iss <- string(:alphanumeric, min_length: 1, max_length: 20) do
      %{sub: sub, iss: iss}
    end
  end

  # Property Tests

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "JWT round-trip preserves core claims (fallback mode)" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token} = Fallback.sign_jwt_local(claims)
      {:ok, decoded} = Fallback.verify_jwt_local(token)
      
      # Core claims should be preserved (keys may be strings or atoms)
      assert to_string(decoded[:sub] || decoded["sub"]) == to_string(claims[:sub])
      assert to_string(decoded[:iss] || decoded["iss"]) == to_string(claims[:iss])
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "signed JWT has three parts separated by dots" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token} = Fallback.sign_jwt_local(claims)
      
      parts = String.split(token, ".")
      assert length(parts) == 3
      
      # Each part should be valid base64url
      for part <- parts do
        assert {:ok, _} = Base.url_decode64(part, padding: false)
      end
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "verification fails for tampered tokens" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token} = Fallback.sign_jwt_local(claims)
      
      # Tamper with the payload
      [header, payload, signature] = String.split(token, ".")
      tampered_payload = payload <> "x"
      tampered_token = "#{header}.#{tampered_payload}.#{signature}"
      
      result = Fallback.verify_jwt_local(tampered_token)
      assert {:error, _} = result
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "verification fails for invalid signature" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token} = Fallback.sign_jwt_local(claims)
      
      # Replace signature with random bytes
      [header, payload, _signature] = String.split(token, ".")
      fake_signature = Base.url_encode64(:crypto.strong_rand_bytes(32), padding: false)
      tampered_token = "#{header}.#{payload}.#{fake_signature}"
      
      result = Fallback.verify_jwt_local(tampered_token)
      assert {:error, _} = result
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "different claims produce different tokens" do
    check all claims1 <- simple_claims(),
              claims2 <- simple_claims(),
              claims1 != claims2,
              max_runs: 100 do
      {:ok, token1} = Fallback.sign_jwt_local(claims1)
      {:ok, token2} = Fallback.sign_jwt_local(claims2)
      
      assert token1 != token2
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "same claims with same secret produce verifiable tokens" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token1} = Fallback.sign_jwt_local(claims)
      {:ok, token2} = Fallback.sign_jwt_local(claims)
      
      # Both should be verifiable
      assert {:ok, _} = Fallback.verify_jwt_local(token1)
      assert {:ok, _} = Fallback.verify_jwt_local(token2)
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "JWT header contains alg and typ" do
    check all claims <- simple_claims(), max_runs: 100 do
      {:ok, token} = Fallback.sign_jwt_local(claims)
      
      [header_b64, _, _] = String.split(token, ".")
      {:ok, header_json} = Base.url_decode64(header_b64, padding: false)
      {:ok, header} = Jason.decode(header_json)
      
      assert Map.has_key?(header, "alg")
      assert Map.has_key?(header, "typ")
      assert header["typ"] == "JWT"
    end
  end

  @tag property: true
  @tag validates: "Requirements 2.1, 2.3"
  property "verification returns error for malformed tokens" do
    check all invalid <- one_of([
              string(:alphanumeric, min_length: 1, max_length: 50),
              constant("not.a.valid.jwt"),
              constant(""),
              constant("a.b")
            ]),
            max_runs: 100 do
      result = Fallback.verify_jwt_local(invalid)
      assert {:error, _} = result
    end
  end
end
