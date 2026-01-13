"""Property-based tests for TokenValidator - December 2025 State of Art.

Property 12: Token Validation Consistency
- Valid tokens always produce valid claims
- Expired tokens always raise TokenExpiredError
- Invalid signatures always raise ValidationError
- Key type selection is deterministic
"""

from __future__ import annotations

import time
from datetime import UTC, datetime, timedelta
from typing import Any

import jwt
import pytest
from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ec, rsa
from hypothesis import given, settings, assume
from hypothesis import strategies as st

from auth_platform_sdk.config import AuthPlatformConfig
from auth_platform_sdk.core.token_validator import TokenValidator, SUPPORTED_ALGORITHMS
from auth_platform_sdk.errors import TokenExpiredError, ValidationError
from auth_platform_sdk.models import JWK


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
    
    # Convert to base64url
    import base64
    
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
    
    # Convert to base64url
    import base64
    
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


class TestTokenValidationConsistency:
    """Property tests for token validation consistency."""

    @given(
        subject=subject_strategy,
        issuer=issuer_strategy,
        scopes=scope_strategy,
    )
    @settings(max_examples=100)
    def test_valid_ec_token_produces_valid_claims(
        self,
        subject: str,
        issuer: str,
        scopes: list[str],
    ) -> None:
        """Property: Valid EC-signed tokens always produce valid claims."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        private_key, jwk_dict = generate_ec_key_pair()
        jwk = JWK(**jwk_dict)
        
        # Create valid token
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": issuer,
            "aud": config.client_id,
            "exp": int((now + timedelta(hours=1)).timestamp()),
            "iat": int(now.timestamp()),
        }
        if scopes:
            claims["scope"] = " ".join(scopes)
        
        token = jwt.encode(claims, private_key, algorithm="ES256", headers={"kid": jwk.kid})
        
        # Validate token
        result = validator.validate(token, jwk, issuer=issuer)
        
        assert result.sub == subject
        assert result.iss == issuer
        assert result.aud == config.client_id
        if scopes:
            assert set(result.scopes) == set(scopes)

    @given(
        subject=subject_strategy,
        issuer=issuer_strategy,
    )
    @settings(max_examples=100, deadline=None)
    def test_valid_rsa_token_produces_valid_claims(
        self,
        subject: str,
        issuer: str,
    ) -> None:
        """Property: Valid RSA-signed tokens always produce valid claims."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        private_key, jwk_dict = generate_rsa_key_pair()
        jwk = JWK(**jwk_dict)
        
        # Create valid token
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": issuer,
            "aud": config.client_id,
            "exp": int((now + timedelta(hours=1)).timestamp()),
            "iat": int(now.timestamp()),
        }
        
        token = jwt.encode(claims, private_key, algorithm="RS256", headers={"kid": jwk.kid})
        
        # Validate token
        result = validator.validate(token, jwk, issuer=issuer)
        
        assert result.sub == subject
        assert result.iss == issuer

    @given(
        subject=subject_strategy,
        issuer=issuer_strategy,
        seconds_expired=st.integers(min_value=1, max_value=3600),
    )
    @settings(max_examples=100)
    def test_expired_token_raises_error(
        self,
        subject: str,
        issuer: str,
        seconds_expired: int,
    ) -> None:
        """Property: Expired tokens always raise TokenExpiredError."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        private_key, jwk_dict = generate_ec_key_pair()
        jwk = JWK(**jwk_dict)
        
        # Create expired token
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": issuer,
            "aud": config.client_id,
            "exp": int((now - timedelta(seconds=seconds_expired)).timestamp()),
            "iat": int((now - timedelta(hours=1)).timestamp()),
        }
        
        token = jwt.encode(claims, private_key, algorithm="ES256", headers={"kid": jwk.kid})
        
        with pytest.raises(TokenExpiredError):
            validator.validate(token, jwk, issuer=issuer)

    @given(
        subject=subject_strategy,
        issuer=issuer_strategy,
    )
    @settings(max_examples=100)
    def test_wrong_key_raises_validation_error(
        self,
        subject: str,
        issuer: str,
    ) -> None:
        """Property: Tokens signed with wrong key raise ValidationError."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        # Generate two different key pairs
        private_key1, _ = generate_ec_key_pair()
        _, jwk_dict2 = generate_ec_key_pair()
        jwk2 = JWK(**jwk_dict2)
        
        # Create token with key1
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": issuer,
            "aud": config.client_id,
            "exp": int((now + timedelta(hours=1)).timestamp()),
            "iat": int(now.timestamp()),
        }
        
        token = jwt.encode(claims, private_key1, algorithm="ES256")
        
        # Validate with key2 - should fail
        with pytest.raises(ValidationError):
            validator.validate(token, jwk2, issuer=issuer)

    @given(
        subject=subject_strategy,
        issuer=issuer_strategy,
        wrong_audience=st.text(min_size=5, max_size=32),
    )
    @settings(max_examples=100)
    def test_wrong_audience_raises_validation_error(
        self,
        subject: str,
        issuer: str,
        wrong_audience: str,
    ) -> None:
        """Property: Tokens with wrong audience raise ValidationError."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        assume(wrong_audience != config.client_id)
        
        private_key, jwk_dict = generate_ec_key_pair()
        jwk = JWK(**jwk_dict)
        
        # Create token with wrong audience
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": issuer,
            "aud": wrong_audience,
            "exp": int((now + timedelta(hours=1)).timestamp()),
            "iat": int(now.timestamp()),
        }
        
        token = jwt.encode(claims, private_key, algorithm="ES256", headers={"kid": jwk.kid})
        
        with pytest.raises(ValidationError, match="Invalid audience"):
            validator.validate(token, jwk, issuer=issuer)


class TestKeyTypeSelection:
    """Property tests for key type selection."""

    def test_ec_key_type_selection(self) -> None:
        """Property: EC keys are correctly identified and used."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        _, jwk_dict = generate_ec_key_pair()
        jwk = JWK(**jwk_dict)
        
        key = validator._build_public_key(jwk)
        
        # Key should be usable for EC verification
        assert key is not None

    def test_rsa_key_type_selection(self) -> None:
        """Property: RSA keys are correctly identified and used."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        _, jwk_dict = generate_rsa_key_pair()
        jwk = JWK(**jwk_dict)
        
        key = validator._build_public_key(jwk)
        
        # Key should be usable for RSA verification
        assert key is not None

    def test_unsupported_key_type_raises_error(self) -> None:
        """Property: Unsupported key types raise ValidationError."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        jwk = JWK(kty="OKP", kid="test-key")  # OKP not in our supported list
        
        with pytest.raises(ValidationError, match="Unsupported key type"):
            validator._build_public_key(jwk)


class TestHeaderExtraction:
    """Property tests for token header extraction."""

    @given(
        subject=subject_strategy,
        kid=st.text(min_size=1, max_size=32, alphabet="abcdefghijklmnopqrstuvwxyz0123456789-_"),
    )
    @settings(max_examples=100)
    def test_key_id_extraction(
        self,
        subject: str,
        kid: str,
    ) -> None:
        """Property: Key ID is correctly extracted from token header."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        private_key, _ = generate_ec_key_pair()
        
        now = datetime.now(UTC)
        claims = {
            "sub": subject,
            "iss": "https://auth.example.com",
            "aud": config.client_id,
            "exp": int((now + timedelta(hours=1)).timestamp()),
            "iat": int(now.timestamp()),
        }
        
        token = jwt.encode(claims, private_key, algorithm="ES256", headers={"kid": kid})
        
        extracted_kid = validator.get_key_id(token)
        
        assert extracted_kid == kid

    def test_invalid_token_format_raises_error(self) -> None:
        """Property: Invalid token format raises ValidationError."""
        config = create_test_config()
        validator = TokenValidator(config)
        
        with pytest.raises(ValidationError, match="Invalid token format"):
            validator.get_unverified_header("not.a.valid.token")


class TestDPoPBinding:
    """Property tests for DPoP binding verification."""

    @given(
        thumbprint=st.text(min_size=32, max_size=64, alphabet="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"),
    )
    @settings(max_examples=100)
    def test_valid_dpop_binding(self, thumbprint: str) -> None:
        """Property: Valid DPoP binding is correctly verified."""
        from auth_platform_sdk.models import TokenClaims
        
        config = create_test_config()
        validator = TokenValidator(config)
        
        # Create claims with DPoP binding
        claims = TokenClaims(
            sub="user123",
            iss="https://auth.example.com",
            aud="test-client",
            exp=int((datetime.now(UTC) + timedelta(hours=1)).timestamp()),
            iat=int(datetime.now(UTC).timestamp()),
            cnf={"jkt": thumbprint},
        )
        
        result = validator.verify_dpop_binding(claims, thumbprint)
        
        assert result is True

    @given(
        thumbprint1=st.text(min_size=32, max_size=64, alphabet="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"),
        thumbprint2=st.text(min_size=32, max_size=64, alphabet="abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"),
    )
    @settings(max_examples=100)
    def test_invalid_dpop_binding(self, thumbprint1: str, thumbprint2: str) -> None:
        """Property: Mismatched DPoP binding is correctly rejected."""
        from auth_platform_sdk.models import TokenClaims
        
        assume(thumbprint1 != thumbprint2)
        
        config = create_test_config()
        validator = TokenValidator(config)
        
        # Create claims with different thumbprint
        claims = TokenClaims(
            sub="user123",
            iss="https://auth.example.com",
            aud="test-client",
            exp=int((datetime.now(UTC) + timedelta(hours=1)).timestamp()),
            iat=int(datetime.now(UTC).timestamp()),
            cnf={"jkt": thumbprint1},
        )
        
        result = validator.verify_dpop_binding(claims, thumbprint2)
        
        assert result is False

    def test_missing_cnf_claim(self) -> None:
        """Property: Missing cnf claim returns False."""
        from auth_platform_sdk.models import TokenClaims
        
        config = create_test_config()
        validator = TokenValidator(config)
        
        claims = TokenClaims(
            sub="user123",
            iss="https://auth.example.com",
            aud="test-client",
            exp=int((datetime.now(UTC) + timedelta(hours=1)).timestamp()),
            iat=int(datetime.now(UTC).timestamp()),
        )
        
        result = validator.verify_dpop_binding(claims, "some-thumbprint")
        
        assert result is False
