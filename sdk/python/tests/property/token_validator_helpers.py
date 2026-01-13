"""Shared helpers for TokenValidator property tests - December 2025 State of Art.

Provides test configuration, key generation, and Hypothesis strategies
shared across token validator test modules.
"""

from __future__ import annotations

import base64
from typing import Any

from cryptography.hazmat.primitives.asymmetric import ec, rsa
from hypothesis import strategies as st

from auth_platform_sdk.config import AuthPlatformConfig


def create_test_config(client_id: str = "test-client") -> AuthPlatformConfig:
    """Create a test configuration."""
    return AuthPlatformConfig(
        client_id=client_id,
        base_url="https://auth.example.com",
    )


def generate_ec_key_pair() -> tuple[ec.EllipticCurvePrivateKey, dict[str, Any]]:
    """Generate EC key pair and JWK."""
    private_key = ec.generate_private_key(ec.SECP256R1())
    public_key = private_key.public_key()
    
    # Get public key numbers
    public_numbers = public_key.public_numbers()
    
    def int_to_base64url(n: int, length: int) -> str:
        data = n.to_bytes(length, byteorder="big")
        return base64.urlsafe_b64encode(data).rstrip(b"=").decode("ascii")
    
    jwk_dict = {
        "kty": "EC",
        "crv": "P-256",
        "x": int_to_base64url(public_numbers.x, 32),
        "y": int_to_base64url(public_numbers.y, 32),
        "kid": "test-key-1",
        "use": "sig",
        "alg": "ES256",
    }
    
    return private_key, jwk_dict


def generate_rsa_key_pair() -> tuple[rsa.RSAPrivateKey, dict[str, Any]]:
    """Generate RSA key pair and JWK."""
    private_key = rsa.generate_private_key(
        public_exponent=65537,
        key_size=2048,
    )
    public_key = private_key.public_key()
    
    # Get public key numbers
    public_numbers = public_key.public_numbers()
    
    def int_to_base64url(n: int) -> str:
        byte_length = (n.bit_length() + 7) // 8
        data = n.to_bytes(byte_length, byteorder="big")
        return base64.urlsafe_b64encode(data).rstrip(b"=").decode("ascii")
    
    jwk_dict = {
        "kty": "RSA",
        "n": int_to_base64url(public_numbers.n),
        "e": int_to_base64url(public_numbers.e),
        "kid": "test-rsa-key-1",
        "use": "sig",
        "alg": "RS256",
    }
    
    return private_key, jwk_dict


# Strategies for generating test data
subject_strategy = st.text(
    alphabet=st.characters(whitelist_categories=("L", "N"), whitelist_characters="-_"),
    min_size=1,
    max_size=64,
).filter(lambda x: len(x.strip()) > 0)

issuer_strategy = st.sampled_from([
    "https://auth.example.com",
    "https://issuer.example.org",
    "https://identity.example.io",
])

scope_strategy = st.lists(
    st.sampled_from(["openid", "profile", "email", "read", "write"]),
    min_size=0,
    max_size=3,
    unique=True,
)
