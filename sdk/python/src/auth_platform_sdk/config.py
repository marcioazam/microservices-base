"""Configuration for Auth Platform SDK."""

from dataclasses import dataclass, field
from typing import Optional


@dataclass
class AuthPlatformConfig:
    """Configuration for Auth Platform client."""

    base_url: str
    client_id: str
    client_secret: Optional[str] = None
    scopes: list[str] = field(default_factory=list)
    timeout: float = 30.0
    jwks_cache_ttl: int = 3600  # 1 hour
    max_retries: int = 3
    retry_delay: float = 1.0

    def __post_init__(self) -> None:
        if not self.base_url:
            raise ValueError("base_url is required")
        if not self.client_id:
            raise ValueError("client_id is required")
        # Remove trailing slash
        self.base_url = self.base_url.rstrip("/")
