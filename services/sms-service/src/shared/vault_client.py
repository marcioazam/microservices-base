"""HashiCorp Vault Client for Secret Management.

This module provides a secure client for retrieving secrets from HashiCorp Vault.
Supports caching, automatic token renewal, and error handling.
"""

import logging
import os
from functools import lru_cache
from typing import Any, Dict, Optional

import hvac
from hvac.exceptions import InvalidPath, VaultError

logger = logging.getLogger(__name__)


class VaultClient:
    """HashiCorp Vault client for secure secret management."""

    def __init__(
        self,
        vault_addr: Optional[str] = None,
        vault_token: Optional[str] = None,
        vault_namespace: Optional[str] = None,
        mount_point: str = "secret",
    ):
        """
        Initialize Vault client.

        Args:
            vault_addr: Vault server URL (default: from VAULT_ADDR env var)
            vault_token: Vault authentication token (default: from VAULT_TOKEN env var)
            vault_namespace: Vault namespace (default: from VAULT_NAMESPACE env var)
            mount_point: KV secrets engine mount point (default: "secret")
        """
        self.vault_addr = vault_addr or os.getenv("VAULT_ADDR", "http://localhost:8200")
        self.vault_token = vault_token or os.getenv("VAULT_TOKEN", "")
        self.vault_namespace = vault_namespace or os.getenv("VAULT_NAMESPACE", "")
        self.mount_point = mount_point

        if not self.vault_token:
            raise ValueError(
                "Vault token is required. Set VAULT_TOKEN environment variable or pass vault_token parameter."
            )

        # Initialize client
        self.client = hvac.Client(
            url=self.vault_addr,
            token=self.vault_token,
            namespace=self.vault_namespace if self.vault_namespace else None,
        )

        # Verify authentication
        if not self.client.is_authenticated():
            raise VaultError(f"Failed to authenticate with Vault at {self.vault_addr}")

        logger.info(
            f"Vault client initialized successfully",
            extra={
                "vault_addr": self.vault_addr,
                "namespace": self.vault_namespace or "default",
                "authenticated": True,
            },
        )

    def get_secret(self, path: str, key: Optional[str] = None) -> Any:
        """
        Retrieve a secret from Vault.

        Args:
            path: Secret path (e.g., "sms-service" for secret/sms-service)
            key: Specific key to retrieve from the secret (optional)

        Returns:
            Secret value(s)

        Raises:
            InvalidPath: If secret path doesn't exist
            VaultError: If Vault operation fails

        Example:
            >>> vault = VaultClient()
            >>> jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")
            >>> all_secrets = vault.get_secret("sms-service")
        """
        try:
            # Read secret from KV v2
            secret_response = self.client.secrets.kv.v2.read_secret_version(
                path=path,
                mount_point=self.mount_point,
            )

            secret_data = secret_response["data"]["data"]

            # Return specific key or all data
            if key:
                if key not in secret_data:
                    raise KeyError(
                        f"Key '{key}' not found in secret path '{path}'. "
                        f"Available keys: {list(secret_data.keys())}"
                    )
                return secret_data[key]

            return secret_data

        except InvalidPath:
            logger.error(f"Secret path not found: {path}")
            raise
        except VaultError as e:
            logger.error(f"Vault error retrieving secret: {e}")
            raise

    def get_secrets_batch(self, path: str, keys: list[str]) -> Dict[str, Any]:
        """
        Retrieve multiple secrets in a single call.

        Args:
            path: Secret path
            keys: List of keys to retrieve

        Returns:
            Dictionary mapping keys to their values

        Example:
            >>> vault = VaultClient()
            >>> secrets = vault.get_secrets_batch(
            ...     "sms-service",
            ...     ["jwt_secret_key", "twilio_auth_token"]
            ... )
        """
        secret_data = self.get_secret(path)

        result = {}
        missing_keys = []

        for key in keys:
            if key in secret_data:
                result[key] = secret_data[key]
            else:
                missing_keys.append(key)

        if missing_keys:
            logger.warning(
                f"Some keys not found in Vault path '{path}': {missing_keys}"
            )

        return result

    def put_secret(self, path: str, secrets: Dict[str, Any]) -> None:
        """
        Store secrets in Vault.

        Args:
            path: Secret path
            secrets: Dictionary of key-value pairs to store

        Example:
            >>> vault = VaultClient()
            >>> vault.put_secret("sms-service", {
            ...     "jwt_secret_key": "new_secret",
            ...     "api_key": "new_key"
            ... })
        """
        try:
            self.client.secrets.kv.v2.create_or_update_secret(
                path=path,
                secret=secrets,
                mount_point=self.mount_point,
            )
            logger.info(f"Successfully stored secrets at path: {path}")

        except VaultError as e:
            logger.error(f"Failed to store secrets: {e}")
            raise

    def delete_secret(self, path: str) -> None:
        """
        Delete a secret from Vault.

        Args:
            path: Secret path to delete

        Example:
            >>> vault = VaultClient()
            >>> vault.delete_secret("sms-service/temp")
        """
        try:
            self.client.secrets.kv.v2.delete_metadata_and_all_versions(
                path=path,
                mount_point=self.mount_point,
            )
            logger.info(f"Successfully deleted secret at path: {path}")

        except VaultError as e:
            logger.error(f"Failed to delete secret: {e}")
            raise

    def renew_token(self) -> None:
        """
        Renew the Vault token to extend its lease.

        Should be called periodically to keep the token valid.
        """
        try:
            self.client.auth.token.renew_self()
            logger.info("Vault token renewed successfully")

        except VaultError as e:
            logger.error(f"Failed to renew token: {e}")
            raise

    def is_authenticated(self) -> bool:
        """Check if client is authenticated with Vault."""
        return self.client.is_authenticated()

    def get_token_ttl(self) -> int:
        """
        Get remaining TTL for current token in seconds.

        Returns:
            Remaining TTL in seconds
        """
        try:
            lookup = self.client.auth.token.lookup_self()
            return lookup["data"]["ttl"]

        except VaultError as e:
            logger.error(f"Failed to lookup token TTL: {e}")
            return 0


@lru_cache(maxsize=1)
def get_vault_client() -> VaultClient:
    """
    Get cached Vault client instance.

    This function returns a singleton Vault client that is reused across the application.
    The client is cached to avoid repeated authentication.

    Returns:
        VaultClient instance

    Example:
        >>> from src.shared.vault_client import get_vault_client
        >>> vault = get_vault_client()
        >>> jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")
    """
    return VaultClient()


def load_secrets_from_vault(path: str, keys: Optional[list[str]] = None) -> Dict[str, Any]:
    """
    Helper function to load secrets from Vault.

    Args:
        path: Secret path in Vault
        keys: Optional list of specific keys to retrieve

    Returns:
        Dictionary of secrets

    Example:
        >>> secrets = load_secrets_from_vault("sms-service")
        >>> jwt_secret = secrets["jwt_secret_key"]

        >>> # Or load specific keys only
        >>> secrets = load_secrets_from_vault("sms-service", ["jwt_secret_key"])
    """
    vault = get_vault_client()

    if keys:
        return vault.get_secrets_batch(path, keys)
    else:
        return vault.get_secret(path)
