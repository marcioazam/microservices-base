"""HTTP client utilities for Auth Platform SDK - December 2025 State of Art.

Provides resilient HTTP client with retry logic, circuit breaker,
and OpenTelemetry integration.
"""

from __future__ import annotations

import asyncio
import time
from enum import StrEnum
from typing import TYPE_CHECKING, Any

import httpx

from .errors import NetworkError, RateLimitError, ServerError, TimeoutError
from .telemetry import get_logger, trace_operation

if TYPE_CHECKING:
    from .config import AuthPlatformConfig, RetryConfig


class CircuitState(StrEnum):
    """Circuit breaker states."""

    CLOSED = "closed"
    OPEN = "open"
    HALF_OPEN = "half_open"


class CircuitBreaker:
    """Simple circuit breaker for resilience."""

    def __init__(
        self,
        failure_threshold: int = 5,
        recovery_timeout: float = 30.0,
        half_open_requests: int = 1,
    ) -> None:
        self.failure_threshold = failure_threshold
        self.recovery_timeout = recovery_timeout
        self.half_open_requests = half_open_requests

        self._state = CircuitState.CLOSED
        self._failure_count = 0
        self._last_failure_time: float = 0
        self._half_open_successes = 0

    @property
    def state(self) -> CircuitState:
        """Get current circuit state."""
        if self._state == CircuitState.OPEN:
            if time.time() - self._last_failure_time >= self.recovery_timeout:
                self._state = CircuitState.HALF_OPEN
                self._half_open_successes = 0
        return self._state

    def record_success(self) -> None:
        """Record a successful request."""
        if self._state == CircuitState.HALF_OPEN:
            self._half_open_successes += 1
            if self._half_open_successes >= self.half_open_requests:
                self._state = CircuitState.CLOSED
                self._failure_count = 0
        elif self._state == CircuitState.CLOSED:
            self._failure_count = 0

    def record_failure(self) -> None:
        """Record a failed request."""
        self._failure_count += 1
        self._last_failure_time = time.time()

        if self._state == CircuitState.HALF_OPEN:
            self._state = CircuitState.OPEN
        elif self._failure_count >= self.failure_threshold:
            self._state = CircuitState.OPEN

    def allow_request(self) -> bool:
        """Check if request should be allowed."""
        return self.state != CircuitState.OPEN


def create_http_client(config: AuthPlatformConfig) -> httpx.Client:
    """Create configured sync HTTP client.

    Args:
        config: SDK configuration.

    Returns:
        Configured httpx.Client.
    """
    return httpx.Client(
        base_url=config.base_url_str,
        timeout=httpx.Timeout(
            connect=config.connect_timeout,
            read=config.timeout,
            write=config.timeout,
            pool=config.timeout,
        ),
        headers={
            "User-Agent": "auth-platform-sdk/1.0.0 Python",
            "Accept": "application/json",
        },
        follow_redirects=False,
    )


def create_async_http_client(config: AuthPlatformConfig) -> httpx.AsyncClient:
    """Create configured async HTTP client.

    Args:
        config: SDK configuration.

    Returns:
        Configured httpx.AsyncClient.
    """
    return httpx.AsyncClient(
        base_url=config.base_url_str,
        timeout=httpx.Timeout(
            connect=config.connect_timeout,
            read=config.timeout,
            write=config.timeout,
            pool=config.timeout,
        ),
        headers={
            "User-Agent": "auth-platform-sdk/1.0.0 Python",
            "Accept": "application/json",
        },
        follow_redirects=False,
    )


def request_with_retry(
    client: httpx.Client,
    method: str,
    url: str,
    retry_config: RetryConfig,
    *,
    circuit_breaker: CircuitBreaker | None = None,
    **kwargs: Any,
) -> httpx.Response:
    """Make HTTP request with retry logic.

    Args:
        client: HTTP client.
        method: HTTP method.
        url: Request URL.
        retry_config: Retry configuration.
        circuit_breaker: Optional circuit breaker.
        **kwargs: Additional request arguments.

    Returns:
        HTTP response.

    Raises:
        NetworkError: On network failure after retries.
        RateLimitError: On rate limiting.
        ServerError: On server error.
    """
    logger = get_logger()
    last_error: Exception | None = None

    for attempt in range(retry_config.max_retries + 1):
        if circuit_breaker and not circuit_breaker.allow_request():
            raise NetworkError("Circuit breaker is open")

        try:
            with trace_operation(
                "http_request",
                attributes={"http.method": method, "http.url": url, "attempt": attempt},
            ):
                response = client.request(method, url, **kwargs)

                if response.status_code == 429:
                    retry_after = response.headers.get("Retry-After")
                    if circuit_breaker:
                        circuit_breaker.record_failure()
                    raise RateLimitError(
                        retry_after=int(retry_after) if retry_after else None
                    )

                if response.status_code >= 500:
                    if circuit_breaker:
                        circuit_breaker.record_failure()
                    raise ServerError(
                        f"Server error: {response.status_code}",
                        status_code=response.status_code,
                    )

                if circuit_breaker:
                    circuit_breaker.record_success()

                return response

        except RateLimitError as e:
            last_error = e
            delay = e.retry_after or retry_config.get_delay(attempt)
            logger.warning(
                "Rate limited, retrying",
                attempt=attempt,
                delay=delay,
            )
            time.sleep(delay)

        except (httpx.TimeoutException, httpx.ConnectError) as e:
            last_error = NetworkError(str(e), cause=e)
            if circuit_breaker:
                circuit_breaker.record_failure()

            if attempt < retry_config.max_retries:
                delay = retry_config.get_delay(attempt)
                logger.warning(
                    "Request failed, retrying",
                    attempt=attempt,
                    delay=delay,
                    error=str(e),
                )
                time.sleep(delay)

        except httpx.HTTPError as e:
            if circuit_breaker:
                circuit_breaker.record_failure()
            raise NetworkError(str(e), cause=e) from e

    raise last_error or NetworkError("Request failed after retries")


async def async_request_with_retry(
    client: httpx.AsyncClient,
    method: str,
    url: str,
    retry_config: RetryConfig,
    *,
    circuit_breaker: CircuitBreaker | None = None,
    **kwargs: Any,
) -> httpx.Response:
    """Make async HTTP request with retry logic.

    Args:
        client: Async HTTP client.
        method: HTTP method.
        url: Request URL.
        retry_config: Retry configuration.
        circuit_breaker: Optional circuit breaker.
        **kwargs: Additional request arguments.

    Returns:
        HTTP response.

    Raises:
        NetworkError: On network failure after retries.
        RateLimitError: On rate limiting.
        ServerError: On server error.
    """
    logger = get_logger()
    last_error: Exception | None = None

    for attempt in range(retry_config.max_retries + 1):
        if circuit_breaker and not circuit_breaker.allow_request():
            raise NetworkError("Circuit breaker is open")

        try:
            with trace_operation(
                "http_request",
                attributes={"http.method": method, "http.url": url, "attempt": attempt},
            ):
                response = await client.request(method, url, **kwargs)

                if response.status_code == 429:
                    retry_after = response.headers.get("Retry-After")
                    if circuit_breaker:
                        circuit_breaker.record_failure()
                    raise RateLimitError(
                        retry_after=int(retry_after) if retry_after else None
                    )

                if response.status_code >= 500:
                    if circuit_breaker:
                        circuit_breaker.record_failure()
                    raise ServerError(
                        f"Server error: {response.status_code}",
                        status_code=response.status_code,
                    )

                if circuit_breaker:
                    circuit_breaker.record_success()

                return response

        except RateLimitError as e:
            last_error = e
            delay = e.retry_after or retry_config.get_delay(attempt)
            logger.warning(
                "Rate limited, retrying",
                attempt=attempt,
                delay=delay,
            )
            await asyncio.sleep(delay)

        except (httpx.TimeoutException, httpx.ConnectError) as e:
            last_error = NetworkError(str(e), cause=e)
            if circuit_breaker:
                circuit_breaker.record_failure()

            if attempt < retry_config.max_retries:
                delay = retry_config.get_delay(attempt)
                logger.warning(
                    "Request failed, retrying",
                    attempt=attempt,
                    delay=delay,
                    error=str(e),
                )
                await asyncio.sleep(delay)

        except httpx.HTTPError as e:
            if circuit_breaker:
                circuit_breaker.record_failure()
            raise NetworkError(str(e), cause=e) from e

    raise last_error or NetworkError("Request failed after retries")
