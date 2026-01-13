"""Unit tests for DPoP implementation.

Tests RFC 9449 compliance with property-based testing using Hypothesis.
"""

import base64
import hashlib
import json
import time

import jwt
import pytest
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.dpop import (
    DPoPKeyPair,
    verify_dpop_proof,
    _jwk_to_public_key,
)
from auth_platform_sdk.errors import DPoPError


class TestDPoPKeyPair:
    """Tests for DPoP key pair management."""

    def test_generates_key_pair(self) -> None:
        """Should generate valid EC key pair."""
        key_pair = DPoPKeyPair()

        assert key_pair.algorithm == "ES256"
        assert key_pair.thumbprint is not None
        assert len(key_pair.thumbprint) > 0

    @given(algorithm=st.sampled_from(["ES256", "ES384", "ES512"]))
    @settings(max_examples=10)
    def test_supports_ec_algorithms(self, algorithm: str) -> None:
        """Should support ES256, ES384, ES512."""
        key_pair = DPoPKeyPair(algorithm=algorithm)
        assert key_pair.algorithm == algorithm

    def test_invalid_algorithm(self) -> None:
        """Should reject unsupported algorithms."""
        with pytest.raises(ValueError, match="Unsupported algorithm"):
            DPoPKeyPair(algorithm="RS256")

    def test_jwk_structure(self) -> None:
        """JWK should have correct structure."""
        key_pair = DPoPKeyPair()
        jwk = key_pair.jwk

        assert jwk["kty"] == "EC"
        assert jwk["crv"] == "P-256"
        assert "x" in jwk
        assert "y" in jwk
        # Should not contain private key
        assert "d" not in jwk

    def test_thumbprint_is_deterministic(self) -> None:
        """Same key should produce same thumbprint."""
        key_pair = DPoPKeyPair()
        thumbprint1 = key_pair.thumbprint
        thumbprint2 = key_pair.thumbprint

        assert thumbprint1 == thumbprint2

    def test_different_keys_different_thumbprints(self) -> None:
        """Different keys should have different thumbprints."""
        key_pair1 = DPoPKeyPair()
        key_pair2 = DPoPKeyPair()

        assert key_pair1.thumbprint != key_pair2.thumbprint


class TestDPoPProofCreation:
    """Tests for DPoP proof creation."""

    def test_creates_valid_proof(self) -> None:
        """Should create valid DPoP proof JWT."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        assert proof.proof is not None
        assert proof.thumbprint == key_pair.thumbprint

    def test_proof_header(self) -> None:
        """Proof header should have correct structure."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        header = jwt.get_unverified_header(proof.proof)

        assert header["typ"] == "dpop+jwt"
        assert header["alg"] == "ES256"
        assert "jwk" in header

    def test_proof_payload(self) -> None:
        """Proof payload should have required claims."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        payload = jwt.decode(proof.proof, options={"verify_signature": False})

        assert "jti" in payload
        assert payload["htm"] == "POST"
        assert payload["htu"] == "https://auth.example.com/token"
        assert "iat" in payload

    def test_proof_with_access_token(self) -> None:
        """Proof should include ath claim when access token provided."""
        key_pair = DPoPKeyPair()
        access_token = "test_access_token"
        proof = key_pair.create_proof(
            "GET",
            "https://api.example.com/resource",
            access_token=access_token,
        )

        payload = jwt.decode(proof.proof, options={"verify_signature": False})

        # Verify ath is SHA-256 hash of access token
        expected_ath = hashlib.sha256(access_token.encode()).digest()
        expected_ath_b64 = base64.urlsafe_b64encode(expected_ath).decode().rstrip("=")

        assert payload["ath"] == expected_ath_b64

    def test_proof_with_nonce(self) -> None:
        """Proof should include nonce when provided."""
        key_pair = DPoPKeyPair()
        nonce = "server_provided_nonce"
        proof = key_pair.create_proof(
            "POST",
            "https://auth.example.com/token",
            nonce=nonce,
        )

        payload = jwt.decode(proof.proof, options={"verify_signature": False})
        assert payload["nonce"] == nonce
        assert proof.nonce == nonce

    @given(method=st.sampled_from(["GET", "POST", "PUT", "DELETE", "PATCH"]))
    @settings(max_examples=10)
    def test_http_method_in_proof(self, method: str) -> None:
        """Proof should include correct HTTP method."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(method, "https://example.com/api")

        payload = jwt.decode(proof.proof, options={"verify_signature": False})
        assert payload["htm"] == method.upper()


class TestDPoPProofVerification:
    """Tests for DPoP proof verification."""

    def test_verifies_valid_proof(self) -> None:
        """Should verify valid proof."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        payload = verify_dpop_proof(
            proof.proof,
            "POST",
            "https://auth.example.com/token",
        )

        assert payload["htm"] == "POST"
        assert payload["htu"] == "https://auth.example.com/token"

    def test_rejects_wrong_method(self) -> None:
        """Should reject proof with wrong HTTP method."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        with pytest.raises(DPoPError, match="HTTP method mismatch"):
            verify_dpop_proof(
                proof.proof,
                "GET",  # Wrong method
                "https://auth.example.com/token",
            )

    def test_rejects_wrong_uri(self) -> None:
        """Should reject proof with wrong URI."""
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        with pytest.raises(DPoPError, match="HTTP URI mismatch"):
            verify_dpop_proof(
                proof.proof,
                "POST",
                "https://other.example.com/token",  # Wrong URI
            )

    def test_rejects_expired_proof(self) -> None:
        """Should reject expired proof."""
        import time
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        # Wait a moment so the proof ages
        time.sleep(0.1)

        with pytest.raises(DPoPError, match="expired"):
            verify_dpop_proof(
                proof.proof,
                "POST",
                "https://auth.example.com/token",
                max_age_seconds=0,  # Immediately expired after any time passes
            )

    def test_verifies_access_token_hash(self) -> None:
        """Should verify access token hash."""
        key_pair = DPoPKeyPair()
        access_token = "test_access_token"
        proof = key_pair.create_proof(
            "GET",
            "https://api.example.com/resource",
            access_token=access_token,
        )

        # Should succeed with correct token
        verify_dpop_proof(
            proof.proof,
            "GET",
            "https://api.example.com/resource",
            access_token=access_token,
        )

        # Should fail with wrong token
        with pytest.raises(DPoPError, match="Access token hash mismatch"):
            verify_dpop_proof(
                proof.proof,
                "GET",
                "https://api.example.com/resource",
                access_token="wrong_token",
            )


class TestKeyExportImport:
    """Tests for key export and import."""

    def test_export_private_key(self) -> None:
        """Should export private key in PEM format."""
        key_pair = DPoPKeyPair()
        pem = key_pair.export_private_key()

        assert pem.startswith(b"-----BEGIN PRIVATE KEY-----")
        assert pem.endswith(b"-----END PRIVATE KEY-----\n")

    def test_import_private_key(self) -> None:
        """Should import private key from PEM."""
        original = DPoPKeyPair()
        pem = original.export_private_key()

        imported = DPoPKeyPair.from_private_key_pem(pem)

        # Should have same thumbprint
        assert imported.thumbprint == original.thumbprint

    def test_imported_key_creates_valid_proofs(self) -> None:
        """Imported key should create valid proofs."""
        original = DPoPKeyPair()
        pem = original.export_private_key()
        imported = DPoPKeyPair.from_private_key_pem(pem)

        proof = imported.create_proof("POST", "https://auth.example.com/token")

        # Should be verifiable
        verify_dpop_proof(
            proof.proof,
            "POST",
            "https://auth.example.com/token",
        )
