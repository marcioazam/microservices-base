"""OpenTelemetry integration for Auth Platform SDK - December 2025 State of Art.

Provides tracing, metrics, and structured logging for observability.
"""

from __future__ import annotations

import functools
from contextlib import contextmanager
from typing import TYPE_CHECKING, Any, Callable, ParamSpec, TypeVar

import structlog
from opentelemetry import trace
from opentelemetry.trace import Status, StatusCode

if TYPE_CHECKING:
    from collections.abc import Generator

    from .config import TelemetryConfig

P = ParamSpec("P")
T = TypeVar("T")

# Module-level tracer and logger
_tracer: trace.Tracer | None = None
_logger: structlog.BoundLogger | None = None


def get_tracer() -> trace.Tracer:
    """Get or create the SDK tracer."""
    global _tracer
    if _tracer is None:
        _tracer = trace.get_tracer("auth-platform-sdk", "1.0.0")
    return _tracer


def get_logger() -> structlog.BoundLogger:
    """Get or create the SDK logger."""
    global _logger
    if _logger is None:
        _logger = structlog.get_logger("auth-platform-sdk")
    return _logger


def configure_telemetry(config: TelemetryConfig) -> None:
    """Configure telemetry based on config.

    Args:
        config: Telemetry configuration.
    """
    global _tracer, _logger

    if not config.enabled:
        _tracer = trace.NoOpTracer()
        return

    # Configure structlog
    structlog.configure(
        processors=[
            structlog.contextvars.merge_contextvars,
            structlog.processors.add_log_level,
            structlog.processors.TimeStamper(fmt="iso"),
            structlog.processors.StackInfoRenderer(),
            structlog.processors.format_exc_info,
            structlog.processors.JSONRenderer(),
        ],
        wrapper_class=structlog.make_filtering_bound_logger(
            _log_level_to_int(config.log_level)
        ),
        context_class=dict,
        logger_factory=structlog.PrintLoggerFactory(),
        cache_logger_on_first_use=True,
    )

    _tracer = trace.get_tracer(config.service_name, "1.0.0")
    _logger = structlog.get_logger(config.service_name)


def _log_level_to_int(level: str) -> int:
    """Convert log level string to integer."""
    levels = {
        "DEBUG": 10,
        "INFO": 20,
        "WARNING": 30,
        "ERROR": 40,
        "CRITICAL": 50,
    }
    return levels.get(level.upper(), 20)


@contextmanager
def trace_operation(
    name: str,
    *,
    attributes: dict[str, Any] | None = None,
) -> Generator[trace.Span, None, None]:
    """Context manager for tracing an operation.

    Args:
        name: Name of the operation.
        attributes: Optional span attributes.

    Yields:
        The active span.
    """
    tracer = get_tracer()
    with tracer.start_as_current_span(name) as span:
        if attributes:
            for key, value in attributes.items():
                span.set_attribute(key, value)
        try:
            yield span
        except Exception as e:
            span.set_status(Status(StatusCode.ERROR, str(e)))
            span.record_exception(e)
            raise


def traced(
    name: str | None = None,
    *,
    record_args: bool = False,
) -> Callable[[Callable[P, T]], Callable[P, T]]:
    """Decorator to trace a function.

    Args:
        name: Optional span name (defaults to function name).
        record_args: Whether to record function arguments as attributes.

    Returns:
        Decorated function.
    """

    def decorator(func: Callable[P, T]) -> Callable[P, T]:
        span_name = name or func.__name__

        @functools.wraps(func)
        def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
            attributes: dict[str, Any] = {}
            if record_args:
                # Only record safe, serializable arguments
                for i, arg in enumerate(args):
                    if isinstance(arg, (str, int, float, bool)):
                        attributes[f"arg_{i}"] = arg
                for key, value in kwargs.items():
                    if isinstance(value, (str, int, float, bool)):
                        attributes[f"kwarg_{key}"] = value

            with trace_operation(span_name, attributes=attributes):
                return func(*args, **kwargs)

        return wrapper

    return decorator


def traced_async(
    name: str | None = None,
    *,
    record_args: bool = False,
) -> Callable[[Callable[P, T]], Callable[P, T]]:
    """Decorator to trace an async function.

    Args:
        name: Optional span name (defaults to function name).
        record_args: Whether to record function arguments as attributes.

    Returns:
        Decorated async function.
    """

    def decorator(func: Callable[P, T]) -> Callable[P, T]:
        span_name = name or func.__name__

        @functools.wraps(func)
        async def wrapper(*args: P.args, **kwargs: P.kwargs) -> T:
            attributes: dict[str, Any] = {}
            if record_args:
                for i, arg in enumerate(args):
                    if isinstance(arg, (str, int, float, bool)):
                        attributes[f"arg_{i}"] = arg
                for key, value in kwargs.items():
                    if isinstance(value, (str, int, float, bool)):
                        attributes[f"kwarg_{key}"] = value

            with trace_operation(span_name, attributes=attributes):
                return await func(*args, **kwargs)  # type: ignore[misc]

        return wrapper  # type: ignore[return-value]

    return decorator


class SDKLogger:
    """Structured logger for SDK operations."""

    def __init__(self, name: str = "auth-platform-sdk") -> None:
        self._logger = structlog.get_logger(name)

    def debug(self, message: str, **kwargs: Any) -> None:
        """Log debug message."""
        self._logger.debug(message, **kwargs)

    def info(self, message: str, **kwargs: Any) -> None:
        """Log info message."""
        self._logger.info(message, **kwargs)

    def warning(self, message: str, **kwargs: Any) -> None:
        """Log warning message."""
        self._logger.warning(message, **kwargs)

    def error(self, message: str, **kwargs: Any) -> None:
        """Log error message."""
        self._logger.error(message, **kwargs)

    def bind(self, **kwargs: Any) -> "SDKLogger":
        """Create a new logger with bound context."""
        new_logger = SDKLogger.__new__(SDKLogger)
        new_logger._logger = self._logger.bind(**kwargs)
        return new_logger
