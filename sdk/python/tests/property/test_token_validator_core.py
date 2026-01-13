"""Property-based tests for TokenValidator core validation - December 2025 State of Art.

Property 12: Token Validation Consistency
- Valid tokens always produce valid claims
- Expired tokens always raise TokenExpiredError
- Invalid signatures always raise ValidationError
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from typing import Any

import jwt
import pytest
from cryptography.hazmat.primitives.asymmetric import ec, rsa
from hypothesis import given, settings, assume
from hypothesis import strategies as st

from auth_platform_sdk.config import AuthPlatformConfig
from auth_platform_sdk.core.token_validator import TokenValidator
from auth_platform_sdk.errors import TokenExpiredError, ValidationError
from auth_platform_sdk.models import JWK

from .token_validator_helpers import (
    create_test_config,
    generate_ec_key_pair,
    generate_rsa_key_pair,
    subject_strategy,
    issuer_strategy,
    scope_strategy,
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
