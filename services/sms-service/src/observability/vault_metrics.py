"""
Vault Metrics Module

Provides Prometheus metrics for monitoring Vault operations including:
- Operation counts and latency
- Token TTL monitoring
- Rate limiter status
- Cache statistics
- Error tracking

Usage:
    from src.observability.vault_metrics import VaultMetrics

    # Record an operation
    with VaultMetrics.track_operation("get_secret"):
        vault.get_secret("path", "key")

    # Or manually
    start = time.time()
    try:
        result = vault.get_secret("path", "key")
        VaultMetrics.record_success("get_secret", time.time() - start)
    except Exception as e:
        VaultMetrics.record_error("get_secret", type(e).__name__)
"""

import logging
import time
from contextlib import contextmanager
from typing import Optional, Generator

try:
    from prometheus_client import Counter, Histogram, Gauge, Info
    PROMETHEUS_AVAILABLE = True
except ImportError:
    PROMETHEUS_AVAILABLE = False

logger = logging.getLogger(__name__)


class VaultMetrics:
    """
    Prometheus metrics for Vault client operations.

    All metrics are prefixed with 'vault_client_' for easy identification.
    """

    _initialized = False

    # Metrics definitions
    _operations_total: Optional["Counter"] = None
    _operation_duration_seconds: Optional["Histogram"] = None
    _errors_total: Optional["Counter"] = None
    _token_ttl_seconds: Optional["Gauge"] = None
    _token_renewals_total: Optional["Counter"] = None
    _rate_limit_wait_seconds: Optional["Histogram"] = None
    _cache_hits_total: Optional["Counter"] = None
    _cache_misses_total: Optional["Counter"] = None
    _client_info: Optional["Info"] = None

    @classmethod
    def initialize(cls) -> bool:
        """
        Initialize Prometheus metrics.

        Returns:
            True if metrics were initialized, False if prometheus_client not available
        """
        if cls._initialized:
            return True

        if not PROMETHEUS_AVAILABLE:
            logger.warning(
                "prometheus_client not installed. Vault metrics disabled. "
                "Install with: pip install prometheus-client"
            )
            return False

        try:
            # Operation counter
            cls._operations_total = Counter(
                "vault_client_operations_total",
                "Total number of Vault operations",
                ["operation", "status"]
            )

            # Operation latency histogram
            cls._operation_duration_seconds = Histogram(
                "vault_client_operation_duration_seconds",
                "Duration of Vault operations in seconds",
                ["operation"],
                buckets=(0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0)
            )

            # Error counter by type
            cls._errors_total = Counter(
                "vault_client_errors_total",
                "Total number of Vault errors by type",
                ["operation", "error_type"]
            )

            # Token TTL gauge
            cls._token_ttl_seconds = Gauge(
                "vault_client_token_ttl_seconds",
                "Current Vault token TTL in seconds"
            )

            # Token renewal counter
            cls._token_renewals_total = Counter(
                "vault_client_token_renewals_total",
                "Total number of token renewals",
                ["status"]
            )

            # Rate limiter wait time
            cls._rate_limit_wait_seconds = Histogram(
                "vault_client_rate_limit_wait_seconds",
                "Time spent waiting for rate limiter in seconds",
                buckets=(0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0)
            )

            # Cache metrics
            cls._cache_hits_total = Counter(
                "vault_client_cache_hits_total",
                "Total number of cache hits"
            )

            cls._cache_misses_total = Counter(
                "vault_client_cache_misses_total",
                "Total number of cache misses"
            )

            # Client info
            cls._client_info = Info(
                "vault_client",
                "Vault client information"
            )

            cls._initialized = True
            logger.info("Vault metrics initialized successfully")
            return True

        except Exception as e:
            logger.error(f"Failed to initialize Vault metrics: {e}")
            return False

    @classmethod
    @contextmanager
    def track_operation(cls, operation: str) -> Generator[None, None, None]:
        """
        Context manager to track operation duration and status.

        Args:
            operation: Name of the operation (e.g., "get_secret", "put_secret")

        Example:
            with VaultMetrics.track_operation("get_secret"):
                vault.get_secret("path", "key")
        """
        if not cls._initialized:
            cls.initialize()

        start_time = time.time()
        error_occurred = False
        error_type = None

        try:
            yield
        except Exception as e:
            error_occurred = True
            error_type = type(e).__name__
            raise
        finally:
            duration = time.time() - start_time

            if cls._initialized and PROMETHEUS_AVAILABLE:
                # Record duration
                if cls._operation_duration_seconds:
                    cls._operation_duration_seconds.labels(operation=operation).observe(duration)

                # Record status
                status = "error" if error_occurred else "success"
                if cls._operations_total:
                    cls._operations_total.labels(operation=operation, status=status).inc()

                # Record error type if applicable
                if error_occurred and error_type and cls._errors_total:
                    cls._errors_total.labels(operation=operation, error_type=error_type).inc()

    @classmethod
    def record_success(cls, operation: str, duration: float) -> None:
        """Record a successful operation."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE:
            if cls._operations_total:
                cls._operations_total.labels(operation=operation, status="success").inc()
            if cls._operation_duration_seconds:
                cls._operation_duration_seconds.labels(operation=operation).observe(duration)

    @classmethod
    def record_error(cls, operation: str, error_type: str) -> None:
        """Record an operation error."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE:
            if cls._operations_total:
                cls._operations_total.labels(operation=operation, status="error").inc()
            if cls._errors_total:
                cls._errors_total.labels(operation=operation, error_type=error_type).inc()

    @classmethod
    def record_token_ttl(cls, ttl_seconds: int) -> None:
        """Record current token TTL."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._token_ttl_seconds:
            cls._token_ttl_seconds.set(ttl_seconds)

    @classmethod
    def record_token_renewal(cls, success: bool) -> None:
        """Record a token renewal attempt."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._token_renewals_total:
            status = "success" if success else "failure"
            cls._token_renewals_total.labels(status=status).inc()

    @classmethod
    def record_rate_limit_wait(cls, wait_seconds: float) -> None:
        """Record time spent waiting for rate limiter."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._rate_limit_wait_seconds:
            cls._rate_limit_wait_seconds.observe(wait_seconds)

    @classmethod
    def record_cache_hit(cls) -> None:
        """Record a cache hit."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._cache_hits_total:
            cls._cache_hits_total.inc()

    @classmethod
    def record_cache_miss(cls) -> None:
        """Record a cache miss."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._cache_misses_total:
            cls._cache_misses_total.inc()

    @classmethod
    def set_client_info(
        cls,
        vault_addr: str,
        namespace: str = "",
        auto_renew: bool = False,
        rate_limit: Optional[float] = None
    ) -> None:
        """Set Vault client information."""
        if not cls._initialized:
            cls.initialize()

        if cls._initialized and PROMETHEUS_AVAILABLE and cls._client_info:
            cls._client_info.info({
                "vault_addr": vault_addr,
                "namespace": namespace or "default",
                "auto_renew": str(auto_renew).lower(),
                "rate_limit": str(rate_limit) if rate_limit else "unlimited"
            })


# Convenience functions for simpler usage
def track_vault_operation(operation: str):
    """Decorator/context manager for tracking Vault operations."""
    return VaultMetrics.track_operation(operation)


def record_vault_success(operation: str, duration: float):
    """Record a successful Vault operation."""
    VaultMetrics.record_success(operation, duration)


def record_vault_error(operation: str, error_type: str):
    """Record a Vault operation error."""
    VaultMetrics.record_error(operation, error_type)
