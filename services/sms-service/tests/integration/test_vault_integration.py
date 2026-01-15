"""Integration tests for HashiCorp Vault."""

import os
import pytest
from unittest.mock import patch, MagicMock

# Skip tests if Vault is not configured
pytestmark = pytest.mark.skipif(
    not os.getenv("VAULT_ADDR") or not os.getenv("VAULT_TOKEN"),
    reason="Vault not configured (missing VAULT_ADDR or VAULT_TOKEN)",
)


class TestVaultClient:
    """Integration tests for VaultClient."""

    def test_vault_connection(self):
        """Test basic Vault connectivity."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()
        assert vault.is_authenticated()

    def test_read_secret(self):
        """Test reading a secret from Vault."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()

        # Read JWT secret
        jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")

        assert jwt_secret is not None
        assert isinstance(jwt_secret, str)
        assert len(jwt_secret) >= 32  # Minimum length enforced

    def test_read_all_secrets(self):
        """Test reading all secrets from a path."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()

        # Read all sms-service secrets
        secrets = vault.get_secret("sms-service")

        assert isinstance(secrets, dict)
        assert "jwt_secret_key" in secrets

    def test_read_secrets_batch(self):
        """Test reading multiple specific secrets."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()

        # Read specific secrets
        secrets = vault.get_secrets_batch(
            "sms-service", ["jwt_secret_key", "twilio_auth_token"]
        )

        assert isinstance(secrets, dict)
        assert "jwt_secret_key" in secrets

    def test_secret_not_found(self):
        """Test handling of non-existent secret path."""
        from src.shared.vault_client import VaultClient
        from hvac.exceptions import InvalidPath

        vault = VaultClient()

        with pytest.raises(InvalidPath):
            vault.get_secret("non-existent-path")

    def test_key_not_found(self):
        """Test handling of non-existent secret key."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()

        with pytest.raises(KeyError) as exc_info:
            vault.get_secret("sms-service", "non_existent_key")

        assert "not found" in str(exc_info.value).lower()

    def test_token_ttl(self):
        """Test getting token TTL."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()
        ttl = vault.get_token_ttl()

        assert isinstance(ttl, int)
        assert ttl > 0  # Token should have remaining time

    def test_put_and_delete_secret(self):
        """Test writing and deleting secrets."""
        from src.shared.vault_client import VaultClient

        vault = VaultClient()

        # Write temporary secret
        test_path = "sms-service/test"
        test_secrets = {"test_key": "test_value"}

        vault.put_secret(test_path, test_secrets)

        # Read it back
        retrieved = vault.get_secret(test_path, "test_key")
        assert retrieved == "test_value"

        # Clean up
        vault.delete_secret(test_path)

        # Verify deletion
        from hvac.exceptions import InvalidPath

        with pytest.raises(InvalidPath):
            vault.get_secret(test_path)


class TestVaultSettings:
    """Integration tests for Vault-integrated settings."""

    def test_settings_load_from_vault(self):
        """Test loading settings with Vault integration."""
        from src.config.vault_settings import get_settings_with_vault

        settings = get_settings_with_vault()

        # JWT secret should be loaded from Vault
        assert settings.jwt_secret_key is not None
        assert len(settings.jwt_secret_key) >= 32

    def test_settings_fallback_to_env(self, monkeypatch):
        """Test fallback to environment variables when Vault fails."""
        from src.config.vault_settings import get_settings_with_vault

        # Temporarily break Vault connection
        monkeypatch.setenv("VAULT_TOKEN", "invalid_token")

        # Should fallback to environment variables
        with patch("src.config.vault_settings.get_vault_client") as mock_vault:
            mock_vault.side_effect = Exception("Vault connection failed")

            # Set JWT secret in environment
            monkeypatch.setenv("JWT_SECRET_KEY", "a" * 32)

            settings = get_settings_with_vault()

            # Should have fallback value from environment
            assert settings.jwt_secret_key == "a" * 32

    def test_settings_vault_disabled(self, monkeypatch):
        """Test settings when Vault is explicitly disabled."""
        from src.config.vault_settings import get_settings_with_vault

        # Disable Vault
        monkeypatch.setenv("VAULT_ENABLED", "false")
        monkeypatch.setenv("JWT_SECRET_KEY", "b" * 32)

        settings = get_settings_with_vault(vault_enabled=False)

        # Should load from environment only
        assert settings.jwt_secret_key == "b" * 32


class TestVaultCaching:
    """Test Vault client caching behavior."""

    def test_vault_client_singleton(self):
        """Test that get_vault_client returns the same instance."""
        from src.shared.vault_client import get_vault_client

        # Clear cache first
        get_vault_client.cache_clear()

        client1 = get_vault_client()
        client2 = get_vault_client()

        # Should be the same instance (cached)
        assert client1 is client2

    def test_cache_invalidation(self):
        """Test cache invalidation."""
        from src.shared.vault_client import get_vault_client

        # Get client
        client1 = get_vault_client()

        # Clear cache
        get_vault_client.cache_clear()

        # Get new client
        client2 = get_vault_client()

        # Should be different instances
        assert client1 is not client2


class TestVaultErrorHandling:
    """Test Vault error handling."""

    def test_missing_vault_addr(self, monkeypatch):
        """Test error when VAULT_ADDR is missing."""
        from src.shared.vault_client import VaultClient

        # Remove VAULT_ADDR
        monkeypatch.delenv("VAULT_ADDR", raising=False)
        monkeypatch.setenv("VAULT_TOKEN", "test_token")

        # Should use default
        vault = VaultClient()
        assert vault.vault_addr == "http://localhost:8200"

    def test_missing_vault_token(self, monkeypatch):
        """Test error when VAULT_TOKEN is missing."""
        from src.shared.vault_client import VaultClient

        # Remove VAULT_TOKEN
        monkeypatch.delenv("VAULT_TOKEN", raising=False)

        # Should raise error
        with pytest.raises(ValueError) as exc_info:
            VaultClient()

        assert "token is required" in str(exc_info.value).lower()

    def test_invalid_vault_token(self, monkeypatch):
        """Test handling of invalid Vault token."""
        from src.shared.vault_client import VaultClient
        from hvac.exceptions import VaultError

        # Set invalid token
        monkeypatch.setenv("VAULT_TOKEN", "invalid_token_12345")

        # Should raise error during authentication check
        with pytest.raises(VaultError):
            VaultClient()


class TestVaultSecretValidation:
    """Test secret validation after loading from Vault."""

    def test_jwt_secret_meets_requirements(self):
        """Test that JWT secret from Vault meets security requirements."""
        from src.config.vault_settings import get_settings_with_vault

        settings = get_settings_with_vault()

        # Should pass all validation checks
        assert len(settings.jwt_secret_key) >= 32
        assert not any(
            pattern in settings.jwt_secret_key.lower()
            for pattern in ["change-me", "secret", "password", "test"]
        )

    def test_secrets_are_not_logged(self, caplog):
        """Test that secrets are not logged in plaintext."""
        from src.config.vault_settings import get_settings_with_vault

        with caplog.at_level("DEBUG"):
            settings = get_settings_with_vault()

            # Check that actual secret value is not in logs
            secret = settings.jwt_secret_key
            assert secret not in caplog.text

            # But should mention loading from Vault
            assert "vault" in caplog.text.lower() or "loaded" in caplog.text.lower()


@pytest.fixture(scope="session")
def vault_test_cleanup():
    """Cleanup test secrets from Vault after test session."""
    yield

    # Cleanup after all tests
    try:
        from src.shared.vault_client import VaultClient

        vault = VaultClient()
        vault.delete_secret("sms-service/test")
    except Exception:
        pass  # Ignore cleanup errors
