"""Centralized token operations for Auth Platform SDK - December 2025 State of Art.

Provides shared token request building and processing logic used by
both sync and async clients.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

from ..models import TokenData, TokenResponse

if TYPE_CHECKING:
    from ..config import AuthPlatformConfig
    from ..dpop import DPoPKeyPair


class TokenOperations:
    """Centralized token operations shared by sync and async clients.
    
    This class provides all token-related business logic that is identical
    between synchronous and asynchronous client implementations.
    """

    def __init__(
        self,
        config: AuthPlatformConfig,
        dpop_key: DPoPKeyPair | None = None,
    ) -> None:
        """Initialize token operations.
        
        Args:
            config: SDK configuration.
            dpop_key: Optional DPoP key pair for proof generation.
        """
        self.config = config
        self._dpop_key = dpop_key
        self._dpop_nonce: str | None = None
        self._tokens: TokenData | None = None

    @property
    def tokens(self) -> TokenData | None:
        """Get current token data."""
        return self._tokens

    @property
    def dpop_nonce(self) -> str | None:
        """Get current DPoP nonce."""
        return self._dpop_nonce

    def set_dpop_nonce(self, nonce: str | None) -> None:
        """Set DPoP nonce from server response."""
        self._dpop_nonce = nonce

    def build_client_credentials_request(
        self,
        scopes: list[str] | None = None,
    ) -> dict[str, Any]:
        """Build client credentials grant request payload.
        
        Args:
            scopes: Optional scopes to request.
            
        Returns:
            Request payload dictionary.
            
        Raises:
            ValueError: If client_secret not configured.
        """
        if not self.config.client_secret:
            msg = "client_secret required for client credentials flow"
            raise ValueError(msg)
        
        scope = " ".join(scopes) if scopes else self.config.scope_string
        data: dict[str, Any] = {
            "grant_type": "client_credentials",
            "client_id": self.config.client_id,
            "client_secret": self.config.client_secret.get_secret_value(),
        }
        if scope:
            data["scope"] = scope
        
        return data

    def build_refresh_token_request(
        self,
        refresh_token: str | None = None,
    ) -> dict[str, Any]:
        """Build refresh token grant request payload.
        
        Args:
            refresh_token: Refresh token (uses stored token if not provided).
            
        Returns:
            Request payload dictionary.
            
        Raises:
            ValueError: If no refresh token available.
        """
        token = refresh_token or (self._tokens.refresh_token if self._tokens else None)
        if not token:
            msg = "No refresh token available"
            raise ValueError(msg)
        
        data: dict[str, Any] = {
            "grant_type": "refresh_token",
            "refresh_token": token,
            "client_id": self.config.client_id,
        }
        if self.config.client_secret:
            data["client_secret"] = self.config.client_secret.get_secret_value()
        
        return data

    def build_authorization_code_request(
        self,
        code: str,
        redirect_uri: str,
        code_verifier: str | None = None,
    ) -> dict[str, Any]:
        """Build authorization code exchange request payload.
        
        Args:
            code: Authorization code.
            redirect_uri: Redirect URI used in authorization.
            code_verifier: PKCE code verifier.
            
        Returns:
            Request payload dictionary.
        """
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
        
        return data

    def build_token_request_headers(self) -> dict[str, str]:
        """Build headers for token request, including DPoP if enabled.
        
        Returns:
            Headers dictionary.
        """
        headers: dict[str, str] = {
            "Content-Type": "application/x-www-form-urlencoded",
        }
        
        if self._dpop_key:
            proof = self._dpop_key.create_proof(
                "POST",
                self.config.token_endpoint or "",
                nonce=self._dpop_nonce,
            )
            headers["DPoP"] = proof.proof
        
        return headers

    def process_token_response(
        self,
        response: TokenResponse,
    ) -> TokenData:
        """Process token response and update internal state.
        
        Args:
            response: Token response from server.
            
        Returns:
            Processed token data.
        """
        self._tokens = TokenData.from_response(
            response,
            buffer_seconds=self.config.cache.token_buffer,
        )
        
        # Update DPoP nonce if present
        if response.dpop_nonce:
            self._dpop_nonce = response.dpop_nonce
        
        return self._tokens

    def get_access_token(self, *, auto_refresh: bool = True) -> str | None:
        """Get valid access token if available.
        
        Args:
            auto_refresh: Whether token should be refreshed if expired.
            
        Returns:
            Access token if valid, None if expired/unavailable.
        """
        if self._tokens is None:
            return None
        
        if self._tokens.is_expired():
            if auto_refresh and self._tokens.refresh_token:
                return None  # Signal that refresh is needed
            return None
        
        return self._tokens.access_token

    def has_refresh_token(self) -> bool:
        """Check if a refresh token is available."""
        return self._tokens is not None and self._tokens.refresh_token is not None

    def clear_tokens(self) -> None:
        """Clear stored tokens."""
        self._tokens = None

    def is_token_expired(self) -> bool:
        """Check if current token is expired."""
        if self._tokens is None:
            return True
        return self._tokens.is_expired()
