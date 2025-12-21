"""
Property-based tests for PKCE module.

**Feature: python-sdk-modernization, Property 8: PKCE Verifier Length**
**Feature: python-sdk-modernization, Property 9: PKCE Challenge Verification Round-Trip**
**Feature: python-sdk-modernization, Property 10: PKCE State/Nonce Uniqueness**
**Validates: Requirements 8.1, 8.2, 8.3, 8.4, 8.5**
"""

from hypothesis import given, settings, strategies as st

from auth_platform_sdk.pkce import (
    create_pkce_challenge,
    generate_code_challenge,
    generate_code_verifier,
    generate_nonce,
    generate_state,
    verify_code_challenge,
)


# URL-safe base64 alphabet
URL_SAFE_CHARS = set(
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
)


class TestPKCEVerifierLengthProperties:
    """Property tests for PKCE verifier length."""

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_verifier_has_exact_length(self, length: int) -> None:
        """
        Property 8: PKCE Verifier Length
        For any length in [43, 128], generate_code_verifier(length)
        SHALL return a string of exactly that length.
        """
        verifier = generate_code_verifier(length)
        assert len(verifier) == length

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_verifier_contains_only_url_safe_chars(self, length: int) -> None:
        """
        Property 8: PKCE Verifier Length
        Verifier SHALL contain only URL-safe characters.
        """
        verifier = generate_code_verifier(length)
        assert all(c in URL_SAFE_CHARS for c in verifier)

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_verifier_is_ascii(self, length: int) -> None:
        """
        Property 8: PKCE Verifier Length
        Verifier SHALL be ASCII-encodable.
        """
        verifier = generate_code_verifier(length)
        # Should not raise
        verifier.encode("ascii")

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_pkce_challenge_verifier_has_correct_length(self, length: int) -> None:
        """
        Property 8: PKCE Verifier Length
        PKCEChallenge.code_verifier SHALL have the requested length.
        """
        pkce = create_pkce_challenge(length)
        assert len(pkce.code_verifier) == length


class TestPKCEChallengeVerificationProperties:
    """Property tests for PKCE challenge verification round-trip."""

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_verification_round_trip(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        For any valid code verifier, verify_code_challenge(verifier,
        generate_code_challenge(verifier)) SHALL return True.
        """
        verifier = generate_code_verifier(length)
        challenge = generate_code_challenge(verifier)

        assert verify_code_challenge(verifier, challenge) is True

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_pkce_challenge_verifies(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        PKCEChallenge SHALL verify correctly.
        """
        pkce = create_pkce_challenge(length)

        assert verify_code_challenge(pkce.code_verifier, pkce.code_challenge) is True

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_challenge_is_s256(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        Challenge method SHALL be S256.
        """
        pkce = create_pkce_challenge(length)
        assert pkce.code_challenge_method == "S256"

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_challenge_has_correct_length(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        S256 challenge SHALL be 43 characters (256 bits / 6 bits per char).
        """
        verifier = generate_code_verifier(length)
        challenge = generate_code_challenge(verifier)

        assert len(challenge) == 43

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_challenge_is_url_safe(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        Challenge SHALL contain only URL-safe characters.
        """
        verifier = generate_code_verifier(length)
        challenge = generate_code_challenge(verifier)

        assert all(c in URL_SAFE_CHARS for c in challenge)

    @given(length=st.integers(min_value=43, max_value=128))
    @settings(max_examples=100)
    def test_wrong_verifier_fails_verification(self, length: int) -> None:
        """
        Property 9: PKCE Challenge Verification Round-Trip
        Wrong verifier SHALL fail verification.
        """
        pkce = create_pkce_challenge(length)
        wrong_verifier = generate_code_verifier(length)

        # Different verifier should fail (with very high probability)
        if wrong_verifier != pkce.code_verifier:
            assert verify_code_challenge(wrong_verifier, pkce.code_challenge) is False


class TestPKCEStateNonceUniquenessProperties:
    """Property tests for PKCE state/nonce uniqueness."""

    @given(n=st.integers(min_value=10, max_value=100))
    @settings(max_examples=50)
    def test_state_uniqueness(self, n: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        For any N calls to generate_state(), the resulting set
        SHALL have N unique values.
        """
        states = [generate_state() for _ in range(n)]
        unique_states = set(states)

        assert len(unique_states) == n

    @given(n=st.integers(min_value=10, max_value=100))
    @settings(max_examples=50)
    def test_nonce_uniqueness(self, n: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        For any N calls to generate_nonce(), the resulting set
        SHALL have N unique values.
        """
        nonces = [generate_nonce() for _ in range(n)]
        unique_nonces = set(nonces)

        assert len(unique_nonces) == n

    @given(n=st.integers(min_value=10, max_value=100))
    @settings(max_examples=50)
    def test_verifier_uniqueness(self, n: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        For any N calls to generate_code_verifier(), the resulting set
        SHALL have N unique values.
        """
        verifiers = [generate_code_verifier() for _ in range(n)]
        unique_verifiers = set(verifiers)

        assert len(unique_verifiers) == n

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=100)
    def test_state_is_url_safe(self, length: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        State SHALL contain only URL-safe characters.
        """
        state = generate_state(length)
        assert all(c in URL_SAFE_CHARS for c in state)

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=100)
    def test_nonce_is_url_safe(self, length: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        Nonce SHALL contain only URL-safe characters.
        """
        nonce = generate_nonce(length)
        assert all(c in URL_SAFE_CHARS for c in nonce)

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=100)
    def test_state_has_sufficient_entropy(self, length: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        State SHALL have sufficient length for security.
        """
        state = generate_state(length)
        # token_urlsafe produces ~1.3x the requested length
        assert len(state) >= length

    @given(length=st.integers(min_value=16, max_value=64))
    @settings(max_examples=100)
    def test_nonce_has_sufficient_entropy(self, length: int) -> None:
        """
        Property 10: PKCE State/Nonce Uniqueness
        Nonce SHALL have sufficient length for security.
        """
        nonce = generate_nonce(length)
        # token_urlsafe produces ~1.3x the requested length
        assert len(nonce) >= length
