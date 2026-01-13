"""Property tests for JWKSCacheBase - December 2025 State of Art.

Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
Validates: Requirements 3.1, 3.2, 3.3, 3.4
"""

from __future__ import annotations

import time
from unittest.mock import patch

import pytest
from hypothesis import given, settings, strategies as st, assume

from auth_platform_sdk.core.jwks_base import JWKSCacheBase
from auth_platform_sdk.models import JWK, JWKS


# Strategies for generating test data
ttl_seconds = st.integers(min_value=60, max_value=86400)  # 1 min to 24 hours
refresh_ahead_seconds = st.integers(min_value=0, max_value=3600)  # 0 to 1 hour
elapsed_times = st.floats(min_value=0, max_value=100000, allow_nan=False, allow_infinity=False)
jwks_uris = st.text(min_size=10, max_size=100).map(lambda s: f"https://example.com/{s}")


def create_test_jwks() -> JWKS:
    """Create a test JWKS for caching."""
    return JWKS(keys=[
        JWK(kty="EC", kid="key-1", use="sig", alg="ES256", crv="P-256", x="test", y="test"),
        JWK(kty="RSA", kid="key-2", use="sig", alg="RS256", n="test", e="AQAB"),
    ])


class TestJWKSCacheRefreshLogic:
    """Property tests for JWKS cache refresh logic."""

    @given(ttl=ttl_seconds, refresh_ahead=refresh_ahead_seconds)
    @settings(max_examples=100)
    def test_empty_cache_always_needs_refresh(self, ttl: int, refresh_ahead: int) -> None:
        """Property 10: Empty cache always returns should_refresh=True.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.1, 3.2
        """
        assume(refresh_ahead <= ttl)
        
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )
        
        assert cache.should_refresh() is True
        assert cache.is_expired() is True
        assert cache.is_cached is False

    @given(ttl=ttl_seconds, refresh_ahead=refresh_ahead_seconds)
    @settings(max_examples=100)
    def test_fresh_cache_does_not_need_refresh(self, ttl: int, refresh_ahead: int) -> None:
        """Property 10: Freshly populated cache does not need refresh.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.1, 3.2
        """
        assume(refresh_ahead < ttl)
        
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )
        cache.update_cache(create_test_jwks())
        
        assert cache.should_refresh() is False
        assert cache.is_expired() is False
        assert cache.is_cached is True

    @given(
        ttl=st.integers(min_value=100, max_value=10000),
        refresh_ahead=st.integers(min_value=10, max_value=1000),
    )
    @settings(max_examples=100)
    def test_refresh_triggers_at_threshold(self, ttl: int, refresh_ahead: int) -> None:
        """Property 10: Refresh triggers when elapsed > (TTL - refresh_ahead).
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.2
        """
        assume(refresh_ahead < ttl)
        
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )
        cache.update_cache(create_test_jwks())
        
        # Simulate time passing just past the threshold
        threshold = ttl - refresh_ahead
        with patch.object(time, "time", return_value=cache._cache_time + threshold + 1):
            assert cache.should_refresh() is True
            # But not yet expired
            assert cache.is_expired() is False

    @given(ttl=ttl_seconds)
    @settings(max_examples=100)
    def test_cache_expires_after_ttl(self, ttl: int) -> None:
        """Property 10: Cache expires after TTL seconds.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.2
        """
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=0,
        )
        cache.update_cache(create_test_jwks())
        
        # Simulate time passing past TTL
        with patch.object(time, "time", return_value=cache._cache_time + ttl + 1):
            assert cache.is_expired() is True
            assert cache.should_refresh() is True

    @given(ttl=ttl_seconds, refresh_ahead=refresh_ahead_seconds)
    @settings(max_examples=100)
    def test_invalidate_forces_refresh(self, ttl: int, refresh_ahead: int) -> None:
        """Property 10: Invalidate forces refresh on next access.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.4
        """
        assume(refresh_ahead <= ttl)
        
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )
        cache.update_cache(create_test_jwks())
        
        # Cache should be valid
        assert cache.is_cached is True
        
        # Invalidate
        cache.invalidate()
        
        # Now should need refresh
        assert cache.should_refresh() is True
        assert cache.is_cached is False
        assert cache.cached_jwks is None


class TestJWKSCacheKeyLookup:
    """Property tests for JWKS key lookup."""

    @given(kid=st.text(min_size=1, max_size=50, alphabet=st.characters(whitelist_categories=("L", "N"))))
    @settings(max_examples=100)
    def test_get_key_returns_none_for_empty_cache(self, kid: str) -> None:
        """Property 10: get_key returns None for empty cache.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.3
        """
        cache = JWKSCacheBase("https://example.com/.well-known/jwks.json")
        
        assert cache.get_key(kid) is None

    def test_get_key_returns_correct_key(self) -> None:
        """Property 10: get_key returns correct key by ID.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.3
        """
        cache = JWKSCacheBase("https://example.com/.well-known/jwks.json")
        cache.update_cache(create_test_jwks())
        
        key1 = cache.get_key("key-1")
        key2 = cache.get_key("key-2")
        
        assert key1 is not None
        assert key1.kid == "key-1"
        assert key1.kty == "EC"
        
        assert key2 is not None
        assert key2.kid == "key-2"
        assert key2.kty == "RSA"

    def test_get_signing_keys_returns_all_sig_keys(self) -> None:
        """Property 10: get_signing_keys returns all keys with use=sig.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.3
        """
        cache = JWKSCacheBase("https://example.com/.well-known/jwks.json")
        cache.update_cache(create_test_jwks())
        
        signing_keys = cache.get_signing_keys()
        
        assert len(signing_keys) == 2
        assert all(k.use in (None, "sig") for k in signing_keys)


class TestJWKSCacheTimingCalculations:
    """Property tests for timing calculations."""

    @given(
        ttl=st.integers(min_value=100, max_value=10000),
        refresh_ahead=st.integers(min_value=10, max_value=1000),
        elapsed=st.floats(min_value=0, max_value=5000, allow_nan=False, allow_infinity=False),
    )
    @settings(max_examples=100)
    def test_time_until_refresh_calculation(
        self,
        ttl: int,
        refresh_ahead: int,
        elapsed: float,
    ) -> None:
        """Property 10: time_until_refresh is correctly calculated.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.2
        """
        assume(refresh_ahead < ttl)
        
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
            refresh_ahead_seconds=refresh_ahead,
        )
        cache.update_cache(create_test_jwks())
        
        with patch.object(time, "time", return_value=cache._cache_time + elapsed):
            time_until = cache.time_until_refresh()
            threshold = ttl - refresh_ahead
            expected = max(0, threshold - elapsed)
            
            assert abs(time_until - expected) < 0.01  # Allow small float error

    @given(
        ttl=st.integers(min_value=100, max_value=10000),
        elapsed=st.floats(min_value=0, max_value=15000, allow_nan=False, allow_infinity=False),
    )
    @settings(max_examples=100)
    def test_time_until_expiry_calculation(self, ttl: int, elapsed: float) -> None:
        """Property 10: time_until_expiry is correctly calculated.
        
        Feature: python-sdk-state-of-art-2025, Property 10: JWKS Cache Refresh Logic
        Validates: Requirements 3.2
        """
        cache = JWKSCacheBase(
            "https://example.com/.well-known/jwks.json",
            ttl_seconds=ttl,
        )
        cache.update_cache(create_test_jwks())
        
        with patch.object(time, "time", return_value=cache._cache_time + elapsed):
            time_until = cache.time_until_expiry()
            expected = max(0, ttl - elapsed)
            
            assert abs(time_until - expected) < 0.01  # Allow small float error
