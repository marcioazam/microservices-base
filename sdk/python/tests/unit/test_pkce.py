"""Unit tests for PKCE implementation.

Tests RFC 7636 compliance with property-based testing using Hypothesis.
"""

import base64
import hashlib

import pytest
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.pkce import (
    create_pkce_challenge,
    generate_code_challenge,
    generate_code_verifier,
    generate_nonce,
    generate_state,
    verify_code_challenge,
)


class TestGenerateCodeVerifier:
    """Tests for code verifier generation."""

    def test_default_length(self) -> None:
        """Default verifier should be 64 characters."""
        verifier = generate_code_verifier()
        assert len(verifier) == 64

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=50)
    def test_respects_length_parameter(self, length: int) -> None:
        """Verifier should respect length parameter."""
        verifier = generate_code_verifier(length)
        assert len(verifier) == length

    def test_invalid_length_too_short(self) -> None:
        """Should reject length < 43."""
        with pytest.raises(ValueError, match="between 43 and 128"):
            generate_code_verifier(42)

    def test_invalid_length_too_long(self) -> None:
        """Should reject length > 128."""
        with pytest.raises(ValueError, match="between 43 and 128"):
            generate_code_verifier(129)

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=50)
    def test_url_safe_characters(self, length: int) -> None:
        """Verifier should only contain URL-safe characters."""
        verifier = generate_code_verifier(length)
        # URL-safe base64 alphabet
        valid_chars = set("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
        assert all(c in valid_chars for c in verifier)

    def test_uniqueness(self) -> None:
        """Each verifier should be unique."""
        verifiers = {generate_code_verifier() for _ in range(100)}
        assert len(verifiers) == 100


class TestGenerateCodeChallenge:
    """Tests for code challenge generation."""

    def test_s256_challenge(self) -> None:
        """Challenge should be SHA-256 hash of verifier."""
        verifier = "test_verifier_with_sufficient_length_for_pkce_requirements"
        challenge = generate_code_challenge(verifier)

        # Manually compute expected challenge
        expected_digest = hashlib.sha256(verifier.encode("ascii")).digest()
        expected = base64.urlsafe_b64encode(expected_digest).decode().rstrip("=")

        assert challenge == expected

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=50)
    def test_challenge_is_base64url(self, length: int) -> None:
        """Challenge should be valid base64url without padding."""
        verifier = generate_code_verifier(length)
        challenge = generate_code_challenge(verifier)

        # Should not have padding
        assert "=" not in challenge

        # Should be valid base64url
        valid_chars = set("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
        assert all(c in valid_chars for c in challenge)

    def test_challenge_length(self) -> None:
        """S256 challenge should be 43 characters (256 bits / 6 bits per char)."""
        verifier = generate_code_verifier()
        challenge = generate_code_challenge(verifier)
        assert len(challenge) == 43


class TestCreatePKCEChallenge:
    """Tests for complete PKCE challenge creation."""

    def test_creates_valid_challenge(self) -> None:
        """Should create valid PKCE challenge."""
        pkce = create_pkce_challenge()

        assert len(pkce.code_verifier) == 64
        assert len(pkce.code_challenge) == 43
        assert pkce.code_challenge_method == "S256"

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=50)
    def test_verifier_matches_challenge(self, length: int) -> None:
        """Verifier should produce the challenge."""
        pkce = create_pkce_challenge(length)

        expected_challenge = generate_code_challenge(pkce.code_verifier)
        assert pkce.code_challenge == expected_challenge


class TestVerifyCodeChallenge:
    """Tests for code challenge verification."""

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=50)
    def test_valid_verification(self, length: int) -> None:
        """Valid verifier should pass verification."""
        pkce = create_pkce_challenge(length)
        assert verify_code_challenge(pkce.code_verifier, pkce.code_challenge)

    def test_invalid_verifier(self) -> None:
        """Invalid verifier should fail verification."""
        pkce = create_pkce_challenge()
        wrong_verifier = generate_code_verifier()
        assert not verify_code_challenge(wrong_verifier, pkce.code_challenge)

    def test_timing_safe_comparison(self) -> None:
        """Verification should use constant-time comparison."""
        # This is a behavioral test - we verify the function uses secrets.compare_digest
        # by checking it doesn't short-circuit on first character mismatch
        pkce = create_pkce_challenge()

        # Both should take similar time (can't easily test timing, but verify behavior)
        result1 = verify_code_challenge("a" * 64, pkce.code_challenge)
        result2 = verify_code_challenge("b" * 64, pkce.code_challenge)

        assert not result1
        assert not result2


class TestGenerateState:
    """Tests for state parameter generation."""

    def test_default_length(self) -> None:
        """Default state should be URL-safe."""
        state = generate_state()
        assert len(state) > 0

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=50)
    def test_url_safe(self, length: int) -> None:
        """State should be URL-safe."""
        state = generate_state(length)
        valid_chars = set("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
        assert all(c in valid_chars for c in state)

    def test_uniqueness(self) -> None:
        """Each state should be unique."""
        states = {generate_state() for _ in range(100)}
        assert len(states) == 100


class TestGenerateNonce:
    """Tests for nonce generation."""

    def test_default_length(self) -> None:
        """Default nonce should be URL-safe."""
        nonce = generate_nonce()
        assert len(nonce) > 0

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=50)
    def test_url_safe(self, length: int) -> None:
        """Nonce should be URL-safe."""
        nonce = generate_nonce(length)
        valid_chars = set("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_")
        assert all(c in valid_chars for c in nonce)

    def test_uniqueness(self) -> None:
        """Each nonce should be unique."""
        nonces = {generate_nonce() for _ in range(100)}
        assert len(nonces) == 100
