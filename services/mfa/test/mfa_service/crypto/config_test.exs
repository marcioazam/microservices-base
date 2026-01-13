defmodule MfaService.Crypto.ConfigTest do
  use ExUnit.Case, async: true

  alias MfaService.Crypto.Config

  describe "host/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_SERVICE_HOST")
      assert Config.host() == "localhost"
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_SERVICE_HOST", "crypto.example.com")
      assert Config.host() == "crypto.example.com"
      System.delete_env("CRYPTO_SERVICE_HOST")
    end
  end

  describe "port/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_SERVICE_PORT")
      assert Config.port() == 50051
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_SERVICE_PORT", "9999")
      assert Config.port() == 9999
      System.delete_env("CRYPTO_SERVICE_PORT")
    end
  end

  describe "address/0" do
    test "returns combined host:port" do
      System.delete_env("CRYPTO_SERVICE_HOST")
      System.delete_env("CRYPTO_SERVICE_PORT")
      assert Config.address() == "localhost:50051"
    end

    test "returns custom address when env vars set" do
      System.put_env("CRYPTO_SERVICE_HOST", "crypto.example.com")
      System.put_env("CRYPTO_SERVICE_PORT", "8080")
      assert Config.address() == "crypto.example.com:8080"
      System.delete_env("CRYPTO_SERVICE_HOST")
      System.delete_env("CRYPTO_SERVICE_PORT")
    end
  end

  describe "connection_timeout/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_CONNECTION_TIMEOUT")
      assert Config.connection_timeout() == 5_000
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_CONNECTION_TIMEOUT", "10000")
      assert Config.connection_timeout() == 10_000
      System.delete_env("CRYPTO_CONNECTION_TIMEOUT")
    end
  end

  describe "request_timeout/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_REQUEST_TIMEOUT")
      assert Config.request_timeout() == 30_000
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_REQUEST_TIMEOUT", "60000")
      assert Config.request_timeout() == 60_000
      System.delete_env("CRYPTO_REQUEST_TIMEOUT")
    end
  end

  describe "key_namespace/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_KEY_NAMESPACE")
      assert Config.key_namespace() == "mfa"
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_KEY_NAMESPACE", "custom-mfa")
      assert Config.key_namespace() == "custom-mfa"
      System.delete_env("CRYPTO_KEY_NAMESPACE")
    end
  end

  describe "totp_key_namespace/0" do
    test "returns namespace with :totp suffix" do
      System.delete_env("CRYPTO_KEY_NAMESPACE")
      assert Config.totp_key_namespace() == "mfa:totp"
    end

    test "uses custom namespace prefix" do
      System.put_env("CRYPTO_KEY_NAMESPACE", "auth")
      assert Config.totp_key_namespace() == "auth:totp"
      System.delete_env("CRYPTO_KEY_NAMESPACE")
    end
  end

  describe "circuit_breaker_threshold/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_CB_THRESHOLD")
      assert Config.circuit_breaker_threshold() == 5
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_CB_THRESHOLD", "10")
      assert Config.circuit_breaker_threshold() == 10
      System.delete_env("CRYPTO_CB_THRESHOLD")
    end
  end

  describe "retry_max_attempts/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_RETRY_ATTEMPTS")
      assert Config.retry_max_attempts() == 3
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_RETRY_ATTEMPTS", "5")
      assert Config.retry_max_attempts() == 5
      System.delete_env("CRYPTO_RETRY_ATTEMPTS")
    end
  end

  describe "cache_ttl/0" do
    test "returns default value when env var not set" do
      System.delete_env("CRYPTO_CACHE_TTL")
      assert Config.cache_ttl() == 300
    end

    test "returns env var value when set" do
      System.put_env("CRYPTO_CACHE_TTL", "600")
      assert Config.cache_ttl() == 600
      System.delete_env("CRYPTO_CACHE_TTL")
    end
  end

  describe "mtls_enabled?/0" do
    test "returns false when env var is 'false'" do
      System.put_env("CRYPTO_MTLS_ENABLED", "false")
      refute Config.mtls_enabled?()
      System.delete_env("CRYPTO_MTLS_ENABLED")
    end

    test "returns true when env var is 'true'" do
      System.put_env("CRYPTO_MTLS_ENABLED", "true")
      assert Config.mtls_enabled?()
      System.delete_env("CRYPTO_MTLS_ENABLED")
    end
  end

  describe "to_map/0" do
    test "returns all configuration as a map" do
      System.delete_env("CRYPTO_SERVICE_HOST")
      System.delete_env("CRYPTO_SERVICE_PORT")
      
      config = Config.to_map()
      
      assert is_map(config)
      assert config.host == "localhost"
      assert config.port == 50051
      assert config.connection_timeout == 5_000
      assert config.request_timeout == 30_000
      assert config.key_namespace == "mfa"
      assert config.circuit_breaker_threshold == 5
      assert config.retry_max_attempts == 3
      assert config.cache_ttl == 300
    end
  end
end
