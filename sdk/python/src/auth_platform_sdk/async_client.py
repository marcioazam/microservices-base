"""Async Auth Platform SDK client - December 2025 State of Art.

Provides async client with DPoP, PKCE, JWKS caching,
circuit breaker, and OpenTelemetry integration.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, Self

import jwt
from jwt import algorithms

from .config import AuthPlatformConfig
from .dpop import DPoPKeyPair
from .errors import (
    TokenExpiredError,
    TokenRefreshError,
    ValidationError,
)
from .http import (
    CircuitBreaker,
    create_async_http_client,
    async_request_with_retry,
)
from .jwks import AsyncJWKSCache
from .models import TokenClaims, TokenData, TokenResponse
from .pkce import create_pkce_challenge, generate_nonce, generate_state
from .telemetry import get_logger, trace_operation

if TYPE_CHECKING:
    from .models import PKCEChallenge


class AsyncAuthPlatformClient:
    """Asynchronous Auth Platform client with full OAuth 2.0 support."""

    def __init__(self, config: AuthPlatformConfig) -> None:
        """Initialize async client.

        Args:
            config: SDK configuration.
        """
        self.config = config
        self._http = create_async_http_client(config)
        self._jwks_cache = AsyncJWKSCache(
            config.jwks_uri or f"{config.base_url_str}/.well-known/jwks.json",
            ttl_seconds=config.cache.jwks_ttl,
            refresh_ahead_seconds=config.cache.jwks_refresh_ahead,
        )
        self._tokens: TokenData | None = None
        self._circuit_breaker = CircuitBreaker()
        self._dpop_key: DPoPKeyPair | None = None
        self._dpop_nonce: str | None = None
        self._logger = get_logger()

        if config.dpop.enabled:
            self._dpop_key = DPoPKeyPair(algorithm=config.dpop.algorithm)

    async def __aenter__(self) -> Self:
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()

    async def close(self) -> None:
        """Close the HTTP client."""
        await self._http.aclose()

    async def validate_token(self, token: str) -> TokenClaims:
        """Validate JWT and return claims.

        Args:
            token: JWT access token.

        Returns:
            Validated token claims.

        Raises:
            ValidationError: If token is invalid.
        """
        with trace_operation("validate_token"):
            try:
                # Get key ID from token header
                header = jwt.get_unverified_header(token)
                kid = header.get("kid")

                if not kid:
                    raise ValidationError("Token missing kid header")

                # Get key from cache
                jwk = await self._jwks_cache.get_signing_key(kid)

                # Build key from JWK
                if jwk.kty == "EC":
                    key = algorithms.ECAlgorithm.from_jwk(jwk.model_dump())
                elif jwk.kty == "RSA":
                    key = algorithms.RSAAlgorithm.from_jwk(jwk.model_dump())
                else:
                    raise ValidationError(f"Unsupported key type: {jwk.kty}")

                decoded = jwt.decode(
                    token,
                    key,
                    algorithms=["ES256", "ES384", "RS256", "RS384", "RS512"],
                    audience=self.config.client_id,
                )
                return TokenClaims(**decoded)

            except jwt.exceptions.ExpiredSignatureError as e:
                raise TokenExpiredError() from e
            except jwt.exceptions.InvalidTokenError as e:
                raise ValidationError(f"Invalid token: {e}") from e

    async def client_credentials(
        self, *, scopes: list[str] | None = None
    ) -> TokenResponse:
        """Obtain token using client credentials flow.

        Args:
            scopes: Optional scopes to request.

        Returns:
            Token response.

        Raises:
            ValueError: If client_secret not configured.
            NetworkError: On network failure.
        """
        if not self.config.client_secret:
            msg = "client_secret required for client credentials flow"
            raise ValueError(msg)

        with trace_operation("client_credentials"):
            scope = " ".join(scopes) if scopes else self.config.scope_string
            data: dict[str, Any] = {
                "grant_type": "client_credentials",
                "client_id": self.config.client_id,
                "client_secret": self.config.client_secret.get_secret_value(),
            }
            if scope:
                data["scope"] = scope

            response = await self._token_request(data)
            self._tokens = TokenData.from_response(
                response, buffer_seconds=self.config.cache.token_buffer
            )
            return response

    async def refresh_token(self, refresh_token: str | None = None) -> TokenResponse:
        """Refresh access token.

        Args:
            refresh_token: Refresh token (uses stored token if not provided).

        Returns:
            New token response.

        Raises:
            TokenRefreshError: If refresh fails.
        """
        token = refresh_token or (self._tokens.refresh_token if self._tokens else None)
        if not token:
            raise TokenRefreshError("No refresh token available")

        with trace_operation("refresh_token"):
            try:
                data: dict[str, Any] = {
                    "grant_type": "refresh_token",
                    "refresh_token": token,
                    "client_id": self.config.client_id,
                }
                if self.config.client_secret:
                    data["client_secret"] = self.config.client_secret.get_secret_value()

                response = await self._token_request(data)
                self._tokens = TokenData.from_response(
                    response, buffer_seconds=self.config.cache.token_buffer
                )
                return response
            except Exception as e:
                self._tokens = None
                raise TokenRefreshError(str(e)) from e

    async def get_access_token(self, *, auto_refresh: bool = True) -> str:
        """Get valid access token, refreshing if needed.

        Args:
            auto_refresh: Whether to auto-refresh expired tokens.

        Returns:
            Valid access token.

        Raises:
            TokenExpiredError: If token expired and can't refresh.
        """
        if self._tokens is None:
            raise TokenExpiredError("No tokens available")

        if self._tokens.is_expired():
            if auto_refresh and self._tokens.refresh_token:
                await self.refresh_token()
            else:
                raise TokenExpiredError()

        return self._tokens.access_token

    def create_authorization_url(
        self,
        redirect_uri: str,
        *,
        scopes: list[str] | None = None,
        state: str | None = None,
        nonce: str | None = None,
        use_pkce: bool = True,
    ) -> tuple[str, str | None, PKCEChallenge | None]:
        """Create authorization URL for authorization code flow.

        Args:
            redirect_uri: Redirect URI after authorization.
            scopes: Scopes to request.
            state: CSRF state (generated if not provided).
            nonce: OpenID Connect nonce (generated if not provided).
            use_pkce: Whether to use PKCE.

        Returns:
            Tuple of (authorization_url, state, pkce_challenge).
        """
        state = state or generate_state()
        nonce = nonce or generate_nonce()
        pkce = create_pkce_challenge() if use_pkce else None

        params: dict[str, str] = {
            "response_type": "code",
            "client_id": self.config.client_id,
            "redirect_uri": redirect_uri,
            "state": state,
            "nonce": nonce,
        }

        scope = " ".join(scopes) if scopes else self.config.scope_string
        if scope:
            params["scope"] = scope

        if pkce:
            params["code_challenge"] = pkce.code_challenge
            params["code_challenge_method"] = pkce.code_challenge_method

        query = "&".join(f"{k}={v}" for k, v in params.items())
        url = f"{self.config.authorization_endpoint}?{query}"

        return url, state, pkce

    async def exchange_code(
        self,
        code: str,
        redirect_uri: str,
        *,
        code_verifier: str | None = None,
    ) -> TokenResponse:
        """Exchange authorization code for tokens.

        Args:
            code: Authorization code.
            redirect_uri: Redirect URI used in authorization.
            code_verifier: PKCE code verifier.

        Returns:
            Token response.
        """
        with trace_operation("exchange_code"):
            data: dict[str, Any] = {
                "grant_type": "authorization_code",
                "code": code,
                "redirect_uri": redirect_uri,
                "client_id": self.config.client_id,
            }
            if self.config.client_secret:
                data["client_secret"] = self.config.client_secret.get_secret_value()
            if code_verifier:
                data["code_verifier"] = code_verifier

            response = await self._token_request(data)
            self._tokens = TokenData.from_response(
                response, buffer_seconds=self.config.cache.token_buffer
            )
            return response

    async def _token_request(self, data: dict[str, Any]) -> TokenResponse:
        """Make token request with optional DPoP."""
        headers: dict[str, str] = {"Content-Type": "application/x-www-form-urlencoded"}

        if self._dpop_key:
            proof = self._dpop_key.create_proof(
                "POST",
                self.config.token_endpoint or "",
                nonce=self._dpop_nonce,
            )
            headers["DPoP"] = proof.proof

        response = await async_request_with_retry(
            self._http,
            "POST",
            self.config.token_endpoint or "/oauth/token",
            self.config.retry,
            circuit_breaker=self._circuit_breaker,
            data=data,
            headers=headers,
        )

        # Handle DPoP nonce
        if "DPoP-Nonce" in response.headers:
            self._dpop_nonce = response.headers["DPoP-Nonce"]

        if response.status_code == 400:
            error_data = response.json()
            if error_data.get("error") == "use_dpop_nonce":
                self._dpop_nonce = response.headers.get("DPoP-Nonce")
                return await self._token_request(data)

        response.raise_for_status()
        return TokenResponse(**response.json())
