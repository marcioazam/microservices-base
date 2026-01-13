defmodule SessionIdentityCore.Shared.KeysPropertyTest do
  @moduledoc """
  Property tests for session key namespacing.
  
  Property 8: Session Key Namespacing
  """

  use ExUnit.Case, async: true
  use ExUnitProperties

  alias SessionIdentityCore.Shared.Keys
  alias SessionIdentityCore.Test.Generators

  @iterations 100

  describe "Property 8: Session Key Namespacing" do
    property "session keys start with 'session:' prefix" do
      check all(session_id <- Generators.uuid(), max_runs: @iterations) do
        key = Keys.session_key(session_id)

        assert String.starts_with?(key, "session:")
        assert key == "session:#{session_id}"
      end
    end

    property "user_sessions keys start with 'user_sessions:' prefix" do
      check all(user_id <- Generators.uuid(), max_runs: @iterations) do
        key = Keys.user_sessions_key(user_id)

        assert String.starts_with?(key, "user_sessions:")
        assert key == "user_sessions:#{user_id}"
      end
    end

    property "oauth_code keys start with 'oauth_code:' prefix" do
      check all(code <- string(:alphanumeric, min_length: 32, max_length: 64), max_runs: @iterations) do
        key = Keys.oauth_code_key(code)

        assert String.starts_with?(key, "oauth_code:")
        assert key == "oauth_code:#{code}"
      end
    end

    property "event keys start with 'events:' prefix" do
      check all(event_id <- Generators.uuid(), max_runs: @iterations) do
        key = Keys.event_key(event_id)

        assert String.starts_with?(key, "events:")
        assert key == "events:#{event_id}"
      end
    end

    property "refresh_token keys start with 'refresh_token:' prefix" do
      check all(token_id <- Generators.uuid(), max_runs: @iterations) do
        key = Keys.refresh_token_key(token_id)

        assert String.starts_with?(key, "refresh_token:")
        assert key == "refresh_token:#{token_id}"
      end
    end

    property "aggregate keys have proper format" do
      check all(
              aggregate_type <- member_of(["Session", "User", "Token"]),
              aggregate_id <- Generators.uuid(),
              max_runs: @iterations
            ) do
        key = Keys.aggregate_key(aggregate_type, aggregate_id)

        assert String.starts_with?(key, "aggregate:")
        assert key == "aggregate:#{aggregate_type}:#{aggregate_id}"
      end
    end

    property "different IDs produce different keys" do
      check all(
              id1 <- Generators.uuid(),
              id2 <- Generators.uuid(),
              id1 != id2,
              max_runs: @iterations
            ) do
        key1 = Keys.session_key(id1)
        key2 = Keys.session_key(id2)

        assert key1 != key2
      end
    end

    property "keys are deterministic" do
      check all(session_id <- Generators.uuid(), max_runs: @iterations) do
        key1 = Keys.session_key(session_id)
        key2 = Keys.session_key(session_id)

        assert key1 == key2
      end
    end
  end
end
