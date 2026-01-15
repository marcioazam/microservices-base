"""Vault-integrated settings loader.

This module provides settings loading with HashiCorp Vault integration.
Secrets can be loaded from Vault while other configuration comes from environment variables.
"""

import logging
import os
from typing import Optional
from urllib.parse import urlparse, urlunparse

from src.config.settings import Settings, get_settings
from src.shared.vault_client import VaultClient, get_vault_client

logger = logging.getLogger(__name__)


class VaultSettings:
    """Settings loader with Vault integration."""

    def __init__(
        self,
        vault_enabled: bool = True,
        vault_path: str = "sms-service",
        fallback_to_env: bool = True,
    ):
        """
        Initialize Vault settings loader.

        Args:
            vault_enabled: Whether to use Vault for secrets (default: True)
            vault_path: Path to secrets in Vault (default: "sms-service")
            fallback_to_env: Fallback to environment variables if Vault fails (default: True)
        """
        self.vault_enabled = vault_enabled and self._is_vault_configured()
        self.vault_path = vault_path
        self.fallback_to_env = fallback_to_env

        # Load base settings from environment
        self.base_settings = get_settings()

        # Load secrets from Vault if enabled
        if self.vault_enabled:
            try:
                self._load_secrets_from_vault()
                logger.info("Successfully loaded secrets from Vault")
            except Exception as e:
                logger.error(f"Failed to load secrets from Vault: {e}")
                if not self.fallback_to_env:
                    raise
                logger.warning("Falling back to environment variables")

    def _is_vault_configured(self) -> bool:
        """Check if Vault is properly configured."""
        vault_addr = os.getenv("VAULT_ADDR")
        vault_token = os.getenv("VAULT_TOKEN")

        if not vault_addr or not vault_token:
            logger.info(
                "Vault not configured (missing VAULT_ADDR or VAULT_TOKEN). "
                "Using environment variables for secrets."
            )
            return False

        return True

    def _load_secrets_from_vault(self) -> None:
        """Load secrets from Vault and override settings."""
        vault = get_vault_client()

        # Load all secrets from the configured path
        secrets = vault.get_secret(self.vault_path)

        # Map Vault secrets to settings attributes
        secret_mapping = {
            "jwt_secret_key": "jwt_secret_key",
            "twilio_auth_token": "twilio_auth_token",
            "twilio_webhook_secret": "twilio_webhook_secret",
            "messagebird_api_key": "messagebird_api_key",
            "messagebird_webhook_secret": "messagebird_webhook_secret",
            "database_password": "_database_password",  # Used to reconstruct database_url
        }

        # Override settings with Vault secrets
        for vault_key, settings_key in secret_mapping.items():
            if vault_key in secrets:
                setattr(self.base_settings, settings_key, secrets[vault_key])
                logger.debug(f"Loaded {vault_key} from Vault")

        # Reconstruct database URL if password was loaded from Vault
        if "_database_password" in dir(self.base_settings):
            self._update_database_url(self.base_settings._database_password)

    def _update_database_url(self, password: str) -> None:
        """
        Update database URL with password from Vault.

        Uses urllib.parse for robust URL manipulation that handles edge cases like
        IPv6 addresses, special characters, and various URL formats.

        Args:
            password: Database password from Vault
        """
        try:
            current_url = str(self.base_settings.database_url)

            # Parse URL using urllib.parse for robustness
            parsed = urlparse(current_url)

            if not parsed.username:
                logger.warning(
                    "Database URL has no username, cannot update password",
                    extra={"current_url_scheme": parsed.scheme},
                )
                return

            # Reconstruct netloc with new password
            # Format: user:password@host:port (handles IPv6 and standard hosts)
            if parsed.port:
                new_netloc = f"{parsed.username}:{password}@{parsed.hostname}:{parsed.port}"
            else:
                new_netloc = f"{parsed.username}:{password}@{parsed.hostname}"

            # Reconstruct URL with new netloc
            new_url = urlunparse(
                (
                    parsed.scheme,
                    new_netloc,
                    parsed.path,
                    parsed.params,
                    parsed.query,
                    parsed.fragment,
                )
            )

            self.base_settings.database_url = new_url
            logger.debug("Updated database URL with Vault password")

        except Exception as e:
            logger.error(
                "Failed to update database URL with Vault password",
                extra={
                    "error": str(e),
                    "vault_path": self.vault_path,
                },
            )
            # Don't raise - fall back to original URL to avoid breaking the application

    def get_settings(self) -> Settings:
        """
        Get settings with Vault secrets loaded.

        Returns:
            Settings instance with secrets from Vault
        """
        return self.base_settings

    def reload_secrets(self) -> None:
        """Reload secrets from Vault (useful for token rotation)."""
        if self.vault_enabled:
            try:
                self._load_secrets_from_vault()
                logger.info("Successfully reloaded secrets from Vault")
            except Exception as e:
                logger.error(f"Failed to reload secrets from Vault: {e}")
                if not self.fallback_to_env:
                    raise


def get_settings_with_vault(
    vault_enabled: Optional[bool] = None,
    vault_path: str = "sms-service",
) -> Settings:
    """
    Get settings with Vault integration.

    This is the recommended way to load settings when using Vault.

    Args:
        vault_enabled: Whether to use Vault (default: auto-detect from env)
        vault_path: Path to secrets in Vault (default: "sms-service")

    Returns:
        Settings instance with secrets loaded from Vault

    Example:
        >>> from src.config.vault_settings import get_settings_with_vault
        >>> settings = get_settings_with_vault()
        >>> # JWT secret will be loaded from Vault if configured
        >>> jwt_secret = settings.jwt_secret_key
    """
    if vault_enabled is None:
        vault_enabled = os.getenv("VAULT_ENABLED", "true").lower() == "true"

    vault_settings = VaultSettings(
        vault_enabled=vault_enabled,
        vault_path=vault_path,
        fallback_to_env=True,
    )

    return vault_settings.get_settings()
