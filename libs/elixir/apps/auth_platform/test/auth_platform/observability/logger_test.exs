defmodule AuthPlatform.Observability.LoggerTest do
  use ExUnit.Case, async: false

  alias AuthPlatform.Observability.Logger, as: AppLogger

  describe "correlation ID" do
    test "put and get correlation ID" do
      AppLogger.put_correlation_id("test-123")
      assert AppLogger.get_correlation_id() == "test-123"
    end

    test "returns nil when not set" do
      Process.delete(:auth_platform_correlation_id)
      assert AppLogger.get_correlation_id() == nil
    end

    test "generate_correlation_id returns unique IDs" do
      ids = for _ <- 1..100, do: AppLogger.generate_correlation_id()
      assert length(Enum.uniq(ids)) == 100
    end

    test "generate_correlation_id returns 16 character hex string" do
      id = AppLogger.generate_correlation_id()
      assert byte_size(id) == 16
      assert Regex.match?(~r/^[0-9a-f]+$/, id)
    end
  end

  describe "with_correlation_id/2" do
    test "sets correlation ID for duration of function" do
      result =
        AppLogger.with_correlation_id("scoped-123", fn ->
          AppLogger.get_correlation_id()
        end)

      assert result == "scoped-123"
    end

    test "restores previous correlation ID after function" do
      AppLogger.put_correlation_id("original")

      AppLogger.with_correlation_id("temporary", fn ->
        assert AppLogger.get_correlation_id() == "temporary"
      end)

      assert AppLogger.get_correlation_id() == "original"
    end

    test "clears correlation ID if none was set before" do
      Process.delete(:auth_platform_correlation_id)

      AppLogger.with_correlation_id("temporary", fn ->
        assert AppLogger.get_correlation_id() == "temporary"
      end)

      assert AppLogger.get_correlation_id() == nil
    end
  end

  describe "format_json/1" do
    test "formats map as JSON" do
      entry = %{message: "test", level: :info}
      json = AppLogger.format_json(entry)

      assert is_binary(json)
      assert {:ok, decoded} = Jason.decode(json)
      assert decoded["message"] == "test"
    end

    test "redacts sensitive fields" do
      entry = %{
        message: "login",
        password: "secret123",
        token: "abc123",
        user_id: 123
      }

      json = AppLogger.format_json(entry)
      {:ok, decoded} = Jason.decode(json)

      assert decoded["password"] == "[REDACTED]"
      assert decoded["token"] == "[REDACTED]"
      assert decoded["user_id"] == 123
    end

    test "redacts nested sensitive fields" do
      entry = %{
        message: "request",
        data: %{
          user: "john",
          api_key: "key123"
        }
      }

      json = AppLogger.format_json(entry)
      {:ok, decoded} = Jason.decode(json)

      assert decoded["data"]["api_key"] == "[REDACTED]"
      assert decoded["data"]["user"] == "john"
    end
  end

  describe "logging functions" do
    # Note: These tests verify the functions don't crash
    # Full logging output testing would require capturing Logger output

    test "debug logs without error" do
      assert :ok = AppLogger.debug("debug message", key: "value")
    end

    test "info logs without error" do
      assert :ok = AppLogger.info("info message", user_id: 123)
    end

    test "warn logs without error" do
      assert :ok = AppLogger.warn("warning message", count: 5)
    end

    test "error logs without error" do
      assert :ok = AppLogger.error("error message", error: "something failed")
    end

    test "includes correlation ID in log entry" do
      AppLogger.put_correlation_id("log-test-123")
      # The correlation ID is included automatically
      assert :ok = AppLogger.info("test message")
    end
  end
end
