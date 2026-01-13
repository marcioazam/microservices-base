"""Centralized HTTP executors for Auth Platform SDK - December 2025 State of Art.

Provides shared HTTP execution logic with retry and circuit breaker
for both sync and async clients.
"""

from __future__ import annotations

import asyncio
import time
from abc import ABC, abstractmethod
from typing import TYPE_CHECKING, Any, Protocol

import httpx

from ..errors import NetworkError, RateLimitError, ServerError
from ..http import CircuitBreaker
from ..telemetry import get_logger, trace_operation

if TYPE_CHECKING:
    from ..config import AuthPlatformConfig, RetryConfig


class HTTPExecutorProtocol(Protocol):
    """Protocol for HTTP executors."""

    def execute(
        self,
        method: str,
        url: str,
        **kwargs: Any,
    ) -> httpx.Response:
        """Execute HTTP request."""
        ...


def calculate_retry_delay(retry_config: RetryConfig, attempt: int) -> float:
    """Calculate retry delay with exponential backoff.
    
    Args:
        retry_config: Retry configuration.
        attempt: Current attempt number (0-indexed).
        
    Returns:
        Delay in seconds.
    """
    return retry_config.get_delay(attempt)


def should_retry_status(status_code: int) -> bool:
    """Check if status code should trigger retry.
    
    Args:
        status_code: HTTP status code.
        
    Returns:
        True if should retry.
    """
    return status_code == 429 or status_code >= 500


class SyncHTTPExecutor:
    """Synchronous HTTP executor with retry and circuit breaker."""

    def __init__(
        self,
        client: httpx.Client,
        retry_config: RetryConfig,
        circuit_breaker: CircuitBreaker | None = None,
    ) -> None:
        """Initialize sync HTTP executor.
        
        Args:
            client: HTTP client.
            retry_config: Retry configuration.
            circuit_breaker: Optional circuit breaker.
        """
        self._client = client
        self._retry_config = retry_config
        self._circuit_breaker = circuit_breaker or CircuitBreaker()
        self._logger = get_logger()

    @property
    def circuit_breaker(self) -> CircuitBreaker:
        """Get circuit breaker."""
        return self._circuit_breaker

    def execute(
        self,
        method: str,
        url: str,
        **kwargs: Any,
    ) -> httpx.Response:
        """Execute HTTP request with retry logic.
        
        Args:
            method: HTTP method.
            url: Request URL.
            **kwargs: Additional request arguments.
            
        Returns:
            HTTP response.
            
        Raises:
            NetworkError: On network failure after retries.
            RateLimitError: On rate limiting.
            ServerError: On server error.
        """
        last_error: Exception | None = None

        for attempt in range(self._retry_config.max_retries + 1):
            if not self._circuit_breaker.allow_request():
                raise NetworkError("Circuit breaker is open")

            try:
                response = self._execute_single(method, url, attempt, **kwargs)
                return response

            except RateLimitError as e:
                last_error = e
                delay = e.retry_after or calculate_retry_delay(
                    self._retry_config, attempt
                )
                self._log_retry("Rate limited", attempt, delay)
                time.sleep(delay)

            except (httpx.TimeoutException, httpx.ConnectError) as e:
                last_error = NetworkError(str(e), cause=e)
                self._circuit_breaker.record_failure()

                if attempt < self._retry_config.max_retries:
                    delay = calculate_retry_delay(self._retry_config, attempt)
                    self._log_retry("Request failed", attempt, delay, str(e))
                    time.sleep(delay)

            except httpx.HTTPError as e:
                self._circuit_breaker.record_failure()
                raise NetworkError(str(e), cause=e) from e

        raise last_error or NetworkError("Request failed after retries")

    def _execute_single(
        self,
        method: str,
        url: str,
        attempt: int,
        **kwargs: Any,
    ) -> httpx.Response:
        """Execute single HTTP request.
        
        Args:
            method: HTTP method.
            url: Request URL.
            attempt: Current attempt number.
            **kwargs: Additional request arguments.
            
        Returns:
            HTTP response.
            
        Raises:
            RateLimitError: On rate limiting.
            ServerError: On server error.
        """
        with trace_operation(
            "http_request",
            attributes={"http.method": method, "http.url": url, "attempt": attempt},
        ):
            response = self._client.request(method, url, **kwargs)

            if response.status_code == 429:
                retry_after = response.headers.get("Retry-After")
                self._circuit_breaker.record_failure()
                raise RateLimitError(
                    retry_after=int(retry_after) if retry_after else None
                )

            if response.status_code >= 500:
                self._circuit_breaker.record_failure()
                raise ServerError(
                    f"Server error: {response.status_code}",
                    status_code=response.status_code,
                )

            self._circuit_breaker.record_success()
            return response

    def _log_retry(
        self,
        message: str,
        attempt: int,
        delay: float,
        error: str | None = None,
    ) -> None:
        """Log retry attempt."""
        self._logger.warning(
            message,
            attempt=attempt,
            delay=delay,
            error=error,
        )


class AsyncHTTPExecutor:
    """Asynchronous HTTP executor with retry and circuit breaker."""

    def __init__(
        self,
        client: httpx.AsyncClient,
        retry_config: RetryConfig,
        circuit_breaker: CircuitBreaker | None = None,
    ) -> None:
        """Initialize async HTTP executor.
        
        Args:
            client: Async HTTP client.
            retry_config: Retry configuration.
            circuit_breaker: Optional circuit breaker.
        """
        self._client = client
        self._retry_config = retry_config
        self._circuit_breaker = circuit_breaker or CircuitBreaker()
        self._logger = get_logger()

    @property
    def circuit_breaker(self) -> CircuitBreaker:
        """Get circuit breaker."""
        return self._circuit_breaker

    async def execute(
        self,
        method: str,
        url: str,
        **kwargs: Any,
    ) -> httpx.Response:
        """Execute async HTTP request with retry logic.
        
        Args:
            method: HTTP method.
            url: Request URL.
            **kwargs: Additional request arguments.
            
        Returns:
            HTTP response.
            
        Raises:
            NetworkError: On network failure after retries.
            RateLimitError: On rate limiting.
            ServerError: On server error.
        """
        last_error: Exception | None = None

        for attempt in range(self._retry_config.max_retries + 1):
            if not self._circuit_breaker.allow_request():
                raise NetworkError("Circuit breaker is open")

            try:
                response = await self._execute_single(method, url, attempt, **kwargs)
                return response

            except RateLimitError as e:
                last_error = e
                delay = e.retry_after or calculate_retry_delay(
                    self._retry_config, attempt
                )
                self._log_retry("Rate limited", attempt, delay)
                await asyncio.sleep(delay)

            except (httpx.TimeoutException, httpx.ConnectError) as e:
                last_error = NetworkError(str(e), cause=e)
                self._circuit_breaker.record_failure()

                if attempt < self._retry_config.max_retries:
                    delay = calculate_retry_delay(self._retry_config, attempt)
                    self._log_retry("Request failed", attempt, delay, str(e))
                    await asyncio.sleep(delay)

            except httpx.HTTPError as e:
                self._circuit_breaker.record_failure()
                raise NetworkError(str(e), cause=e) from e

        raise last_error or NetworkError("Request failed after retries")

    async def _execute_single(
        self,
        method: str,
        url: str,
        attempt: int,
        **kwargs: Any,
    ) -> httpx.Response:
        """Execute single async HTTP request.
        
        Args:
            method: HTTP method.
            url: Request URL.
            attempt: Current attempt number.
            **kwargs: Additional request arguments.
            
        Returns:
            HTTP response.
            
        Raises:
            RateLimitError: On rate limiting.
            ServerError: On server error.
        """
        with trace_operation(
            "http_request",
            attributes={"http.method": method, "http.url": url, "attempt": attempt},
        ):
            response = await self._client.request(method, url, **kwargs)

            if response.status_code == 429:
                retry_after = response.headers.get("Retry-After")
                self._circuit_breaker.record_failure()
                raise RateLimitError(
                    retry_after=int(retry_after) if retry_after else None
                )

            if response.status_code >= 500:
                self._circuit_breaker.record_failure()
                raise ServerError(
                    f"Server error: {response.status_code}",
                    status_code=response.status_code,
                )

            self._circuit_breaker.record_success()
            return response

    def _log_retry(
        self,
        message: str,
        attempt: int,
        delay: float,
        error: str | None = None,
    ) -> None:
        """Log retry attempt."""
        self._logger.warning(
            message,
            attempt=attempt,
            delay=delay,
            error=error,
        )
