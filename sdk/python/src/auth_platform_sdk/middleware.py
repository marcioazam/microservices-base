"""Framework middleware for Auth Platform SDK - December 2025 State of Art.

Provides authentication middleware for FastAPI, Flask, and Django
with async support and OpenTelemetry integration.
"""

from __future__ import annotations

from typing import TYPE_CHECKING, Any, Callable

from .async_client import AsyncAuthPlatformClient
from .client import AuthPlatformClient
from .errors import AuthPlatformError, ValidationError
from .models import TokenClaims

if TYPE_CHECKING:
    from .config import AuthPlatformConfig


def create_fastapi_middleware(config: AuthPlatformConfig) -> Any:
    """Create FastAPI dependency for token validation.

    Args:
        config: SDK configuration.

    Returns:
        FastAPI dependency function.

    Raises:
        ImportError: If FastAPI not installed.
    """
    try:
        from fastapi import Depends, HTTPException, Request
        from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
    except ImportError as e:
        msg = "FastAPI not installed. Install with: pip install fastapi"
        raise ImportError(msg) from e

    security = HTTPBearer(auto_error=True)
    client = AsyncAuthPlatformClient(config)

    async def get_current_user(
        request: Request,
        credentials: HTTPAuthorizationCredentials = Depends(security),
    ) -> TokenClaims:
        """Dependency to get current user from token."""
        try:
            claims = await client.validate_token(credentials.credentials)

            # Store claims in request state for access in route handlers
            request.state.user_claims = claims

            return claims

        except ValidationError as e:
            raise HTTPException(
                status_code=401,
                detail=str(e),
                headers={"WWW-Authenticate": "Bearer"},
            ) from e
        except AuthPlatformError as e:
            raise HTTPException(
                status_code=e.status_code or 500,
                detail=str(e),
            ) from e

    return get_current_user


def create_fastapi_optional_auth(config: AuthPlatformConfig) -> Any:
    """Create FastAPI dependency for optional token validation.

    Args:
        config: SDK configuration.

    Returns:
        FastAPI dependency function that returns None if no token.
    """
    try:
        from fastapi import Depends, Request
        from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
    except ImportError as e:
        msg = "FastAPI not installed. Install with: pip install fastapi"
        raise ImportError(msg) from e

    security = HTTPBearer(auto_error=False)
    client = AsyncAuthPlatformClient(config)

    async def get_optional_user(
        request: Request,
        credentials: HTTPAuthorizationCredentials | None = Depends(security),
    ) -> TokenClaims | None:
        """Dependency to optionally get current user from token."""
        if credentials is None:
            return None

        try:
            claims = await client.validate_token(credentials.credentials)
            request.state.user_claims = claims
            return claims
        except (ValidationError, AuthPlatformError):
            return None

    return get_optional_user


def create_flask_middleware(config: AuthPlatformConfig) -> Callable[..., Any]:
    """Create Flask decorator for token validation.

    Args:
        config: SDK configuration.

    Returns:
        Decorator function.

    Raises:
        ImportError: If Flask not installed.
    """
    try:
        from flask import g, request
        from functools import wraps
    except ImportError as e:
        msg = "Flask not installed. Install with: pip install flask"
        raise ImportError(msg) from e

    client = AuthPlatformClient(config)

    def require_auth(f: Callable[..., Any]) -> Callable[..., Any]:
        """Decorator to require authentication."""

        @wraps(f)
        def decorated(*args: Any, **kwargs: Any) -> Any:
            auth_header = request.headers.get("Authorization")

            if not auth_header:
                return {"error": "Missing authorization header"}, 401

            if not auth_header.startswith("Bearer "):
                return {"error": "Invalid authorization header format"}, 401

            token = auth_header[7:]  # Remove "Bearer " prefix

            try:
                g.current_user = client.validate_token(token)
            except ValidationError as e:
                return {"error": str(e)}, 401
            except AuthPlatformError as e:
                return {"error": str(e)}, e.status_code or 500

            return f(*args, **kwargs)

        return decorated

    return require_auth


def create_flask_optional_auth(config: AuthPlatformConfig) -> Callable[..., Any]:
    """Create Flask decorator for optional token validation.

    Args:
        config: SDK configuration.

    Returns:
        Decorator function that sets g.current_user to None if no token.
    """
    try:
        from flask import g, request
        from functools import wraps
    except ImportError as e:
        msg = "Flask not installed. Install with: pip install flask"
        raise ImportError(msg) from e

    client = AuthPlatformClient(config)

    def optional_auth(f: Callable[..., Any]) -> Callable[..., Any]:
        """Decorator for optional authentication."""

        @wraps(f)
        def decorated(*args: Any, **kwargs: Any) -> Any:
            g.current_user = None
            auth_header = request.headers.get("Authorization")

            if auth_header and auth_header.startswith("Bearer "):
                token = auth_header[7:]
                try:
                    g.current_user = client.validate_token(token)
                except (ValidationError, AuthPlatformError):
                    pass  # Token invalid, but that's OK for optional auth

            return f(*args, **kwargs)

        return decorated

    return optional_auth


class DjangoAuthMiddleware:
    """Django middleware for token validation."""

    def __init__(
        self,
        get_response: Callable[..., Any],
        config: AuthPlatformConfig,
    ) -> None:
        """Initialize middleware.

        Args:
            get_response: Django's get_response callable.
            config: SDK configuration.
        """
        try:
            from django.http import JsonResponse
        except ImportError as e:
            msg = "Django not installed. Install with: pip install django"
            raise ImportError(msg) from e

        self.get_response = get_response
        self.client = AuthPlatformClient(config)
        self.exempt_paths: list[str] = []
        self.exempt_methods: set[str] = {"OPTIONS"}

    def __call__(self, request: Any) -> Any:
        """Process request."""
        from django.http import JsonResponse

        # Skip exempt methods
        if request.method in self.exempt_methods:
            return self.get_response(request)

        # Skip exempt paths
        if any(request.path.startswith(path) for path in self.exempt_paths):
            return self.get_response(request)

        auth_header = request.headers.get("Authorization")

        if not auth_header:
            return JsonResponse(
                {"error": "Missing authorization header"},
                status=401,
            )

        if not auth_header.startswith("Bearer "):
            return JsonResponse(
                {"error": "Invalid authorization header format"},
                status=401,
            )

        token = auth_header[7:]

        try:
            request.user_claims = self.client.validate_token(token)
        except ValidationError as e:
            return JsonResponse({"error": str(e)}, status=401)
        except AuthPlatformError as e:
            return JsonResponse({"error": str(e)}, status=e.status_code or 500)

        return self.get_response(request)

    def add_exempt_path(self, path: str) -> None:
        """Add a path that doesn't require authentication."""
        self.exempt_paths.append(path)

    def add_exempt_method(self, method: str) -> None:
        """Add an HTTP method that doesn't require authentication."""
        self.exempt_methods.add(method.upper())


def create_django_middleware(config: AuthPlatformConfig) -> type:
    """Create Django middleware class with config.

    Args:
        config: SDK configuration.

    Returns:
        Configured middleware class.
    """

    class ConfiguredDjangoAuthMiddleware(DjangoAuthMiddleware):
        def __init__(self, get_response: Callable[..., Any]) -> None:
            super().__init__(get_response, config)

    return ConfiguredDjangoAuthMiddleware
