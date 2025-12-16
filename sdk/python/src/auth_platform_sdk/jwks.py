"""JWKS caching for token validation."""

import threading
import time
from typing import Any, Optional

import httpx
import jwt
from jwt import PyJWKClient

from .errors import ValidationError


class JWKSCache:
    """Thread-safe JWKS cache with configurable TTL."""

    def __init__(self, jwks_uri: str, ttl_seconds: int = 3600):
        self.jwks_uri = jwks_uri
        self.ttl_seconds = ttl_seconds
        self._cache: dict[str, Any] = {}
        self._cache_time: float = 0
        self._lock = threading.RLock()
        self._jwk_client: Optional[PyJWKClient] = None

    def get_signing_key(self, token: str) -> Any:
        """Get signing key for token, using cache if available."""
        with self._lock:
            if self._should_refresh():
                self._refresh()

            if self._jwk_client is None:
                raise ValidationError("JWKS not loaded")

            try:
                return self._jwk_client.get_signing_key_from_jwt(token)
            except jwt.exceptions.PyJWKClientError as e:
                raise ValidationError(f"Failed to get signing key: {e}")

    def _should_refresh(self) -> bool:
        """Check if cache should be refreshed."""
        if self._jwk_client is None:
            return True
        return time.time() - self._cache_time > self.ttl_seconds

    def _refresh(self) -> None:
        """Refresh JWKS from server."""
        self._jwk_client = PyJWKClient(self.jwks_uri, cache_keys=True)
        self._cache_time = time.time()

    def invalidate(self) -> None:
        """Invalidate the cache."""
        with self._lock:
            self._jwk_client = None
            self._cache_time = 0


class AsyncJWKSCache:
    """Async JWKS cache with configurable TTL."""

    def __init__(self, jwks_uri: str, ttl_seconds: int = 3600):
        self.jwks_uri = jwks_uri
        self.ttl_seconds = ttl_seconds
        self._keys: dict[str, Any] = {}
        self._cache_time: float = 0
        self._lock = threading.RLock()

    async def get_signing_key(self, kid: str) -> Any:
        """Get signing key by key ID."""
        with self._lock:
            if self._should_refresh():
                await self._refresh()

            if kid not in self._keys:
                raise ValidationError(f"Key {kid} not found in JWKS")

            return self._keys[kid]

    def _should_refresh(self) -> bool:
        """Check if cache should be refreshed."""
        if not self._keys:
            return True
        return time.time() - self._cache_time > self.ttl_seconds

    async def _refresh(self) -> None:
        """Refresh JWKS from server."""
        async with httpx.AsyncClient() as client:
            response = await client.get(self.jwks_uri)
            response.raise_for_status()
            jwks = response.json()

        self._keys = {}
        for key in jwks.get("keys", []):
            if "kid" in key:
                self._keys[key["kid"]] = key

        self._cache_time = time.time()

    def invalidate(self) -> None:
        """Invalidate the cache."""
        with self._lock:
            self._keys = {}
            self._cache_time = 0
