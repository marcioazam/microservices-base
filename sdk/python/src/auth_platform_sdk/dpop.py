"""DPoP (Demonstrating Proof of Possession) implementation - December 2025 State of Art.

Implements RFC 9449 for sender-constrained access tokens using
proof-of-possession with asymmetric keys.
"""

from __future__ import annotations

import base64
import hashlib
import json
import time
import uuid
from typing import TYPE_CHECKING, Any

import jwt
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec

from .errors import DPoPError, ErrorCode
from .models import DPoPProof

if TYPE_CHECKING:
    from cryptography.hazmat.primitives.asymmetric.ec import (
        EllipticCurvePrivateKey,
        EllipticCurvePublicKey,
    )


class DPoPKeyPair:
    """Manages DPoP key pair for proof generation."""

    def __init__(
        self,
        private_key: EllipticCurvePrivateKey | None = None,
        algorithm: str = "ES256",
    ) -> None:
        """Initialize DPoP key pair.

        Args:
            private_key: Optional existing private key. If None, generates new key.
            algorithm: JWT algorithm to use (ES256, ES384, ES512).
        """
        self.algorithm = algorithm
        self._curve = self._get_curve_for_algorithm(algorithm)

        if private_key is None:
            self._private_key = ec.generate_private_key(self._curve)
        else:
            self._private_key = private_key

        self._public_key = self._private_key.public_key()
        self._jwk = self._create_jwk()
        self._thumbprint = self._compute_thumbprint()

    @staticmethod
    def _get_curve_for_algorithm(algorithm: str) -> ec.EllipticCurve:
        """Get elliptic curve for algorithm."""
        curves: dict[str, ec.EllipticCurve] = {
            "ES256": ec.SECP256R1(),
            "ES384": ec.SECP384R1(),
            "ES512": ec.SECP521R1(),
        }
        if algorithm not in curves:
            msg = f"Unsupported algorithm: {algorithm}"
            raise ValueError(msg)
        return curves[algorithm]

    def _create_jwk(self) -> dict[str, Any]:
        """Create JWK representation of public key."""
        public_numbers = self._public_key.public_numbers()

        # Get curve name for JWK
        curve_names = {
            "secp256r1": "P-256",
            "secp384r1": "P-384",
            "secp521r1": "P-521",
        }
        curve_name = curve_names.get(self._curve.name, "P-256")

        # Calculate byte length for coordinates
        key_size = self._public_key.key_size
        coord_bytes = (key_size + 7) // 8

        # Encode coordinates as base64url
        x_bytes = public_numbers.x.to_bytes(coord_bytes, "big")
        y_bytes = public_numbers.y.to_bytes(coord_bytes, "big")

        return {
            "kty": "EC",
            "crv": curve_name,
            "x": base64.urlsafe_b64encode(x_bytes).decode().rstrip("="),
            "y": base64.urlsafe_b64encode(y_bytes).decode().rstrip("="),
        }

    def _compute_thumbprint(self) -> str:
        """Compute JWK thumbprint per RFC 7638."""
        # Canonical JSON with sorted keys
        canonical = json.dumps(
            {
                "crv": self._jwk["crv"],
                "kty": self._jwk["kty"],
                "x": self._jwk["x"],
                "y": self._jwk["y"],
            },
            separators=(",", ":"),
            sort_keys=True,
        )

        # SHA-256 hash, base64url encoded
        digest = hashlib.sha256(canonical.encode()).digest()
        return base64.urlsafe_b64encode(digest).decode().rstrip("=")

    @property
    def thumbprint(self) -> str:
        """Get JWK thumbprint."""
        return self._thumbprint

    @property
    def jwk(self) -> dict[str, Any]:
        """Get public JWK."""
        return self._jwk.copy()

    def create_proof(
        self,
        http_method: str,
        http_uri: str,
        *,
        access_token: str | None = None,
        nonce: str | None = None,
    ) -> DPoPProof:
        """Create DPoP proof JWT.

        Args:
            http_method: HTTP method (GET, POST, etc.).
            http_uri: Full HTTP URI of the request.
            access_token: Optional access token for token-bound proofs.
            nonce: Optional server-provided nonce.

        Returns:
            DPoPProof containing the proof JWT and thumbprint.
        """
        now = int(time.time())

        # DPoP proof header
        header = {
            "typ": "dpop+jwt",
            "alg": self.algorithm,
            "jwk": self._jwk,
        }

        # DPoP proof payload
        payload: dict[str, Any] = {
            "jti": str(uuid.uuid4()),
            "htm": http_method.upper(),
            "htu": http_uri,
            "iat": now,
        }

        # Add access token hash if provided (for resource requests)
        if access_token:
            ath = hashlib.sha256(access_token.encode()).digest()
            payload["ath"] = base64.urlsafe_b64encode(ath).decode().rstrip("=")

        # Add nonce if provided
        if nonce:
            payload["nonce"] = nonce

        # Get private key in PEM format for PyJWT
        private_pem = self._private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        )

        # Sign the proof
        proof = jwt.encode(payload, private_pem, algorithm=self.algorithm, headers=header)

        return DPoPProof(
            proof=proof,
            thumbprint=self._thumbprint,
            nonce=nonce,
        )

    def export_private_key(self) -> bytes:
        """Export private key in PEM format."""
        return self._private_key.private_bytes(
            encoding=serialization.Encoding.PEM,
            format=serialization.PrivateFormat.PKCS8,
            encryption_algorithm=serialization.NoEncryption(),
        )

    @classmethod
    def from_private_key_pem(cls, pem_data: bytes, algorithm: str = "ES256") -> "DPoPKeyPair":
        """Create DPoPKeyPair from PEM-encoded private key."""
        private_key = serialization.load_pem_private_key(pem_data, password=None)
        if not isinstance(private_key, ec.EllipticCurvePrivateKey):
            msg = "Key must be an EC private key"
            raise ValueError(msg)
        return cls(private_key=private_key, algorithm=algorithm)


def verify_dpop_proof(
    proof: str,
    http_method: str,
    http_uri: str,
    *,
    access_token: str | None = None,
    expected_nonce: str | None = None,
    max_age_seconds: int = 60,
) -> dict[str, Any]:
    """Verify a DPoP proof JWT.

    Args:
        proof: The DPoP proof JWT.
        http_method: Expected HTTP method.
        http_uri: Expected HTTP URI.
        access_token: Expected access token (for ath claim).
        expected_nonce: Expected nonce value.
        max_age_seconds: Maximum age of the proof in seconds.

    Returns:
        Decoded proof payload if valid.

    Raises:
        DPoPError: If proof is invalid.
    """
    try:
        # Decode without verification first to get header
        unverified_header = jwt.get_unverified_header(proof)

        # Validate header
        if unverified_header.get("typ") != "dpop+jwt":
            raise DPoPError("Invalid DPoP proof type", ErrorCode.DPOP_INVALID)

        jwk = unverified_header.get("jwk")
        if not jwk:
            raise DPoPError("Missing JWK in DPoP proof header", ErrorCode.DPOP_INVALID)

        # Build public key from JWK
        algorithm = unverified_header.get("alg", "ES256")
        public_key = _jwk_to_public_key(jwk)

        # Verify and decode
        payload = jwt.decode(
            proof,
            public_key,
            algorithms=[algorithm],
            options={"verify_aud": False},
        )

        # Validate claims
        now = int(time.time())

        if payload.get("htm", "").upper() != http_method.upper():
            raise DPoPError("HTTP method mismatch", ErrorCode.DPOP_INVALID)

        if payload.get("htu") != http_uri:
            raise DPoPError("HTTP URI mismatch", ErrorCode.DPOP_INVALID)

        iat = payload.get("iat", 0)
        if now - iat > max_age_seconds:
            raise DPoPError("DPoP proof expired", ErrorCode.DPOP_INVALID)

        if expected_nonce and payload.get("nonce") != expected_nonce:
            raise DPoPError(
                "Nonce mismatch",
                ErrorCode.DPOP_NONCE_REQUIRED,
                dpop_nonce=expected_nonce,
            )

        # Verify access token hash if provided
        if access_token:
            expected_ath = hashlib.sha256(access_token.encode()).digest()
            expected_ath_b64 = base64.urlsafe_b64encode(expected_ath).decode().rstrip("=")
            if payload.get("ath") != expected_ath_b64:
                raise DPoPError("Access token hash mismatch", ErrorCode.DPOP_INVALID)

        return payload

    except jwt.InvalidTokenError as e:
        raise DPoPError(f"Invalid DPoP proof: {e}", ErrorCode.DPOP_INVALID) from e


def _jwk_to_public_key(jwk: dict[str, Any]) -> EllipticCurvePublicKey:
    """Convert JWK to cryptography public key."""
    if jwk.get("kty") != "EC":
        msg = "Only EC keys are supported for DPoP"
        raise ValueError(msg)

    crv = jwk.get("crv", "P-256")
    curves: dict[str, ec.EllipticCurve] = {
        "P-256": ec.SECP256R1(),
        "P-384": ec.SECP384R1(),
        "P-521": ec.SECP521R1(),
    }

    if crv not in curves:
        msg = f"Unsupported curve: {crv}"
        raise ValueError(msg)

    # Decode coordinates
    x_bytes = base64.urlsafe_b64decode(jwk["x"] + "==")
    y_bytes = base64.urlsafe_b64decode(jwk["y"] + "==")

    x = int.from_bytes(x_bytes, "big")
    y = int.from_bytes(y_bytes, "big")

    public_numbers = ec.EllipticCurvePublicNumbers(x, y, curves[crv])
    return public_numbers.public_key()
