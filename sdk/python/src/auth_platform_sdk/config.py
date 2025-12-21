"""Configuration for Auth Platform SDK - December 2025 State of Art.

Uses Pydantic v2 for validation with sensible defaults and
comprehensive configuration options.
"""

from __future__ import annotations

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


class RetryConfig(BaseModel):
    """Retry configuration with exponential backoff."""

    model_config = ConfigDict(frozen=True)

    max_retries: Annotated[int, Field(ge=0, le=10)] = 3
    initial_delay: Annotated[float, Field(gt=0, le=60)] = 1.0
    max_delay: Annotated[float, Field(gt=0, le=300)] = 30.0
    exponential_base: Annotated[float, Field(ge=1.5, le=3.0)] = 2.0
    jitter: Annotated[float, Field(ge=0, le=1.0)] = 0.1

    def get_delay(self, attempt: int) -> float:
        """Calculate delay for given attempt with exponential backoff."""
        import random

        delay = min(
            self.initial_delay * (self.exponential_base**attempt),
            self.max_delay,
        )
        # Add jitter to prevent thundering herd
        jitter_range = delay * self.jitter
        return delay + random.uniform(-jitter_range, jitter_range)  # noqa: S311


class TelemetryConfig(BaseModel):
    """OpenTelemetry configuration."""

    model_config = ConfigDict(frozen=True)

    enabled: bool = True
    service_name: str = "auth-platform-sdk"
    trace_requests: bool = True
    trace_token_operations: bool = True
    log_level: str = "INFO"


class DPoPConfig(BaseModel):
    """DPoP (Demonstrating Proof of Possession) configuration."""

    model_config = ConfigDict(frozen=True)

    enabled: bool = False
    algorithm: str = "ES256"
    key_rotation_interval: Annotated[int, Field(gt=0)] = 3600  # 1 hour

    @field_validator("algorithm")
    @classmethod
    def validate_algorithm(cls, v: str) -> str:
        """Validate DPoP algorithm is supported."""
        supported = {"ES256", "ES384", "ES512", "RS256", "RS384", "RS512"}
        if v not in supported:
            msg = f"Unsupported DPoP algorithm: {v}. Supported: {supported}"
            raise ValueError(msg)
        return v


class CacheConfig(BaseModel):
    """Cache configuration for JWKS and tokens."""

    model_config = ConfigDict(frozen=True)

    jwks_ttl: Annotated[int, Field(gt=0)] = 3600  # 1 hour
    jwks_refresh_ahead: Annotated[int, Field(ge=0)] = 300  # 5 minutes
    token_buffer: Annotated[int, Field(ge=0)] = 60  # 1 minute before expiry


class AuthPlatformConfig(BaseModel):
    """Main configuration for Auth Platform SDK."""

    model_config = ConfigDict(frozen=True, validate_default=True)

    # Required
    base_url: HttpUrl
    client_id: str = Field(..., min_length=1)

    # Authentication
    client_secret: SecretStr | None = None
    scopes: list[str] = Field(default_factory=list)

    # HTTP settings
    timeout: Annotated[float, Field(gt=0, le=300)] = 30.0
    connect_timeout: Annotated[float, Field(gt=0, le=60)] = 10.0

    # Sub-configurations
    retry: RetryConfig = Field(default_factory=RetryConfig)
    telemetry: TelemetryConfig = Field(default_factory=TelemetryConfig)
    dpop: DPoPConfig = Field(default_factory=DPoPConfig)
    cache: CacheConfig = Field(default_factory=CacheConfig)

    # Endpoints (auto-derived from base_url if not set)
    token_endpoint: str | None = None
    authorization_endpoint: str | None = None
    jwks_uri: str | None = None
    userinfo_endpoint: str | None = None
    revocation_endpoint: str | None = None
    introspection_endpoint: str | None = None

    @model_validator(mode="after")
    def set_default_endpoints(self) -> Self:
        """Set default endpoints based on base_url."""
        base = str(self.base_url).rstrip("/")

        # Use object.__setattr__ since model is frozen
        if self.token_endpoint is None:
            object.__setattr__(self, "token_endpoint", f"{base}/oauth/token")
        if self.authorization_endpoint is None:
            object.__setattr__(self, "authorization_endpoint", f"{base}/oauth/authorize")
        if self.jwks_uri is None:
            object.__setattr__(self, "jwks_uri", f"{base}/.well-known/jwks.json")
        if self.userinfo_endpoint is None:
            object.__setattr__(self, "userinfo_endpoint", f"{base}/oauth/userinfo")
        if self.revocation_endpoint is None:
            object.__setattr__(self, "revocation_endpoint", f"{base}/oauth/revoke")
        if self.introspection_endpoint is None:
            object.__setattr__(self, "introspection_endpoint", f"{base}/oauth/introspect")

        return self

    @property
    def base_url_str(self) -> str:
        """Get base URL as string without trailing slash."""
        return str(self.base_url).rstrip("/")

    @property
    def scope_string(self) -> str | None:
        """Get scopes as space-separated string."""
        return " ".join(self.scopes) if self.scopes else None

    def with_overrides(self, **kwargs: Any) -> Self:
        """Create new config with overridden values."""
        data = self.model_dump()
        data.update(kwargs)
        return self.__class__(**data)

    @classmethod
    def from_env(cls, prefix: str = "AUTH_PLATFORM_") -> Self:
        """Create config from environment variables."""
        import os

        def get_env(key: str, default: Any = None) -> Any:
            return os.environ.get(f"{prefix}{key}", default)

        base_url = get_env("BASE_URL")
        if not base_url:
            msg = f"{prefix}BASE_URL environment variable is required"
            raise ValueError(msg)

        client_id = get_env("CLIENT_ID")
        if not client_id:
            msg = f"{prefix}CLIENT_ID environment variable is required"
            raise ValueError(msg)

        scopes_str = get_env("SCOPES", "")
        scopes = scopes_str.split() if scopes_str else []

        return cls(
            base_url=base_url,
            client_id=client_id,
            client_secret=get_env("CLIENT_SECRET"),
            scopes=scopes,
            timeout=float(get_env("TIMEOUT", "30.0")),
        )
