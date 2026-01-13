"""Centralized authorization URL builder for Auth Platform SDK - December 2025 State of Art.

Provides shared authorization URL construction logic used by
both sync and async clients.
"""

from __future__ import annotations

from typing import TYPE_CHECKING
from urllib.parse import urlencode

from ..pkce import create_pkce_challenge, generate_nonce, generate_state

if TYPE_CHECKING:
    from ..config import AuthPlatformConfig
    from ..models import PKCEChallenge


class AuthorizationBuilder:
    """Centralized authorization URL builder shared by sync and async clients.
    
    This class provides authorization URL construction logic that is identical
    between synchronous and asynchronous client implementations.
    """

    def __init__(self, config: AuthPlatformConfig) -> None:
        """Initialize authorization builder.
        
        Args:
            config: SDK configuration.
        """
        self.config = config

    def build_authorization_url(
        self,
        redirect_uri: str,
        *,
        scopes: list[str] | None = None,
        state: str | None = None,
        nonce: str | None = None,
        use_pkce: bool = True,
        extra_params: dict[str, str] | None = None,
    ) -> tuple[str, str, PKCEChallenge | None]:
        """Build authorization URL for authorization code flow.
        
        Args:
            redirect_uri: Redirect URI after authorization.
            scopes: Scopes to request.
            state: CSRF state (generated if not provided).
            nonce: OpenID Connect nonce (generated if not provided).
            use_pkce: Whether to use PKCE.
            extra_params: Additional query parameters.
            
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

        if extra_params:
            params.update(extra_params)

        query = urlencode(params)
        url = f"{self.config.authorization_endpoint}?{query}"

        return url, state, pkce

    def parse_callback_url(
        self,
        callback_url: str,
        expected_state: str,
    ) -> tuple[str, str | None]:
        """Parse authorization callback URL and extract code.
        
        Args:
            callback_url: The callback URL with authorization response.
            expected_state: The expected state parameter for CSRF validation.
            
        Returns:
            Tuple of (authorization_code, error_description).
            
        Raises:
            ValueError: If state doesn't match or error in response.
        """
        from urllib.parse import parse_qs, urlparse

        parsed = urlparse(callback_url)
        params = parse_qs(parsed.query)

        # Check for error response
        if "error" in params:
            error = params["error"][0]
            error_desc = params.get("error_description", [""])[0]
            msg = f"Authorization error: {error}"
            if error_desc:
                msg += f" - {error_desc}"
            raise ValueError(msg)

        # Validate state
        state = params.get("state", [""])[0]
        if state != expected_state:
            msg = "State mismatch - possible CSRF attack"
            raise ValueError(msg)

        # Extract code
        if "code" not in params:
            msg = "No authorization code in callback"
            raise ValueError(msg)

        return params["code"][0], None
