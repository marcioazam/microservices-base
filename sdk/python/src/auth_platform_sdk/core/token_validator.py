"""Centralized token validation for Auth Platform SDK - December 2025 State of Art.

Provides shared JWT validation logic used by both sync and async clients.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any

import jwt
from jwt import algorithms

from ..errors import TokenExpiredError, ValidationError
from ..models import JWK, TokenClaims

if TYPE_CHECKING:
    from ..config import AuthPlatformConfig


# Supported algorithms for token validation
SUPPORTED_ALGORITHMS = ["ES256", "ES384", "ES512", "RS256", "RS384", "RS512"]


class TokenValidator:
    """Centralized token validator shared by sync and async clients.
    
    This class provides JWT validation logic that is identical
    between synchronous and asynchronous client implementations.
    """

    def __init__(
        self,
        config: AuthPlatformConfig,
        *,
        algorithms: list[str] | None = None,
    ) -> None:
        """Initialize token validator.
        
        Args:
            config: SDK configuration.
            algorithms: Allowed algorithms (defaults to SUPPORTED_ALGORITHMS).
        """
        self.config = config
        self._algorithms = algorithms or SUPPORTED_ALGORITHMS

    def validate(
        self,
        token: str,
        jwk: JWK,
        *,
        audience: str | list[str] | None = None,
        issuer: str | None = None,
        options: dict[str, Any] | None = None,
    ) -> TokenClaims:
        """Validate JWT token and return claims.
        
        Args:
            token: JWT access token.
            jwk: JWK to use for validation.
            audience: Expected audience (defaults to client_id).
            issuer: Expected issuer (defaults to config issuer).
            options: Additional PyJWT decode options.
            
        Returns:
            Validated token claims.
            
        Raises:
            TokenExpiredError: If token is expired.
            ValidationError: If token is invalid.
        """
        try:
            key = self._build_public_key(jwk)
            
            decode_options = options or {}
            decode_kwargs: dict[str, Any] = {
                "algorithms": self._algorithms,
                "options": decode_options,
            }
            
            # Set audience if provided
            if audience is not None:
                decode_kwargs["audience"] = audience
            elif self.config.client_id:
                decode_kwargs["audience"] = self.config.client_id
            
            # Set issuer if provided
            if issuer is not None:
                decode_kwargs["issuer"] = issuer
            
            decoded = jwt.decode(token, key, **decode_kwargs)
            return TokenClaims(**decoded)
            
        except jwt.exceptions.ExpiredSignatureError as e:
            raise TokenExpiredError() from e
        except jwt.exceptions.InvalidAudienceError as e:
            raise ValidationError(f"Invalid audience: {e}") from e
        except jwt.exceptions.InvalidIssuerError as e:
            raise ValidationError(f"Invalid issuer: {e}") from e
        except jwt.exceptions.InvalidTokenError as e:
            raise ValidationError(f"Invalid token: {e}") from e

    def validate_with_key(
        self,
        token: str,
        key: Any,
        *,
        audience: str | list[str] | None = None,
        issuer: str | None = None,
        options: dict[str, Any] | None = None,
    ) -> TokenClaims:
        """Validate JWT token with a pre-built key.
        
        Args:
            token: JWT access token.
            key: Pre-built cryptographic key.
            audience: Expected audience.
            issuer: Expected issuer.
            options: Additional PyJWT decode options.
            
        Returns:
            Validated token claims.
            
        Raises:
            TokenExpiredError: If token is expired.
            ValidationError: If token is invalid.
        """
        try:
            decode_options = options or {}
            decode_kwargs: dict[str, Any] = {
                "algorithms": self._algorithms,
                "options": decode_options,
            }
            
            if audience is not None:
                decode_kwargs["audience"] = audience
            elif self.config.client_id:
                decode_kwargs["audience"] = self.config.client_id
            
            if issuer is not None:
                decode_kwargs["issuer"] = issuer
            
            decoded = jwt.decode(token, key, **decode_kwargs)
            return TokenClaims(**decoded)
            
        except jwt.exceptions.ExpiredSignatureError as e:
            raise TokenExpiredError() from e
        except jwt.exceptions.InvalidAudienceError as e:
            raise ValidationError(f"Invalid audience: {e}") from e
        except jwt.exceptions.InvalidIssuerError as e:
            raise ValidationError(f"Invalid issuer: {e}") from e
        except jwt.exceptions.InvalidTokenError as e:
            raise ValidationError(f"Invalid token: {e}") from e

    def _build_public_key(self, jwk: JWK) -> Any:
        """Build public key from JWK.
        
        Args:
            jwk: JWK to convert.
            
        Returns:
            Cryptographic public key.
            
        Raises:
            ValidationError: If key type is unsupported.
        """
        jwk_dict = jwk.model_dump(exclude_none=True)
        
        if jwk.kty == "EC":
            return algorithms.ECAlgorithm.from_jwk(jwk_dict)
        elif jwk.kty == "RSA":
            return algorithms.RSAAlgorithm.from_jwk(jwk_dict)
        else:
            raise ValidationError(f"Unsupported key type: {jwk.kty}")

    def get_unverified_header(self, token: str) -> dict[str, Any]:
        """Get token header without verification.
        
        Args:
            token: JWT token.
            
        Returns:
            Token header dictionary.
            
        Raises:
            ValidationError: If token format is invalid.
        """
        try:
            return jwt.get_unverified_header(token)
        except jwt.exceptions.DecodeError as e:
            raise ValidationError(f"Invalid token format: {e}") from e

    def get_key_id(self, token: str) -> str | None:
        """Extract key ID from token header.
        
        Args:
            token: JWT token.
            
        Returns:
            Key ID if present, None otherwise.
        """
        header = self.get_unverified_header(token)
        return header.get("kid")

    def verify_dpop_binding(
        self,
        claims: TokenClaims,
        expected_thumbprint: str,
    ) -> bool:
        """Verify DPoP binding in token claims.
        
        Args:
            claims: Token claims.
            expected_thumbprint: Expected JWK thumbprint.
            
        Returns:
            True if binding is valid, False otherwise.
        """
        if claims.cnf is None:
            return False
        
        jkt = claims.cnf.get("jkt")
        if jkt is None:
            return False
        
        # Use constant-time comparison for security
        import secrets
        return secrets.compare_digest(jkt, expected_thumbprint)
