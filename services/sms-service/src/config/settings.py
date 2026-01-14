"""Application settings using Pydantic Settings."""

from functools import lru_cache
from typing import Literal

from pydantic import Field, PostgresDsn, RedisDsn
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    """Application configuration loaded from environment variables."""

    model_config = SettingsConfigDict(
        env_file=".env",
        env_file_encoding="utf-8",
        case_sensitive=False,
        extra="ignore",
    )

    # Application
    app_name: str = "sms-service"
    app_version: str = "1.0.0"
    environment: Literal["development", "staging", "production"] = "development"
    debug: bool = False
    log_level: str = "INFO"

    # Server
    host: str = "0.0.0.0"
    port: int = 8000

    # Database
    database_url: PostgresDsn = Field(
        default="postgresql+asyncpg://postgres:postgres@localhost:5432/sms_service"
    )
    database_pool_size: int = 10
    database_max_overflow: int = 20

    # RabbitMQ
    rabbitmq_url: str = "amqp://guest:guest@localhost:5672/"
    rabbitmq_exchange: str = "sms.direct"
    rabbitmq_queue: str = "sms.send"
    rabbitmq_dlq: str = "sms.dlq"
    rabbitmq_prefetch_count: int = 10

    # Redis
    redis_url: RedisDsn = Field(default="redis://localhost:6379/0")
    redis_pool_size: int = 10

    # Rate Limiting
    rate_limit_per_phone: int = 5  # requests per window
    rate_limit_per_client: int = 100  # requests per window
    rate_limit_window_seconds: int = 60

    # OTP
    otp_length: int = 6
    otp_ttl_seconds: int = 300  # 5 minutes
    otp_max_attempts: int = 3
    otp_rate_limit_per_phone: int = 3  # OTPs per window
    otp_rate_limit_window_seconds: int = 3600  # 1 hour

    # SMS
    sms_max_message_length: int = 1600

    # Providers
    primary_provider: Literal["twilio", "messagebird"] = "twilio"
    fallback_provider: Literal["twilio", "messagebird"] = "messagebird"

    # Twilio
    twilio_account_sid: str = ""
    twilio_auth_token: str = ""
    twilio_from_number: str = ""
    twilio_webhook_secret: str = ""

    # MessageBird
    messagebird_api_key: str = ""
    messagebird_originator: str = ""
    messagebird_webhook_secret: str = ""

    # Circuit Breaker
    circuit_breaker_failure_threshold: int = 5
    circuit_breaker_recovery_timeout: int = 30  # seconds

    # Retry
    retry_base_delay: float = 2.0  # seconds
    retry_max_attempts: int = 5
    retry_max_delay: float = 60.0  # seconds

    # JWT
    jwt_secret_key: str = "change-me-in-production"
    jwt_algorithm: str = "HS256"
    jwt_issuer: str = "auth-service"

    # OpenTelemetry
    otel_enabled: bool = True
    otel_service_name: str = "sms-service"
    otel_exporter_endpoint: str = "http://localhost:4317"

    # Prometheus
    metrics_enabled: bool = True
    metrics_port: int = 9090


@lru_cache
def get_settings() -> Settings:
    """Get cached settings instance."""
    return Settings()
