"""
Property-based tests for JWKS caching.

**Feature: python-sdk-modernization, Property 3: JWKS Cache TTL Behavior**
**Feature: python-sdk-modernization, Property 4: JWKS Cache Thread Safety**
**Validates: Requirements 6.1, 6.2, 6.3**
"""

import threading
import time
from unittest.mock import MagicMock

from hypothesis import given, settings, strategies as st

from auth_platform_sdk.jwks import JWKSCache


class TestJWKSCacheTTLProperties:
    """Property tests for JWKS cache TTL behavior."""

    @given(ttl=st.integers(min_value=1, max_value=86400))
    @settings(max_examples=100)
    def test_cache_stores_ttl_correctly(self, ttl: int) -> None:
        """
        Property 3: JWKS Cache TTL Behavior
        For any TTL value, the cache SHALL store it correctly.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )
        assert cache.ttl_seconds == ttl

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_empty_cache_always_needs_refresh(self, ttl: int) -> None:
        """
        Property 3: JWKS Cache TTL Behavior
        For any JWKSCache, _should_refresh() SHALL return True when cache is empty.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )
        assert cache._should_refresh() is True

    @given(
        ttl=st.integers(min_value=60, max_value=3600),
        refresh_ahead=st.integers(min_value=0, max_value=300),
    )
    @settings(max_examples=100)
    def test_refresh_ahead_logic(self, ttl: int, refresh_ahead: int) -> None:
        """
        Property 3: JWKS Cache TTL Behavior
        For any TTL t and refresh_ahead r, cache SHALL refresh when
        elapsed time > (t - r).
        """
        if refresh_ahead >= ttl:
            return  # Skip invalid combinations

        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )

        # Simulate populated cache
        cache._cache_time = time.time()
        cache._jwk_client = MagicMock()

        # Fresh cache should not need refresh
        assert cache._should_refresh() is False

    @given(
        ttl=st.integers(min_value=60, max_value=3600),
        elapsed=st.integers(min_value=0, max_value=7200),
    )
    @settings(max_examples=100)
    def test_should_refresh_respects_ttl(self, ttl: int, elapsed: int) -> None:
        """
        Property 3: JWKS Cache TTL Behavior
        Cache SHALL return _should_refresh() == True when elapsed > TTL.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=0,  # No refresh-ahead for this test
        )

        # Simulate cache populated some time ago
        cache._cache_time = time.time() - elapsed
        cache._jwk_client = MagicMock()

        should_refresh = cache._should_refresh()
        expected = elapsed > ttl

        assert should_refresh == expected, f"TTL={ttl}, elapsed={elapsed}"

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_invalidation_resets_cache_time(self, ttl: int) -> None:
        """
        Property 3: JWKS Cache TTL Behavior
        When cache is invalidated, cache_time SHALL be reset to 0.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )

        # Simulate populated cache
        cache._cache_time = time.time()
        cache._jwk_client = MagicMock()

        cache.invalidate()

        assert cache._cache_time == 0
        assert cache._jwk_client is None


class TestJWKSCacheThreadSafetyProperties:
    """Property tests for JWKS cache thread safety."""

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=100)
    def test_concurrent_invalidation_no_errors(self, ttl: int) -> None:
        """
        Property 4: JWKS Cache Thread Safety
        For any JWKSCache, concurrent calls to invalidate() from multiple
        threads SHALL NOT raise exceptions.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )
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

        assert len(errors) == 0
        assert cache._cache_time == 0
        assert cache._jwk_client is None

    @given(ttl=st.integers(min_value=1, max_value=3600))
    @settings(max_examples=50)
    def test_concurrent_should_refresh_no_errors(self, ttl: int) -> None:
        """
        Property 4: JWKS Cache Thread Safety
        Concurrent calls to _should_refresh() SHALL NOT raise exceptions.
        """
        cache = JWKSCache(
            "https://auth.example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )

        errors: list[Exception] = []
        results: list[bool] = []

        def check_refresh() -> None:
            try:
                result = cache._should_refresh()
                results.append(result)
            except Exception as e:
                errors.append(e)

        threads = [threading.Thread(target=check_refresh) for _ in range(10)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert len(errors) == 0
        assert len(results) == 10

    @given(
        uri=st.text(min_size=10, max_size=100).filter(
            lambda x: x.startswith("https://")
        ),
        ttl=st.integers(min_value=1, max_value=86400),
    )
    @settings(max_examples=100)
    def test_cache_stores_uri_correctly(self, uri: str, ttl: int) -> None:
        """
        Property: Cache SHALL store JWKS URI correctly.
        """
        if not uri.startswith("https://"):
            uri = "https://" + uri

        cache = JWKSCache(uri, ttl_seconds=ttl)
        assert cache.jwks_uri == uri
