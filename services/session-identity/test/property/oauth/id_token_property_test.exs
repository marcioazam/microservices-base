defmodule SessionIdentityCore.OAuth.IdTokenPropertyTest do
  @moduledoc """
  Property tests for ID token claims.
  
  Property 11: ID Token Claims Completeness
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.IdToken
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 11: ID Token Claims Completeness" do
    property "all required claims are present" do
      check all(params <- Generators.id_token_params(), max_runs: @iterations) do
        {:ok, claims} = IdToken.build_claims(params)

        assert Map.has_key?(claims, :sub)
        assert Map.has_key?(claims, :iss)
        assert Map.has_key?(claims, :aud)
        assert Map.has_key?(claims, :exp)
        assert Map.has_key?(claims, :iat)

        assert claims.sub == params.sub
        assert claims.iss == params.iss
        assert claims.aud == params.aud
      end
    end

    property "nonce is included when provided" do
      check all(
              sub <- Generators.uuid(),
              aud <- string(:alphanumeric, min_length: 10, max_length: 30),
              nonce <- string(:alphanumeric, min_length: 16, max_length: 32),
              max_runs: @iterations
            ) do
        params = %{
          sub: sub,
          iss: "https://auth.example.com",
          aud: aud,
          nonce: nonce
        }

        {:ok, claims} = IdToken.build_claims(params)

        assert Map.has_key?(claims, :nonce)
        assert claims.nonce == nonce
      end
    end

    property "nonce is not included when not provided" do
      check all(
              sub <- Generators.uuid(),
              aud <- string(:alphanumeric, min_length: 10, max_length: 30),
              max_runs: @iterations
            ) do
        params = %{
          sub: sub,
          iss: "https://auth.example.com",
          aud: aud
        }

        {:ok, claims} = IdToken.build_claims(params)

        refute Map.has_key?(claims, :nonce)
      end
    end

    property "exp equals iat + ttl" do
      check all(
              sub <- Generators.uuid(),
              aud <- string(:alphanumeric, min_length: 10, max_length: 30),
              ttl <- integer(60..7200),
              max_runs: @iterations
            ) do
        params = %{
          sub: sub,
          iss: "https://auth.example.com",
          aud: aud,
          ttl: ttl
        }

        {:ok, claims} = IdToken.build_claims(params)

        assert claims.exp == claims.iat + ttl
      end
    end

    property "missing required claims fails validation" do
      check all(
              sub <- one_of([constant(nil), Generators.uuid()]),
              iss <- one_of([constant(nil), constant("https://auth.example.com")]),
              aud <- one_of([constant(nil), string(:alphanumeric, min_length: 10, max_length: 30)]),
              is_nil(sub) or is_nil(iss) or is_nil(aud),
              max_runs: @iterations
            ) do
        params = %{sub: sub, iss: iss, aud: aud}

        result = IdToken.build_claims(params)

        assert {:error, {:missing_claims, _}} = result
      end
    end

    property "optional claims are included when provided" do
      check all(
              sub <- Generators.uuid(),
              aud <- string(:alphanumeric, min_length: 10, max_length: 30),
              auth_time <- integer(1_700_000_000..1_800_000_000),
              acr <- member_of(["urn:mace:incommon:iap:silver", "urn:mace:incommon:iap:bronze"]),
              amr <- list_of(member_of(["pwd", "otp", "mfa"]), min_length: 1, max_length: 3),
              azp <- string(:alphanumeric, min_length: 10, max_length: 30),
              max_runs: @iterations
            ) do
        params = %{
          sub: sub,
          iss: "https://auth.example.com",
          aud: aud,
          auth_time: auth_time,
          acr: acr,
          amr: amr,
          azp: azp
        }

        {:ok, claims} = IdToken.build_claims(params)

        assert claims.auth_time == auth_time
        assert claims.acr == acr
        assert claims.amr == amr
        assert claims.azp == azp
      end
    end
  end
end
