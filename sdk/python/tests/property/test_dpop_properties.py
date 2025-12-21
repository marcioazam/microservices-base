"""
Property-based tests for DPoP module.

**Feature: python-sdk-modernization, Property 5: DPoP Proof Structure**
**Feature: python-sdk-modernization, Property 6: DPoP Thumbprint Determinism**
**Feature: python-sdk-modernization, Property 7: DPoP Algorithm Support**
**Validates: Requirements 7.1, 7.2, 7.3, 7.4, 7.5**
"""

import base64
import hashlib

import jwt
from hypothesis import given, settings, strategies as st

from auth_platform_sdk.dpop import DPoPKeyPair, verify_dpop_proof


# Strategy for HTTP methods
http_method_strategy = st.sampled_from(["GET", "POST", "PUT", "DELETE", "PATCH"])

# Strategy for valid URIs
uri_strategy = st.sampled_from([
    "https://auth.example.com/token",
    "https://api.example.com/resource",
    "https://identity.company.org/oauth/token",
    "https://auth.test.local:8443/api/v1/users",
])

# Strategy for supported algorithms
algorithm_strategy = st.sampled_from(["ES256", "ES384", "ES512"])

# Strategy for optional nonce
nonce_strategy = st.one_of(
    st.none(),
    st.text(min_size=8, max_size=64).filter(lambda x: x.isalnum()),
)

# Strategy for access tokens
access_token_strategy = st.one_of(
    st.none(),
    st.text(min_size=20, max_size=200).filter(lambda x: len(x.strip()) > 0),
)


class TestDPoPProofStructureProperties:
    """Property tests for DPoP proof structure."""

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
    )
    @settings(max_examples=100)
    def test_proof_has_correct_header_type(
        self,
        http_method: str,
        http_uri: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof header SHALL have typ: "dpop+jwt".
        """
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(http_method, http_uri)

        header = jwt.get_unverified_header(proof.proof)
        assert header["typ"] == "dpop+jwt"

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
        algorithm=algorithm_strategy,
    )
    @settings(max_examples=100)
    def test_proof_has_correct_algorithm(
        self,
        http_method: str,
        http_uri: str,
        algorithm: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof header SHALL have correct alg claim.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)
        proof = key_pair.create_proof(http_method, http_uri)

        header = jwt.get_unverified_header(proof.proof)
        assert header["alg"] == algorithm

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
    )
    @settings(max_examples=100)
    def test_proof_has_jwk_in_header(
        self,
        http_method: str,
        http_uri: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof header SHALL contain jwk claim.
        """
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(http_method, http_uri)

        header = jwt.get_unverified_header(proof.proof)
        assert "jwk" in header
        assert header["jwk"]["kty"] == "EC"

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
    )
    @settings(max_examples=100)
    def test_proof_has_required_payload_claims(
        self,
        http_method: str,
        http_uri: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof payload SHALL have jti, htm, htu, and iat claims.
        """
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(http_method, http_uri)

        payload = jwt.decode(proof.proof, options={"verify_signature": False})

        assert "jti" in payload
        assert payload["htm"] == http_method.upper()
        assert payload["htu"] == http_uri
        assert "iat" in payload
        assert isinstance(payload["iat"], int)

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
        access_token=st.text(min_size=20, max_size=100),
    )
    @settings(max_examples=100)
    def test_proof_includes_ath_when_token_provided(
        self,
        http_method: str,
        http_uri: str,
        access_token: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof SHALL include ath claim when access_token is provided.
        """
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(
            http_method,
            http_uri,
            access_token=access_token,
        )

        payload = jwt.decode(proof.proof, options={"verify_signature": False})

        assert "ath" in payload

        # Verify ath is correct SHA-256 hash
        expected_ath = hashlib.sha256(access_token.encode()).digest()
        expected_ath_b64 = base64.urlsafe_b64encode(expected_ath).decode().rstrip("=")
        assert payload["ath"] == expected_ath_b64

    @given(
        http_method=http_method_strategy,
        http_uri=uri_strategy,
        nonce=st.text(min_size=8, max_size=64).filter(lambda x: x.isalnum()),
    )
    @settings(max_examples=100)
    def test_proof_includes_nonce_when_provided(
        self,
        http_method: str,
        http_uri: str,
        nonce: str,
    ) -> None:
        """
        Property 5: DPoP Proof Structure
        Proof SHALL include nonce claim when nonce is provided.
        """
        key_pair = DPoPKeyPair()
        proof = key_pair.create_proof(
            http_method,
            http_uri,
            nonce=nonce,
        )

        payload = jwt.decode(proof.proof, options={"verify_signature": False})

        assert payload["nonce"] == nonce
        assert proof.nonce == nonce


class TestDPoPThumbprintDeterminismProperties:
    """Property tests for DPoP thumbprint determinism."""

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_thumbprint_is_deterministic(self, algorithm: str) -> None:
        """
        Property 6: DPoP Thumbprint Determinism
        For any DPoPKeyPair, multiple calls to thumbprint property
        SHALL return the same value.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)

        thumbprint1 = key_pair.thumbprint
        thumbprint2 = key_pair.thumbprint
        thumbprint3 = key_pair.thumbprint

        assert thumbprint1 == thumbprint2 == thumbprint3

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_proof_thumbprint_matches_keypair(self, algorithm: str) -> None:
        """
        Property 6: DPoP Thumbprint Determinism
        Proof thumbprint SHALL match key pair thumbprint.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        assert proof.thumbprint == key_pair.thumbprint

    @given(
        algorithm=algorithm_strategy,
        http_method=http_method_strategy,
        http_uri=uri_strategy,
    )
    @settings(max_examples=100)
    def test_multiple_proofs_same_thumbprint(
        self,
        algorithm: str,
        http_method: str,
        http_uri: str,
    ) -> None:
        """
        Property 6: DPoP Thumbprint Determinism
        Multiple proofs from same key pair SHALL have same thumbprint.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)

        proof1 = key_pair.create_proof(http_method, http_uri)
        proof2 = key_pair.create_proof(http_method, http_uri)
        proof3 = key_pair.create_proof("GET", "https://other.example.com/api")

        assert proof1.thumbprint == proof2.thumbprint == proof3.thumbprint


class TestDPoPAlgorithmSupportProperties:
    """Property tests for DPoP algorithm support."""

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_supported_algorithms_create_valid_keypairs(
        self,
        algorithm: str,
    ) -> None:
        """
        Property 7: DPoP Algorithm Support
        For any algorithm in {ES256, ES384, ES512}, DPoPKeyPair creation
        SHALL succeed.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)

        assert key_pair.algorithm == algorithm
        assert key_pair.thumbprint is not None
        assert len(key_pair.thumbprint) > 0

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_supported_algorithms_produce_valid_proofs(
        self,
        algorithm: str,
    ) -> None:
        """
        Property 7: DPoP Algorithm Support
        For any supported algorithm, generated proofs SHALL be valid.
        """
        key_pair = DPoPKeyPair(algorithm=algorithm)
        proof = key_pair.create_proof("POST", "https://auth.example.com/token")

        # Proof should be verifiable
        payload = verify_dpop_proof(
            proof.proof,
            "POST",
            "https://auth.example.com/token",
        )

        assert payload["htm"] == "POST"
        assert payload["htu"] == "https://auth.example.com/token"

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_jwk_has_correct_curve_for_algorithm(
        self,
        algorithm: str,
    ) -> None:
        """
        Property 7: DPoP Algorithm Support
        JWK SHALL have correct curve for the algorithm.
        """
        expected_curves = {
            "ES256": "P-256",
            "ES384": "P-384",
            "ES512": "P-521",
        }

        key_pair = DPoPKeyPair(algorithm=algorithm)
        jwk = key_pair.jwk

        assert jwk["crv"] == expected_curves[algorithm]

    @given(algorithm=algorithm_strategy)
    @settings(max_examples=100)
    def test_exported_key_can_be_reimported(
        self,
        algorithm: str,
    ) -> None:
        """
        Property 7: DPoP Algorithm Support
        Exported keys SHALL be reimportable with same thumbprint.
        """
        original = DPoPKeyPair(algorithm=algorithm)
        pem = original.export_private_key()

        imported = DPoPKeyPair.from_private_key_pem(pem, algorithm=algorithm)

        assert imported.thumbprint == original.thumbprint
        assert imported.algorithm == algorithm
