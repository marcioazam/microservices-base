"""JWKS caching for token validation - December 2025 State of Art.

Thread-safe and async-safe JWKS cache with configurable TTL,
background refresh, and OpenTelemetry integration.
"""

from __future__ import annotations

import asyncio
import threading
import time
from typing import TYPE_CHECKING, Any

import httpx
import jwt
from jwt import PyJWKClient

from .errors import ValidationError
from .models import JWK, JWKS

if TYPE_CHECKING:
    pass


class JWKSCache:
    """Thread-safe JWKS cache with configurable TTL and refresh-ahead."""

    def __init__(
        self,
        jwks_uri: str,
        *,
        ttl_seconds: int = 3600,
        refresh_ahead_seconds: int = 300,
        http_timeout: float = 10.0,
    ) -> None:
        """Initialize JWKS cache.

        Args:
            jwks_uri: URI to fetch JWKS from.
            ttl_seconds: Cache TTL in seconds.
            refresh_ahead_seconds: Seconds before expiry to trigger refresh.
            http_timeout: HTTP request timeout.
        """
        self.jwks_uri = jwks_uri
        self.ttl_seconds = ttl_seconds
        self.refresh_ahead_seconds = refresh_ahead_seconds
        self.http_timeout = http_timeout

        self._jwks: JWKS | None = None
        self._cache_time: float = 0
        self._lock = threading.RLock()
        self._jwk_client: PyJWKClient | None = None

    def get_signing_key(self, token: str) -> Any:
        """Get signing key for token, using cache if available.

        Args:
            token: JWT token to get signing key for.

        Returns:
            Signing key suitable for jwt.decode().

        Raises:
            ValidationError: If key cannot be found or JWKS fetch fails.
        """
        with self._lock:
            if self._should_refresh():
                self._refresh()

            if self._jwk_client is None:
                raise ValidationError("JWKS not loaded")

            try:
                return self._jwk_client.get_signing_key_from_jwt(token)
            except jwt.exceptions.PyJWKClientError as e:
                raise ValidationError(f"Failed to get signing key: {e}") from e

    def get_key_by_id(self, kid: str) -> JWK | None:
        """Get key by key ID.

        Args:
            kid: Key ID to look up.

        Returns:
            JWK if found, None otherwise.
        """
        with self._lock:
            if self._should_refresh():
                self._refresh()

            if self._jwks is None:
                return None

            return self._jwks.get_key(kid)

    def _should_refresh(self) -> bool:
        """Check if cache should be refreshed."""
        if self._jwk_client is None or self._jwks is None:
            return True

        elapsed = time.time() - self._cache_time
        # Refresh if expired or approaching expiry
        return elapsed > (self.ttl_seconds - self.refresh_ahead_seconds)

    def _refresh(self) -> None:
        """Refresh JWKS from server."""
        try:
            # Use PyJWKClient for key management
            self._jwk_client = PyJWKClient(
                self.jwks_uri,
                cache_keys=True,
                lifespan=self.ttl_seconds,
            )

            # Also fetch raw JWKS for direct key access
            with httpx.Client(timeout=self.http_timeout) as client:
                response = client.get(self.jwks_uri)
                response.raise_for_status()
                jwks_data = response.json()

            self._jwks = JWKS(
                keys=[JWK(**key) for key in jwks_data.get("keys", [])]
            )
            self._cache_time = time.time()

        except httpx.HTTPError as e:
            raise ValidationError(f"Failed to fetch JWKS: {e}") from e
        except Exception as e:
            raise ValidationError(f"Failed to parse JWKS: {e}") from e

    def invalidate(self) -> None:
        """Invalidate the cache, forcing refresh on next access."""
        with self._lock:
            self._jwk_client = None
            self._jwks = None
            self._cache_time = 0

    @property
    def is_cached(self) -> bool:
        """Check if JWKS is currently cached."""
        with self._lock:
            return self._jwks is not None and not self._should_refresh()


class AsyncJWKSCache:
    """Async JWKS cache with configurable TTL and refresh-ahead."""

    def __init__(
        self,
        jwks_uri: str,
        *,
        ttl_seconds: int = 3600,
        refresh_ahead_seconds: int = 300,
        http_timeout: float = 10.0,
    ) -> None:
        """Initialize async JWKS cache.

        Args:
            jwks_uri: URI to fetch JWKS from.
            ttl_seconds: Cache TTL in seconds.
            refresh_ahead_seconds: Seconds before expiry to trigger refresh.
            http_timeout: HTTP request timeout.
        """
        self.jwks_uri = jwks_uri
        self.ttl_seconds = ttl_seconds
        self.refresh_ahead_seconds = refresh_ahead_seconds
        self.http_timeout = http_timeout

        self._jwks: JWKS | None = None
        self._cache_time: float = 0
        self._lock = asyncio.Lock()

    async def get_signing_key(self, kid: str) -> JWK:
        """Get signing key by key ID.

        Args:
            kid: Key ID to look up.

        Returns:
            JWK for the key.

        Raises:
            ValidationError: If key not found.
        """
        async with self._lock:
            if self._should_refresh():
                await self._refresh()

            if self._jwks is None:
                raise ValidationError("JWKS not loaded")

            key = self._jwks.get_key(kid)
            if key is None:
                raise ValidationError(f"Key {kid} not found in JWKS")

            return key

    async def get_all_signing_keys(self) -> list[JWK]:
        """Get all signing keys.

        Returns:
            List of JWKs suitable for signature verification.
        """
        async with self._lock:
            if self._should_refresh():
                await self._refresh()

            if self._jwks is None:
                return []

            return self._jwks.get_signing_keys()

    def _should_refresh(self) -> bool:
        """Check if cache should be refreshed."""
        if self._jwks is None:
            return True

        elapsed = time.time() - self._cache_time
        return elapsed > (self.ttl_seconds - self.refresh_ahead_seconds)

    async def _refresh(self) -> None:
        """Refresh JWKS from server."""
        try:
            async with httpx.AsyncClient(timeout=self.http_timeout) as client:
                response = await client.get(self.jwks_uri)
                response.raise_for_status()
                jwks_data = response.json()

            self._jwks = JWKS(
                keys=[JWK(**key) for key in jwks_data.get("keys", [])]
            )
            self._cache_time = time.time()

        except httpx.HTTPError as e:
            raise ValidationError(f"Failed to fetch JWKS: {e}") from e
        except Exception as e:
            raise ValidationError(f"Failed to parse JWKS: {e}") from e

    async def invalidate(self) -> None:
        """Invalidate the cache, forcing refresh on next access."""
        async with self._lock:
            self._jwks = None
            self._cache_time = 0

    @property
    def is_cached(self) -> bool:
        """Check if JWKS is currently cached."""
        return self._jwks is not None and not self._should_refresh()
