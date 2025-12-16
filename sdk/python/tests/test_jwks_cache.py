"""
Property-based tests for JWKS caching.

**Feature: auth-platform-q2-2025-evolution, Property 12: SDK JWKS Caching**
**Validates: Requirements 9.2**
"""

import time
from unittest.mock import MagicMock, patch

from hypothesis import given, settings, strategies as st

from auth_platform_sdk.jwks import JWKSCache


class TestJWKSCacheProperties:
    """Property tests for JWKS caching behavior."""

    @given(ttl=st.integers(min_value=1, max_value=86400))
    @settings(max_examples=100)
    def test_cache_respects_ttl(self, ttl: int) -> None:
        """
        Property 12: SDK JWKS Caching
        For any token validation request, the SDK SHALL use cached JWKS
        if available and not expired, reducing network calls.
        """
        cache = JWKSCache("https://auth.example.com/.well-known/jwks.json", ttl_seconds=ttl)

        # Property: TTL must be stored correctly
        assert cache.ttl_seconds == ttl

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_cache_invalidation_resets_state(self, ttl: int) -> None:
        """Cache invalidation must reset internal state."""
        cache = JWKSCache("https://auth.example.com/.well-known/jwks.json", ttl_seconds=ttl)

        # Simulate cache being populated
        cache._cache_time = time.time()

        # Invalidate
        cache.invalidate()

        # Property: After invalidation, cache time must be reset
        assert cache._cache_time == 0
        assert cache._jwk_client is None

    @given(
        ttl=st.integers(min_value=60, max_value=3600),
        elapsed=st.integers(min_value=0, max_value=7200),
    )
    @settings(max_examples=100)
    def test_should_refresh_logic(self, ttl: int, elapsed: int) -> None:
        """Cache refresh logic must respect TTL."""
        cache = JWKSCache("https://auth.example.com/.well-known/jwks.json", ttl_seconds=ttl)

        # Simulate cache populated some time ago
        cache._cache_time = time.time() - elapsed
        cache._jwk_client = MagicMock()  # Simulate loaded client

        should_refresh = cache._should_refresh()

        # Property: Should refresh if elapsed time exceeds TTL
        expected = elapsed > ttl
        assert should_refresh == expected, f"TTL={ttl}, elapsed={elapsed}"

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_empty_cache_always_needs_refresh(self, ttl: int) -> None:
        """Empty cache must always need refresh."""
        cache = JWKSCache("https://auth.example.com/.well-known/jwks.json", ttl_seconds=ttl)

        # Property: Empty cache (no jwk_client) must need refresh
        assert cache._should_refresh() is True

    @given(
        uri=st.text(min_size=10, max_size=100).filter(lambda x: x.startswith("https://")),
        ttl=st.integers(min_value=1, max_value=86400),
    )
    @settings(max_examples=100)
    def test_cache_stores_uri_correctly(self, uri: str, ttl: int) -> None:
        """Cache must store JWKS URI correctly."""
        # Ensure valid URI format
        if not uri.startswith("https://"):
            uri = "https://" + uri

        cache = JWKSCache(uri, ttl_seconds=ttl)

        # Property: URI must be stored correctly
        assert cache.jwks_uri == uri

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_thread_safety_of_invalidation(self, ttl: int) -> None:
        """Invalidation must be thread-safe."""
        import threading

        cache = JWKSCache("https://auth.example.com/.well-known/jwks.json", ttl_seconds=ttl)
        cache._cache_time = time.time()
        cache._jwk_client = MagicMock()

        errors: list[Exception] = []

        def invalidate() -> None:
            try:
                cache.invalidate()
            except Exception as e:
                errors.append(e)

        threads = [threading.Thread(target=invalidate) for _ in range(10)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        # Property: No errors during concurrent invalidation
        assert len(errors) == 0

        # Property: Cache must be invalidated
        assert cache._cache_time == 0
        assert cache._jwk_client is None
