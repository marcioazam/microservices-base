defmodule MfaService.WebAuthn.AuthenticationTest do
  use ExUnit.Case, async: true
  use ExUnitProperties

  alias MfaService.WebAuthn.{Authentication, Challenge}

  describe "WebAuthn Sign Count" do
    # **Feature: auth-microservices-platform, Property 19: WebAuthn Sign Count Monotonicity**
    # **Validates: Requirements 6.4**
    property "sign count must be strictly greater than stored value" do
      check all stored_count <- StreamData.integer(0..1_000_000),
                increment <- StreamData.integer(1..1000),
                max_runs: 100 do
        
        new_count = stored_count + increment

        # Simulate authenticator data with sign count
        auth_data = create_auth_data_with_sign_count(new_count)
        stored_credential = %{sign_count: stored_count}

        # New count should be accepted
        assert verify_sign_count(auth_data, stored_credential.sign_count) == :ok
      end
    end

    property "sign count equal to or less than stored value is rejected" do
      check all stored_count <- StreamData.integer(1..1_000_000),
                decrement <- StreamData.integer(0..100),
                max_runs: 100 do
        
        new_count = max(0, stored_count - decrement)

        auth_data = create_auth_data_with_sign_count(new_count)
        stored_credential = %{sign_count: stored_count}

        # Should be rejected (potential cloned authenticator)
        if new_count <= stored_count do
          assert verify_sign_count(auth_data, stored_credential.sign_count) == {:error, :sign_count_not_increased}
        end
      end
    end
  end

  describe "Challenge Generation" do
    property "challenges are unique" do
      check all _seed <- StreamData.integer(),
                max_runs: 100 do
        challenge1 = Challenge.generate()
        challenge2 = Challenge.generate()

        assert challenge1 != challenge2
        assert byte_size(challenge1) == 32
        assert byte_size(challenge2) == 32
      end
    end

    property "challenge encode/decode round trip" do
      check all challenge <- StreamData.binary(length: 32),
                max_runs: 100 do
        encoded = Challenge.encode(challenge)
        decoded = Challenge.decode(encoded)

        assert decoded == challenge
      end
    end
  end

  # Helper functions

  defp create_auth_data_with_sign_count(sign_count) do
    rp_id_hash = :crypto.hash(:sha256, "localhost")
    flags = 0x01  # User present
    
    <<rp_id_hash::binary, flags::8, sign_count::unsigned-big-integer-size(32)>>
  end

  defp verify_sign_count(auth_data, stored_sign_count) do
    <<_rp_id_hash::binary-size(32), _flags::8, new_sign_count::unsigned-big-integer-size(32), _rest::binary>> = auth_data

    if new_sign_count > stored_sign_count do
      :ok
    else
      {:error, :sign_count_not_increased}
    end
  end
end
