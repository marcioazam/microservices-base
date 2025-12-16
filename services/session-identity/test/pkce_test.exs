defmodule SessionIdentityCore.OAuth.PKCETest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.OAuth.PKCE

  describe "PKCE Verification" do
    # **Feature: auth-microservices-platform, Property 14: PKCE Verification Correctness**
    # **Validates: Requirements 4.4**
    property "S256(code_verifier) equals code_challenge for valid pairs" do
      check all verifier <- valid_code_verifier_generator(),
                max_runs: 100 do
        
        # Compute challenge from verifier
        challenge = PKCE.compute_s256_challenge(verifier)

        # Verification should succeed
        assert PKCE.verify(verifier, challenge, "S256") == :ok
      end
    end

    property "verification fails for mismatched verifier/challenge" do
      check all verifier1 <- valid_code_verifier_generator(),
                verifier2 <- valid_code_verifier_generator(),
                verifier1 != verifier2,
                max_runs: 100 do
        
        # Compute challenge from verifier1
        challenge = PKCE.compute_s256_challenge(verifier1)

        # Verification with verifier2 should fail
        assert PKCE.verify(verifier2, challenge, "S256") == {:error, :invalid_code_verifier}
      end
    end

    property "code_verifier validation accepts valid format" do
      check all verifier <- valid_code_verifier_generator(),
                max_runs: 100 do
        
        assert PKCE.validate_code_verifier(verifier) == :ok
      end
    end

    property "code_verifier validation rejects too short" do
      check all verifier <- StreamData.string(:alphanumeric, min_length: 1, max_length: 42),
                max_runs: 100 do
        
        assert PKCE.validate_code_verifier(verifier) == {:error, :code_verifier_too_short}
      end
    end

    property "code_verifier validation rejects too long" do
      check all verifier <- StreamData.string(:alphanumeric, min_length: 129, max_length: 200),
                max_runs: 100 do
        
        assert PKCE.validate_code_verifier(verifier) == {:error, :code_verifier_too_long}
      end
    end
  end

  describe "Unit tests" do
    test "compute_s256_challenge produces 43 character output" do
      verifier = String.duplicate("a", 43)
      challenge = PKCE.compute_s256_challenge(verifier)

      assert String.length(challenge) == 43
    end

    test "plain method is not allowed" do
      assert PKCE.verify("verifier", "verifier", "plain") == {:error, :plain_method_not_allowed}
    end

    test "unsupported method returns error" do
      assert PKCE.verify("verifier", "challenge", "unknown") == {:error, :unsupported_method}
    end
  end

  # Generator for valid code verifiers (43-128 chars, alphanumeric + -._~)
  defp valid_code_verifier_generator do
    StreamData.string(
      Enum.concat([?a..?z, ?A..?Z, ?0..?9, [?-, ?., ?_, ?~]]),
      min_length: 43,
      max_length: 128
    )
  end
end
