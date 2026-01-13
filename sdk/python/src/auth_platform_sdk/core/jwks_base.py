"""Base JWKS cache with shared refresh logic - December 2025 State of Art.

Provides centralized TTL and refresh-ahead calculations shared
between sync and async JWKS cache implementations.
"""

from __future__ import annotations

import time
from typing import TYPE_CHECKING

if TYPE_CHECKING:
    from ..models import JWK, JWKS


class JWKSCacheBase:
    """Base JWKS cache with shared refresh logic.
    
    This class provides the core caching logic that is shared between
    synchronous and asynchronous JWKS cache implementations.
    
    Attributes:
        jwks_uri: URI to fetch JWKS from.
        ttl_seconds: Cache TTL in seconds.
        refresh_ahead_seconds: Seconds before expiry to trigger refresh.
    """

    def __init__(
        self,
        jwks_uri: str,
        *,
        ttl_seconds: int = 3600,
        refresh_ahead_seconds: int = 300,
    ) -> None:
        """Initialize JWKS cache base.
        
        Args:
            jwks_uri: URI to fetch JWKS from.
            ttl_seconds: Cache TTL in seconds (default: 1 hour).
            refresh_ahead_seconds: Seconds before expiry to trigger refresh (default: 5 min).
        """
        self.jwks_uri = jwks_uri
        self.ttl_seconds = ttl_seconds
        self.refresh_ahead_seconds = refresh_ahead_seconds
        self._jwks: JWKS | None = None
        self._cache_time: float = 0

    def should_refresh(self) -> bool:
        """Check if cache should be refreshed.
        
        Returns True if:
        - Cache is empty (never populated)
        - Cache has expired
        - Cache is approaching expiry (within refresh_ahead_seconds)
        
        Returns:
            True if cache should be refreshed.
        """
        if self._jwks is None:
            return True
        
        elapsed = time.time() - self._cache_time
        threshold = self.ttl_seconds - self.refresh_ahead_seconds
        return elapsed > threshold

    def is_expired(self) -> bool:
        """Check if cache is fully expired (past TTL).
        
        Returns:
            True if cache is expired.
        """
        if self._jwks is None:
            return True
        
        elapsed = time.time() - self._cache_time
        return elapsed > self.ttl_seconds

    def time_until_refresh(self) -> float:
        """Get time in seconds until refresh is needed.
        
        Returns:
            Seconds until refresh (0 if refresh needed now).
        """
        if self._jwks is None:
            return 0
        
        elapsed = time.time() - self._cache_time
        threshold = self.ttl_seconds - self.refresh_ahead_seconds
        remaining = threshold - elapsed
        return max(0, remaining)

    def time_until_expiry(self) -> float:
        """Get time in seconds until cache expires.
        
        Returns:
            Seconds until expiry (0 if expired).
        """
        if self._jwks is None:
            return 0
        
        elapsed = time.time() - self._cache_time
        remaining = self.ttl_seconds - elapsed
        return max(0, remaining)

    def get_key(self, kid: str) -> JWK | None:
        """Get key by ID from cached JWKS.
        
        Args:
            kid: Key ID to look up.
            
        Returns:
            JWK if found, None otherwise.
        """
        if self._jwks is None:
            return None
        return self._jwks.get_key(kid)

    def get_signing_keys(self) -> list[JWK]:
        """Get all signing keys from cached JWKS.
        
        Returns:
            List of JWKs suitable for signature verification.
        """
        if self._jwks is None:
            return []
        return self._jwks.get_signing_keys()

    def update_cache(self, jwks: JWKS) -> None:
        """Update cache with new JWKS.
        
        Args:
            jwks: New JWKS to cache.
        """
        self._jwks = jwks
        self._cache_time = time.time()

    def invalidate(self) -> None:
        """Invalidate cache, forcing refresh on next access."""
        self._jwks = None
        self._cache_time = 0

    @property
    def is_cached(self) -> bool:
        """Check if JWKS is currently cached and valid.
        
        Returns:
            True if cache is populated and not expired.
        """
        return self._jwks is not None and not self.is_expired()

    @property
    def cached_jwks(self) -> JWKS | None:
        """Get cached JWKS if available.
        
        Returns:
            Cached JWKS or None.
        """
        return self._jwks
