"""Type definitions for Auth Platform SDK."""

from dataclasses import dataclass
from datetime import datetime
from typing import Optional


@dataclass
class TokenResponse:
    """OAuth token response."""

    access_token: str
    token_type: str
    expires_in: int
    refresh_token: Optional[str] = None
    scope: Optional[str] = None


@dataclass
class TokenData:
    """Stored token data."""

    access_token: str
    token_type: str
    expires_at: datetime
    refresh_token: Optional[str] = None
    scope: Optional[str] = None

    @classmethod
    def from_response(cls, response: TokenResponse) -> "TokenData":
        """Create TokenData from TokenResponse."""
        from datetime import timedelta

        return cls(
            access_token=response.access_token,
            token_type=response.token_type,
            expires_at=datetime.utcnow() + timedelta(seconds=response.expires_in),
            refresh_token=response.refresh_token,
            scope=response.scope,
        )

    def is_expired(self, buffer_seconds: int = 60) -> bool:
        """Check if token is expired or about to expire."""
        from datetime import timedelta

        return datetime.utcnow() >= self.expires_at - timedelta(seconds=buffer_seconds)


@dataclass
class TokenClaims:
    """JWT token claims."""

    sub: str
    iss: str
    aud: str | list[str]
    exp: int
    iat: int
    scope: Optional[str] = None
    client_id: Optional[str] = None
