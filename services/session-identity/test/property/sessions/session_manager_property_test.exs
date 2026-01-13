defmodule SessionIdentityCore.Sessions.SessionManagerPropertyTest do
  @moduledoc """
  Property tests for session management.
  
  Property 5: Session Token Entropy
  Property 6: Session Device Binding
  Property 7: Session Events Correlation
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Sessions.SessionManager
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 5: Session Token Entropy" do
    property "session tokens have 256-bit entropy (43+ chars base64url)" do
      check all(_ <- constant(nil), max_runs: @iterations) do
        token = SessionManager.generate_session_token()

        # 32 bytes = 256 bits, base64url encoded = 43 chars
        assert String.length(token) >= 43
        assert Regex.match?(~r/^[A-Za-z0-9_-]+$/, token)
      end
    end

    property "session tokens are unique" do
      check all(_ <- constant(nil), max_runs: @iterations) do
        token1 = SessionManager.generate_session_token()
        token2 = SessionManager.generate_session_token()

        assert token1 != token2
      end
    end

    property "session tokens are cryptographically random" do
      # Generate multiple tokens and verify no patterns
      tokens = for _ <- 1..100, do: SessionManager.generate_session_token()

      # All tokens should be unique
      assert length(Enum.uniq(tokens)) == 100

      # No common prefixes (first 8 chars should vary)
      prefixes = Enum.map(tokens, &String.slice(&1, 0, 8))
      assert length(Enum.uniq(prefixes)) > 90
    end
  end

  describe "Property 6: Session Device Binding" do
    property "sessions require device_fingerprint" do
      check all(
              user_id <- Generators.uuid(),
              ip_address <- Generators.ip_address(),
              max_runs: @iterations
            ) do
        attrs = %{
          user_id: user_id,
          ip_address: ip_address
          # Missing device_fingerprint
        }

        result = SessionManager.create_session(attrs)

        assert {:error, :missing_device_binding} = result
      end
    end

    property "sessions require ip_address" do
      check all(
              user_id <- Generators.uuid(),
              device_fingerprint <- string(:alphanumeric, min_length: 32, max_length: 64),
              max_runs: @iterations
            ) do
        attrs = %{
          user_id: user_id,
          device_fingerprint: device_fingerprint
          # Missing ip_address
        }

        result = SessionManager.create_session(attrs)

        assert {:error, :missing_device_binding} = result
      end
    end
  end
end
