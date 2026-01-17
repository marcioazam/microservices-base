"""HashiCorp Vault Client for Secret Management.

This module provides a secure client for retrieving secrets from HashiCorp Vault.
Supports caching, automatic token renewal, rate limiting, and error handling.
Includes Prometheus metrics for observability.
"""

import logging
import os
import threading
import time
from functools import lru_cache
from typing import Any, Dict, Optional

import hvac
from hvac.exceptions import InvalidPath, VaultError

# Import metrics (optional - gracefully handles missing prometheus_client)
try:
    from src.observability.vault_metrics import VaultMetrics
    METRICS_AVAILABLE = True
except ImportError:
    METRICS_AVAILABLE = False

logger = logging.getLogger(__name__)


class TokenBucket:
    """
    Token bucket rate limiter for controlling API request rates.

    The token bucket algorithm allows for burst traffic while maintaining
    a maximum average rate over time.
    """

    def __init__(self, rate: float, capacity: int):
        """
        Initialize token bucket.

        Args:
            rate: Tokens added per second
            capacity: Maximum number of tokens in the bucket
        """
        self.rate = rate
        self.capacity = capacity
        self._tokens = float(capacity)
        self._lock = threading.Lock()
        self._last_update = time.time()

    def consume(self, tokens: int = 1, blocking: bool = True, timeout: Optional[float] = None) -> bool:
        """
        Consume tokens from the bucket.

        Args:
            tokens: Number of tokens to consume
            blocking: If True, wait until tokens are available
            timeout: Maximum time to wait in seconds (None = wait forever)

        Returns:
            True if tokens were consumed, False if not available and non-blocking

        Raises:
            TimeoutError: If blocking and timeout expires before tokens are available
        """
        start_time = time.time()

        while True:
            with self._lock:
                now = time.time()
                elapsed = now - self._last_update

                # Add tokens based on elapsed time
                self._tokens = min(self.capacity, self._tokens + elapsed * self.rate)
                self._last_update = now

                # Try to consume tokens
                if self._tokens >= tokens:
                    self._tokens -= tokens
                    return True

                # Calculate wait time for required tokens
                tokens_needed = tokens - self._tokens
                wait_time = tokens_needed / self.rate

            # Non-blocking mode
            if not blocking:
                return False

            # Check timeout
            if timeout is not None:
                elapsed_total = time.time() - start_time
                if elapsed_total >= timeout:
                    raise TimeoutError(f"Rate limit timeout after {timeout}s waiting for {tokens} tokens")
                wait_time = min(wait_time, timeout - elapsed_total)

            # Wait for tokens to be available
            time.sleep(min(wait_time, 0.1))  # Sleep in small increments


class VaultClient:
    """HashiCorp Vault client for secure secret management."""

    def __init__(
        self,
        vault_addr: Optional[str] = None,
        vault_token: Optional[str] = None,
        vault_namespace: Optional[str] = None,
        mount_point: str = "secret",
        auto_renew: bool = True,
        renew_threshold: float = 0.3,
        renew_interval: int = 300,
        rate_limit: Optional[float] = 10.0,
        rate_limit_burst: Optional[int] = 20,
    ):
        """
        Initialize Vault client.

        Args:
            vault_addr: Vault server URL (default: from VAULT_ADDR env var)
            vault_token: Vault authentication token (default: from VAULT_TOKEN env var)
            vault_namespace: Vault namespace (default: from VAULT_NAMESPACE env var)
            mount_point: KV secrets engine mount point (default: "secret")
            auto_renew: Enable automatic token renewal (default: True)
            renew_threshold: Renew when TTL drops below this fraction (default: 0.3 = 30%)
            renew_interval: Check interval in seconds (default: 300 = 5 minutes)
            rate_limit: Maximum requests per second (None = no limit, default: 10.0)
            rate_limit_burst: Maximum burst size (default: 20)
        """
        self.vault_addr = vault_addr or os.getenv("VAULT_ADDR", "http://localhost:8200")
        self.vault_token = vault_token or os.getenv("VAULT_TOKEN", "")
        self.vault_namespace = vault_namespace or os.getenv("VAULT_NAMESPACE", "")
        self.mount_point = mount_point
        self.auto_renew = auto_renew
        self.renew_threshold = renew_threshold
        self.renew_interval = renew_interval

        # Rate limiting
        self._rate_limiter: Optional[TokenBucket] = None
        if rate_limit is not None and rate_limit > 0:
            self._rate_limiter = TokenBucket(rate=rate_limit, capacity=rate_limit_burst or int(rate_limit * 2))

        # Token renewal thread management
        self._renewal_thread: Optional[threading.Thread] = None
        self._stop_renewal = threading.Event()
        self._initial_ttl: Optional[int] = None

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

        # Get initial TTL for renewal threshold calculation
        self._initial_ttl = self.get_token_ttl()

        logger.info(
            "Vault client initialized successfully",
            extra={
                "vault_addr": self.vault_addr,
                "namespace": self.vault_namespace or "default",
                "authenticated": True,
                "auto_renew": self.auto_renew,
                "initial_ttl": self._initial_ttl,
            },
        )

        # Initialize metrics
        if METRICS_AVAILABLE:
            VaultMetrics.initialize()
            VaultMetrics.set_client_info(
                vault_addr=self.vault_addr,
                namespace=self.vault_namespace,
                auto_renew=self.auto_renew,
                rate_limit=rate_limit,
            )
            if self._initial_ttl:
                VaultMetrics.record_token_ttl(self._initial_ttl)

        # Start automatic token renewal if enabled
        if self.auto_renew:
            self._start_token_renewal()

    def _apply_rate_limit(self, operation: str = "vault_operation") -> None:
        """
        Apply rate limiting before Vault operation.

        Args:
            operation: Name of the operation for logging

        Raises:
            TimeoutError: If rate limit timeout expires
        """
        if self._rate_limiter:
            start_wait = time.time()
            try:
                self._rate_limiter.consume(tokens=1, blocking=True, timeout=10.0)
            except TimeoutError as e:
                logger.warning(
                    "Rate limit timeout for Vault operation",
                    extra={
                        "operation": operation,
                        "vault_addr": self.vault_addr,
                    },
                )
                raise TimeoutError(f"Rate limit exceeded for {operation}: {e}") from e
            finally:
                wait_time = time.time() - start_wait
                if wait_time > 0.001 and METRICS_AVAILABLE:  # Only record if waited
                    VaultMetrics.record_rate_limit_wait(wait_time)

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
            TimeoutError: If rate limit timeout expires

        Example:
            >>> vault = VaultClient()
            >>> jwt_secret = vault.get_secret("sms-service", "jwt_secret_key")
            >>> all_secrets = vault.get_secret("sms-service")
        """
        # Apply rate limiting
        self._apply_rate_limit(operation="get_secret")

        start_time = time.time()
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
                result = secret_data[key]
            else:
                result = secret_data

            # Record success metrics
            if METRICS_AVAILABLE:
                VaultMetrics.record_success("get_secret", time.time() - start_time)
                VaultMetrics.record_cache_miss()  # Vault read = cache miss

            return result

        except InvalidPath:
            logger.error(
                "Secret path not found in Vault",
                extra={
                    "path": path,
                    "key": key,
                    "mount_point": self.mount_point,
                    "vault_addr": self.vault_addr,
                },
            )
            if METRICS_AVAILABLE:
                VaultMetrics.record_error("get_secret", "InvalidPath")
            raise InvalidPath(f"Secret path '{path}' not found in Vault") from None
        except KeyError as e:
            logger.error(
                "Secret key not found in Vault path",
                extra={
                    "path": path,
                    "key": key,
                    "vault_addr": self.vault_addr,
                },
            )
            if METRICS_AVAILABLE:
                VaultMetrics.record_error("get_secret", "KeyError")
            raise
        except VaultError as e:
            logger.error(
                "Vault error retrieving secret",
                extra={
                    "path": path,
                    "key": key,
                    "mount_point": self.mount_point,
                    "vault_addr": self.vault_addr,
                    "error": str(e),
                },
            )
            if METRICS_AVAILABLE:
                VaultMetrics.record_error("get_secret", "VaultError")
            raise VaultError(f"Failed to retrieve secret from '{path}': {e}") from e

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

        Raises:
            VaultError: If Vault operation fails
            TimeoutError: If rate limit timeout expires

        Example:
            >>> vault = VaultClient()
            >>> vault.put_secret("sms-service", {
            ...     "jwt_secret_key": "new_secret",
            ...     "api_key": "new_key"
            ... })
        """
        # Apply rate limiting
        self._apply_rate_limit(operation="put_secret")

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

        Raises:
            VaultError: If Vault operation fails
            TimeoutError: If rate limit timeout expires

        Example:
            >>> vault = VaultClient()
            >>> vault.delete_secret("sms-service/temp")
        """
        # Apply rate limiting
        self._apply_rate_limit(operation="delete_secret")

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
        With auto_renew enabled, this is handled automatically.

        Raises:
            VaultError: If token renewal fails
            TimeoutError: If rate limit timeout expires
        """
        # Apply rate limiting
        self._apply_rate_limit(operation="renew_token")

        start_time = time.time()
        try:
            self.client.auth.token.renew_self()
            logger.info("Vault token renewed successfully")

            # Record success metrics
            if METRICS_AVAILABLE:
                VaultMetrics.record_success("renew_token", time.time() - start_time)
                VaultMetrics.record_token_renewal(success=True)
                # Update token TTL
                new_ttl = self.get_token_ttl()
                VaultMetrics.record_token_ttl(new_ttl)

        except VaultError as e:
            logger.error(f"Failed to renew token: {e}")
            if METRICS_AVAILABLE:
                VaultMetrics.record_error("renew_token", "VaultError")
                VaultMetrics.record_token_renewal(success=False)
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

    def _start_token_renewal(self) -> None:
        """Start the background token renewal thread."""
        if self._renewal_thread is not None and self._renewal_thread.is_alive():
            logger.warning("Token renewal thread already running")
            return

        self._stop_renewal.clear()
        self._renewal_thread = threading.Thread(
            target=self._token_renewal_loop,
            name="vault-token-renewal",
            daemon=True,
        )
        self._renewal_thread.start()
        logger.info(
            "Token renewal thread started",
            extra={
                "renew_interval": self.renew_interval,
                "renew_threshold": self.renew_threshold,
            },
        )

    def _token_renewal_loop(self) -> None:
        """
        Background thread that periodically checks and renews the Vault token.

        This thread runs continuously, checking the token TTL at regular intervals.
        When the TTL drops below the renewal threshold, it automatically renews the token.
        """
        while not self._stop_renewal.is_set():
            try:
                # Check current TTL
                current_ttl = self.get_token_ttl()

                if current_ttl == 0:
                    logger.error(
                        "Token TTL is 0 or lookup failed, stopping renewal thread",
                        extra={"vault_addr": self.vault_addr},
                    )
                    break

                # Calculate renewal threshold in seconds
                if self._initial_ttl:
                    threshold_seconds = self._initial_ttl * self.renew_threshold
                else:
                    # Fallback: renew when less than 1 hour remains
                    threshold_seconds = 3600

                # Renew if below threshold
                if current_ttl < threshold_seconds:
                    logger.info(
                        "Token TTL below threshold, renewing token",
                        extra={
                            "current_ttl": current_ttl,
                            "threshold_seconds": threshold_seconds,
                            "vault_addr": self.vault_addr,
                        },
                    )

                    try:
                        self.renew_token()
                        new_ttl = self.get_token_ttl()

                        logger.info(
                            "Token renewed successfully",
                            extra={
                                "old_ttl": current_ttl,
                                "new_ttl": new_ttl,
                                "vault_addr": self.vault_addr,
                            },
                        )
                    except VaultError as e:
                        logger.error(
                            "Failed to renew token in background thread",
                            extra={
                                "error": str(e),
                                "current_ttl": current_ttl,
                                "vault_addr": self.vault_addr,
                            },
                        )
                else:
                    logger.debug(
                        "Token TTL above threshold, no renewal needed",
                        extra={
                            "current_ttl": current_ttl,
                            "threshold_seconds": threshold_seconds,
                        },
                    )

            except Exception as e:
                logger.error(
                    "Unexpected error in token renewal loop",
                    extra={"error": str(e), "vault_addr": self.vault_addr},
                )

            # Wait for the configured interval or until stop signal
            self._stop_renewal.wait(timeout=self.renew_interval)

        logger.info("Token renewal thread stopped")

    def stop_token_renewal(self) -> None:
        """
        Stop the automatic token renewal thread.

        This method signals the renewal thread to stop and waits for it to finish.
        Safe to call multiple times.
        """
        if self._renewal_thread is None or not self._renewal_thread.is_alive():
            logger.debug("Token renewal thread not running")
            return

        logger.info("Stopping token renewal thread...")
        self._stop_renewal.set()

        # Wait for thread to finish (with timeout)
        self._renewal_thread.join(timeout=5)

        if self._renewal_thread.is_alive():
            logger.warning("Token renewal thread did not stop within timeout")
        else:
            logger.info("Token renewal thread stopped successfully")

    def __del__(self) -> None:
        """Cleanup: stop token renewal thread when object is destroyed."""
        try:
            self.stop_token_renewal()
        except Exception:
            # Suppress exceptions during cleanup
            pass


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
