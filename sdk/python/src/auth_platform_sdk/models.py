"""Pydantic models for Auth Platform SDK - December 2025 State of Art.

Uses Pydantic v2 with strict validation, frozen models for immutability,
and modern Python 3.11+ type hints.
"""

from __future__ import annotations

from datetime import UTC, datetime, timedelta
from typing import Annotated, Any, Self

from pydantic import (
    BaseModel,
    ConfigDict,
    Field,
    HttpUrl,
    SecretStr,
    field_validator,
    model_validator,
)


class TokenResponse(BaseModel):
    """OAuth 2.0 token response from authorization server."""

    model_config = ConfigDict(frozen=True, extra="ignore")

    access_token: str = Field(..., min_length=1)
    token_type: str = Field(default="Bearer")
    expires_in: Annotated[int, Field(gt=0)]
    refresh_token: str | None = None
    scope: str | None = None
    id_token: str | None = None

    # DPoP support
    dpop_nonce: str | None = Field(default=None, alias="DPoP-Nonce")


class TokenData(BaseModel):
    """Internal token storage with expiration tracking."""

    model_config = ConfigDict(frozen=True)

    access_token: str
    token_type: str
    expires_at: datetime
    refresh_token: str | None = None
    scope: str | None = None
    id_token: str | None = None
    dpop_nonce: str | None = None

    @classmethod
    def from_response(
        cls,
        response: TokenResponse,
        *,
        buffer_seconds: int = 60,
    ) -> Self:
        """Create TokenData from TokenResponse with expiration calculation."""
        expires_at = datetime.now(UTC) + timedelta(
            seconds=response.expires_in - buffer_seconds
        )
        return cls(
            access_token=response.access_token,
            token_type=response.token_type,
            expires_at=expires_at,
            refresh_token=response.refresh_token,
            scope=response.scope,
            id_token=response.id_token,
            dpop_nonce=response.dpop_nonce,
        )

    def is_expired(self) -> bool:
        """Check if token is expired."""
        return datetime.now(UTC) >= self.expires_at

    def time_until_expiry(self) -> timedelta:
        """Get time remaining until token expires."""
        return self.expires_at - datetime.now(UTC)


class TokenClaims(BaseModel):
    """JWT token claims with validation."""

    model_config = ConfigDict(frozen=True, extra="allow")

    sub: str = Field(..., description="Subject identifier")
    iss: str = Field(..., description="Issuer")
    aud: str | list[str] = Field(..., description="Audience")
    exp: int = Field(..., description="Expiration time (Unix timestamp)")
    iat: int = Field(..., description="Issued at time (Unix timestamp)")
    nbf: int | None = Field(default=None, description="Not before time")
    jti: str | None = Field(default=None, description="JWT ID")
    scope: str | None = None
    client_id: str | None = None

    # DPoP confirmation
    cnf: dict[str, Any] | None = Field(default=None, description="Confirmation claim")

    @property
    def scopes(self) -> list[str]:
        """Get scopes as list."""
        if self.scope is None:
            return []
        return self.scope.split()

    @property
    def is_expired(self) -> bool:
        """Check if token claims indicate expiration."""
        return datetime.now(UTC).timestamp() >= self.exp

    @property
    def expires_at(self) -> datetime:
        """Get expiration as datetime."""
        return datetime.fromtimestamp(self.exp, tz=UTC)

    @property
    def issued_at(self) -> datetime:
        """Get issued at as datetime."""
        return datetime.fromtimestamp(self.iat, tz=UTC)


class PKCEChallenge(BaseModel):
    """PKCE challenge data for authorization code flow."""

    model_config = ConfigDict(frozen=True)

    code_verifier: str = Field(..., min_length=43, max_length=128)
    code_challenge: str = Field(..., min_length=43)
    code_challenge_method: str = Field(default="S256")

    @field_validator("code_challenge_method")
    @classmethod
    def validate_method(cls, v: str) -> str:
        """Validate PKCE method is S256 (plain is insecure)."""
        if v != "S256":
            msg = "Only S256 code_challenge_method is supported"
            raise ValueError(msg)
        return v


class DPoPProof(BaseModel):
    """DPoP proof token data."""

    model_config = ConfigDict(frozen=True)

    proof: str = Field(..., description="DPoP proof JWT")
    thumbprint: str = Field(..., description="JWK thumbprint")
    nonce: str | None = Field(default=None, description="Server-provided nonce")


class AuthorizationRequest(BaseModel):
    """OAuth 2.0 authorization request parameters."""

    model_config = ConfigDict(frozen=True)

    response_type: str = Field(default="code")
    client_id: str
    redirect_uri: HttpUrl
    scope: str | None = None
    state: str | None = None
    nonce: str | None = None

    # PKCE
    code_challenge: str | None = None
    code_challenge_method: str | None = None

    # OpenID Connect
    prompt: str | None = None
    login_hint: str | None = None
    acr_values: str | None = None

    def to_query_params(self) -> dict[str, str]:
        """Convert to URL query parameters."""
        params: dict[str, str] = {
            "response_type": self.response_type,
            "client_id": self.client_id,
            "redirect_uri": str(self.redirect_uri),
        }
        if self.scope:
            params["scope"] = self.scope
        if self.state:
            params["state"] = self.state
        if self.nonce:
            params["nonce"] = self.nonce
        if self.code_challenge:
            params["code_challenge"] = self.code_challenge
        if self.code_challenge_method:
            params["code_challenge_method"] = self.code_challenge_method
        if self.prompt:
            params["prompt"] = self.prompt
        if self.login_hint:
            params["login_hint"] = self.login_hint
        if self.acr_values:
            params["acr_values"] = self.acr_values
        return params


class TokenRequest(BaseModel):
    """OAuth 2.0 token request parameters."""

    model_config = ConfigDict(frozen=True)

    grant_type: str
    client_id: str
    client_secret: SecretStr | None = None
    code: str | None = None
    redirect_uri: str | None = None
    refresh_token: str | None = None
    scope: str | None = None
    code_verifier: str | None = None

    @model_validator(mode="after")
    def validate_grant_requirements(self) -> Self:
        """Validate required fields based on grant type."""
        if self.grant_type == "authorization_code":
            if not self.code:
                msg = "code is required for authorization_code grant"
                raise ValueError(msg)
            if not self.redirect_uri:
                msg = "redirect_uri is required for authorization_code grant"
                raise ValueError(msg)
        elif self.grant_type == "refresh_token":
            if not self.refresh_token:
                msg = "refresh_token is required for refresh_token grant"
                raise ValueError(msg)
        return self

    def to_form_data(self) -> dict[str, str]:
        """Convert to form data for token request."""
        data: dict[str, str] = {
            "grant_type": self.grant_type,
            "client_id": self.client_id,
        }
        if self.client_secret:
            data["client_secret"] = self.client_secret.get_secret_value()
        if self.code:
            data["code"] = self.code
        if self.redirect_uri:
            data["redirect_uri"] = self.redirect_uri
        if self.refresh_token:
            data["refresh_token"] = self.refresh_token
        if self.scope:
            data["scope"] = self.scope
        if self.code_verifier:
            data["code_verifier"] = self.code_verifier
        return data


class JWK(BaseModel):
    """JSON Web Key representation."""

    model_config = ConfigDict(frozen=True, extra="allow")

    kty: str = Field(..., description="Key type")
    kid: str | None = Field(default=None, description="Key ID")
    use: str | None = Field(default=None, description="Key use")
    alg: str | None = Field(default=None, description="Algorithm")

    # RSA keys
    n: str | None = None
    e: str | None = None

    # EC keys
    crv: str | None = None
    x: str | None = None
    y: str | None = None


class JWKS(BaseModel):
    """JSON Web Key Set."""

    model_config = ConfigDict(frozen=True)

    keys: list[JWK]

    def get_key(self, kid: str) -> JWK | None:
        """Get key by ID."""
        for key in self.keys:
            if key.kid == kid:
                return key
        return None

    def get_signing_keys(self) -> list[JWK]:
        """Get all keys suitable for signature verification."""
        return [k for k in self.keys if k.use in (None, "sig")]
