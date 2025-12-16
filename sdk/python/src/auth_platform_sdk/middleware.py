"""Framework middleware for Auth Platform SDK."""

from typing import Any, Callable, Optional

from .client import AuthPlatformClient, AsyncAuthPlatformClient
from .config import AuthPlatformConfig
from .errors import AuthPlatformError, ValidationError
from .types import TokenClaims


# FastAPI Middleware
def create_fastapi_middleware(config: AuthPlatformConfig) -> Any:
    """Create FastAPI middleware for token validation."""
    try:
        from fastapi import Depends, HTTPException, Request
        from fastapi.security import HTTPAuthorizationCredentials, HTTPBearer
    except ImportError:
        raise ImportError("FastAPI not installed. Install with: pip install fastapi")

    security = HTTPBearer()
    client = AsyncAuthPlatformClient(config)

    async def get_current_user(
        credentials: HTTPAuthorizationCredentials = Depends(security),
    ) -> TokenClaims:
        """Dependency to get current user from token."""
        try:
            return await client.validate_token(credentials.credentials)
        except ValidationError as e:
            raise HTTPException(status_code=401, detail=str(e))
        except AuthPlatformError as e:
            raise HTTPException(status_code=e.status_code or 500, detail=str(e))

    return get_current_user


# Flask Middleware
def create_flask_middleware(config: AuthPlatformConfig) -> Callable[..., Any]:
    """Create Flask decorator for token validation."""
    try:
        from flask import g, request
        from functools import wraps
    except ImportError:
        raise ImportError("Flask not installed. Install with: pip install flask")

    client = AuthPlatformClient(config)

    def require_auth(f: Callable[..., Any]) -> Callable[..., Any]:
        """Decorator to require authentication."""

        @wraps(f)
        def decorated(*args: Any, **kwargs: Any) -> Any:
            auth_header = request.headers.get("Authorization")
            if not auth_header or not auth_header.startswith("Bearer "):
                return {"error": "Missing or invalid authorization header"}, 401

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


# Django Middleware
class DjangoAuthMiddleware:
    """Django middleware for token validation."""

    def __init__(self, get_response: Callable[..., Any], config: AuthPlatformConfig):
        try:
            from django.http import JsonResponse
        except ImportError:
            raise ImportError("Django not installed. Install with: pip install django")

        self.get_response = get_response
        self.client = AuthPlatformClient(config)
        self.exempt_paths: list[str] = []

    def __call__(self, request: Any) -> Any:
        from django.http import JsonResponse

        # Skip exempt paths
        if any(request.path.startswith(path) for path in self.exempt_paths):
            return self.get_response(request)

        auth_header = request.headers.get("Authorization")
        if not auth_header or not auth_header.startswith("Bearer "):
            return JsonResponse(
                {"error": "Missing or invalid authorization header"},
                status=401,
            )

        token = auth_header[7:]  # Remove "Bearer " prefix

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


def create_django_middleware(config: AuthPlatformConfig) -> type:
    """Create Django middleware class with config."""

    class ConfiguredDjangoAuthMiddleware(DjangoAuthMiddleware):
        def __init__(self, get_response: Callable[..., Any]):
            super().__init__(get_response, config)

    return ConfiguredDjangoAuthMiddleware
