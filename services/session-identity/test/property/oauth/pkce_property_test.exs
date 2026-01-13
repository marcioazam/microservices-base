defmodule SessionIdentityCore.OAuth.PKCEPropertyTest do
  @moduledoc """
  Property tests for PKCE verification correctness.
  
  Property 2: PKCE Verification Correctness
  - For any valid code_verifier, computing S256 and verifying SHALL succeed
  - For any two different code_verifiers, cross-verification SHALL fail
  - For any string outside 43-128 chars, validation SHALL reject
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.PKCE
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 2: PKCE Verification Correctness" do
    property "valid code_verifier produces verifiable challenge" do
      check all(verifier <- Generators.code_verifier(), max_runs: @iterations) do
        challenge = PKCE.compute_s256_challenge(verifier)
        assert :ok = PKCE.verify(verifier, challenge, "S256")
      end
    end

    property "different verifiers produce different challenges" do
      check all(
              verifier1 <- Generators.code_verifier(),
              verifier2 <- Generators.code_verifier(),
              verifier1 != verifier2,
              max_runs: @iterations
            ) do
        challenge1 = PKCE.compute_s256_challenge(verifier1)
        challenge2 = PKCE.compute_s256_challenge(verifier2)
        assert challenge1 != challenge2
      end
    end

    property "cross-verification fails" do
      check all(
              verifier1 <- Generators.code_verifier(),
              verifier2 <- Generators.code_verifier(),
              verifier1 != verifier2,
              max_runs: @iterations
            ) do
        challenge1 = PKCE.compute_s256_challenge(verifier1)
        assert {:error, :invalid_code_verifier} = PKCE.verify(verifier2, challenge1, "S256")
      end
    end

    property "short verifiers are rejected" do
      check all(verifier <- Generators.short_code_verifier(), max_runs: @iterations) do
        assert {:error, :code_verifier_too_short} = PKCE.validate_code_verifier(verifier)
      end
    end

    property "long verifiers are rejected" do
      check all(verifier <- Generators.long_code_verifier(), max_runs: @iterations) do
        assert {:error, :code_verifier_too_long} = PKCE.validate_code_verifier(verifier)
      end
    end

    property "valid verifiers pass validation" do
      check all(verifier <- Generators.code_verifier(), max_runs: @iterations) do
        assert :ok = PKCE.validate_code_verifier(verifier)
      end
    end

    property "S256 challenge is exactly 43 characters" do
      check all(verifier <- Generators.code_verifier(), max_runs: @iterations) do
        challenge = PKCE.compute_s256_challenge(verifier)
        assert String.length(challenge) == 43
      end
    end

    property "plain method is always rejected" do
      check all(verifier <- Generators.code_verifier(), max_runs: @iterations) do
        challenge = PKCE.compute_s256_challenge(verifier)
        assert {:error, :plain_method_not_allowed} = PKCE.verify(verifier, challenge, "plain")
      end
    end
  end
end
