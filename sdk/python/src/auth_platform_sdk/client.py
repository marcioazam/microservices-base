"""Auth Platform SDK clients (sync and async)."""

import time
from typing import Any, Optional

import httpx
import jwt

from .config import AuthPlatformConfig
from .errors import (
    NetworkError,
    RateLimitError,
    TokenExpiredError,
    TokenRefreshError,
    ValidationError,
)
from .jwks import AsyncJWKSCache, JWKSCache
from .types import TokenClaims, TokenData, TokenResponse


class AuthPlatformClient:
    """Synchronous Auth Platform client."""

    def __init__(self, config: AuthPlatformConfig):
        self.config = config
        self._http = httpx.Client(
            base_url=config.base_url,
            timeout=config.timeout,
        )
        self._jwks_cache = JWKSCache(
            f"{config.base_url}/.well-known/jwks.json",
            ttl_seconds=config.jwks_cache_ttl,
        )
        self._tokens: Optional[TokenData] = None

    def __enter__(self) -> "AuthPlatformClient":
        return self

    def __exit__(self, *args: Any) -> None:
        self.close()

    def close(self) -> None:
        """Close the HTTP client."""
        self._http.close()

    def validate_token(self, token: str) -> TokenClaims:
        """Validate JWT and return claims."""
        try:
            signing_key = self._jwks_cache.get_signing_key(token)
            decoded = jwt.decode(
                token,
                signing_key.key,
                algorithms=["ES256", "RS256"],
                audience=self.config.client_id,
            )
            return TokenClaims(
                sub=decoded["sub"],
                iss=decoded["iss"],
                aud=decoded["aud"],
                exp=decoded["exp"],
                iat=decoded["iat"],
                scope=decoded.get("scope"),
                client_id=decoded.get("client_id"),
            )
        except jwt.exceptions.InvalidTokenError as e:
            raise ValidationError(f"Invalid token: {e}")

    def client_credentials(self) -> TokenResponse:
        """Obtain token using client credentials flow."""
        if not self.config.client_secret:
            raise ValueError("client_secret required for client credentials flow")

        response = self._request_with_retry(
            "POST",
            "/oauth/token",
            data={
                "grant_type": "client_credentials",
                "client_id": self.config.client_id,
                "client_secret": self.config.client_secret,
                "scope": " ".join(self.config.scopes) if self.config.scopes else None,
            },
        )

        token_response = TokenResponse(
            access_token=response["access_token"],
            token_type=response["token_type"],
            expires_in=response["expires_in"],
            refresh_token=response.get("refresh_token"),
            scope=response.get("scope"),
        )

        self._tokens = TokenData.from_response(token_response)
        return token_response

    def get_access_token(self) -> str:
        """Get access token, refreshing if necessary."""
        if self._tokens is None:
            raise TokenExpiredError("No tokens available")

        if self._tokens.is_expired():
            if self._tokens.refresh_token:
                self._refresh_tokens()
            else:
                raise TokenExpiredError()

        return self._tokens.access_token

    def _refresh_tokens(self) -> None:
        """Refresh tokens using refresh token."""
        if self._tokens is None or self._tokens.refresh_token is None:
            raise TokenRefreshError("No refresh token available")

        try:
            response = self._request_with_retry(
                "POST",
                "/oauth/token",
                data={
                    "grant_type": "refresh_token",
                    "refresh_token": self._tokens.refresh_token,
                    "client_id": self.config.client_id,
                },
            )

            token_response = TokenResponse(
                access_token=response["access_token"],
                token_type=response["token_type"],
                expires_in=response["expires_in"],
                refresh_token=response.get("refresh_token", self._tokens.refresh_token),
                scope=response.get("scope"),
            )

            self._tokens = TokenData.from_response(token_response)
        except Exception as e:
            self._tokens = None
            raise TokenRefreshError(str(e))

    def _request_with_retry(
        self,
        method: str,
        path: str,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Make HTTP request with retry logic."""
        last_error: Optional[Exception] = None

        for attempt in range(self.config.max_retries):
            try:
                response = self._http.request(method, path, **kwargs)

                if response.status_code == 429:
                    retry_after = response.headers.get("Retry-After")
                    raise RateLimitError(
                        retry_after=int(retry_after) if retry_after else None
                    )

                response.raise_for_status()
                return response.json()

            except RateLimitError as e:
                delay = e.retry_after or (self.config.retry_delay * (2**attempt))
                time.sleep(delay)
                last_error = e

            except httpx.HTTPError as e:
                last_error = NetworkError(str(e))
                if attempt < self.config.max_retries - 1:
                    time.sleep(self.config.retry_delay * (2**attempt))

        raise last_error or NetworkError("Request failed")


class AsyncAuthPlatformClient:
    """Asynchronous Auth Platform client."""

    def __init__(self, config: AuthPlatformConfig):
        self.config = config
        self._http = httpx.AsyncClient(
            base_url=config.base_url,
            timeout=config.timeout,
        )
        self._jwks_cache = AsyncJWKSCache(
            f"{config.base_url}/.well-known/jwks.json",
            ttl_seconds=config.jwks_cache_ttl,
        )
        self._tokens: Optional[TokenData] = None

    async def __aenter__(self) -> "AsyncAuthPlatformClient":
        return self

    async def __aexit__(self, *args: Any) -> None:
        await self.close()

    async def close(self) -> None:
        """Close the HTTP client."""
        await self._http.aclose()

    async def validate_token(self, token: str) -> TokenClaims:
        """Validate JWT and return claims."""
        try:
            # Decode header to get kid
            unverified = jwt.decode(token, options={"verify_signature": False})
            header = jwt.get_unverified_header(token)
            kid = header.get("kid")

            if not kid:
                raise ValidationError("Token missing kid header")

            key_data = await self._jwks_cache.get_signing_key(kid)

            # Build key from JWKS data
            from jwt import algorithms

            if key_data.get("kty") == "EC":
                key = algorithms.ECAlgorithm.from_jwk(key_data)
            elif key_data.get("kty") == "RSA":
                key = algorithms.RSAAlgorithm.from_jwk(key_data)
            else:
                raise ValidationError(f"Unsupported key type: {key_data.get('kty')}")

            decoded = jwt.decode(
                token,
                key,
                algorithms=["ES256", "RS256"],
                audience=self.config.client_id,
            )

            return TokenClaims(
                sub=decoded["sub"],
                iss=decoded["iss"],
                aud=decoded["aud"],
                exp=decoded["exp"],
                iat=decoded["iat"],
                scope=decoded.get("scope"),
                client_id=decoded.get("client_id"),
            )
        except jwt.exceptions.InvalidTokenError as e:
            raise ValidationError(f"Invalid token: {e}")

    async def client_credentials(self) -> TokenResponse:
        """Obtain token using client credentials flow."""
        if not self.config.client_secret:
            raise ValueError("client_secret required for client credentials flow")

        response = await self._request_with_retry(
            "POST",
            "/oauth/token",
            data={
                "grant_type": "client_credentials",
                "client_id": self.config.client_id,
                "client_secret": self.config.client_secret,
                "scope": " ".join(self.config.scopes) if self.config.scopes else None,
            },
        )

        token_response = TokenResponse(
            access_token=response["access_token"],
            token_type=response["token_type"],
            expires_in=response["expires_in"],
            refresh_token=response.get("refresh_token"),
            scope=response.get("scope"),
        )

        self._tokens = TokenData.from_response(token_response)
        return token_response

    async def _request_with_retry(
        self,
        method: str,
        path: str,
        **kwargs: Any,
    ) -> dict[str, Any]:
        """Make HTTP request with retry logic."""
        import asyncio

        last_error: Optional[Exception] = None

        for attempt in range(self.config.max_retries):
            try:
                response = await self._http.request(method, path, **kwargs)

                if response.status_code == 429:
                    retry_after = response.headers.get("Retry-After")
                    raise RateLimitError(
                        retry_after=int(retry_after) if retry_after else None
                    )

                response.raise_for_status()
                return response.json()

            except RateLimitError as e:
                delay = e.retry_after or (self.config.retry_delay * (2**attempt))
                await asyncio.sleep(delay)
                last_error = e

            except httpx.HTTPError as e:
                last_error = NetworkError(str(e))
                if attempt < self.config.max_retries - 1:
                    await asyncio.sleep(self.config.retry_delay * (2**attempt))

        raise last_error or NetworkError("Request failed")
