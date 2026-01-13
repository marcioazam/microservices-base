"""Unit tests for Pydantic models.

Tests model validation, serialization, and edge cases.
"""

from __future__ import annotations

import pytest
from pydantic import ValidationError

from auth_platform_sdk.models import (
    AuthorizationRequest,
    JWK,
    JWKS,
    PKCEChallenge,
    TokenClaims,
    TokenData,
    TokenRequest,
    TokenResponse,
)


class TestTokenResponse:
    """Tests for TokenResponse model."""

    def test_valid_token_response(self) -> None:
        """Test creating a valid token response."""
        response = TokenResponse(
            access_token="access123",
            token_type="Bearer",
            expires_in=3600,
        )
        assert response.access_token == "access123"
        assert response.token_type == "Bearer"
        assert response.expires_in == 3600
        assert response.refresh_token is None
        assert response.scope is None

    def test_token_response_with_refresh(self) -> None:
        """Test token response with refresh token."""
        response = TokenResponse(
            access_token="access123",
            token_type="Bearer",
            expires_in=3600,
            refresh_token="refresh456",
            scope="openid profile",
        )
        assert response.refresh_token == "refresh456"
        assert response.scope == "openid profile"


class TestTokenClaims:
    """Tests for TokenClaims model."""

    def test_valid_claims(self) -> None:
        """Test creating valid token claims."""
        claims = TokenClaims(
            sub="user123",
            iss="https://auth.example.com",
            aud="client123",
            exp=1700000000,
            iat=1699996400,
        )
        assert claims.sub == "user123"
        assert claims.iss == "https://auth.example.com"

    def test_claims_with_custom_fields(self) -> None:
        """Test claims with additional custom fields."""
        claims = TokenClaims(
            sub="user123",
            iss="https://auth.example.com",
            aud="client123",
            exp=1700000000,
            iat=1699996400,
            custom_claim="custom_value",
        )
        assert claims.sub == "user123"


class TestJWK:
    """Tests for JWK model."""

    def test_ec_jwk(self) -> None:
        """Test creating an EC JWK."""
        jwk = JWK(
            kty="EC",
            kid="key1",
            use="sig",
            alg="ES256",
            crv="P-256",
            x="base64x",
            y="base64y",
        )
        assert jwk.kty == "EC"
        assert jwk.kid == "key1"
        assert jwk.crv == "P-256"

    def test_rsa_jwk(self) -> None:
        """Test creating an RSA JWK."""
        jwk = JWK(
            kty="RSA",
            kid="key2",
            use="sig",
            alg="RS256",
            n="modulus",
            e="AQAB",
        )
        assert jwk.kty == "RSA"
        assert jwk.n == "modulus"


class TestJWKS:
    """Tests for JWKS model."""

    def test_jwks_with_keys(self) -> None:
        """Test creating a JWKS with multiple keys."""
        jwks = JWKS(
            keys=[
                JWK(kty="EC", kid="key1", use="sig", alg="ES256"),
                JWK(kty="RSA", kid="key2", use="sig", alg="RS256"),
            ]
        )
        assert len(jwks.keys) == 2

    def test_empty_jwks(self) -> None:
        """Test creating an empty JWKS."""
        jwks = JWKS(keys=[])
        assert len(jwks.keys) == 0


class TestPKCEChallenge:
    """Tests for PKCEChallenge model."""

    def test_valid_pkce_challenge(self) -> None:
        """Test creating a valid PKCE challenge."""
        # Use valid length verifier and challenge (min 43 chars)
        verifier = "a" * 43
        challenge = "b" * 43
        pkce = PKCEChallenge(
            code_verifier=verifier,
            code_challenge=challenge,
            code_challenge_method="S256",
        )
        assert pkce.code_verifier == verifier
        assert pkce.code_challenge_method == "S256"


class TestTokenRequest:
    """Tests for TokenRequest model."""

    def test_client_credentials_request(self) -> None:
        """Test creating a client credentials request."""
        request = TokenRequest(
            grant_type="client_credentials",
            client_id="client123",
            client_secret="secret456",
        )
        assert request.grant_type == "client_credentials"
        assert request.client_id == "client123"


class TestAuthorizationRequest:
    """Tests for AuthorizationRequest model."""

    def test_valid_authorization_request(self) -> None:
        """Test creating a valid authorization request."""
        request = AuthorizationRequest(
            response_type="code",
            client_id="client123",
            redirect_uri="https://app.example.com/callback",
            scope="openid profile",
            state="state123",
        )
        assert request.response_type == "code"
        assert str(request.redirect_uri) == "https://app.example.com/callback"
