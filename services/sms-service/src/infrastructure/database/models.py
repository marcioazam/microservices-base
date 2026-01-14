"""SQLAlchemy models for SMS service."""

from datetime import datetime
from typing import Any
from uuid import uuid4

from sqlalchemy import DateTime, Index, String, Text, func
from sqlalchemy.dialects.postgresql import JSONB, UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    """Base class for all models."""

    type_annotation_map = {
        dict[str, Any]: JSONB,
    }


class SMSRequest(Base):
    """SMS request model - tracks all SMS send requests."""

    __tablename__ = "sms_requests"

    id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid4
    )
    idempotency_key: Mapped[str] = mapped_column(
        String(64), unique=True, nullable=False, index=True
    )
    to_number: Mapped[str] = mapped_column(String(20), nullable=False)
    message_hash: Mapped[str] = mapped_column(String(64), nullable=False)
    sender_id: Mapped[str | None] = mapped_column(String(20), nullable=True)
    metadata: Mapped[dict[str, Any] | None] = mapped_column(JSONB, nullable=True)
    provider: Mapped[str | None] = mapped_column(String(50), nullable=True)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="accepted")
    error_code: Mapped[str | None] = mapped_column(String(50), nullable=True)
    error_detail: Mapped[str | None] = mapped_column(Text, nullable=True)
    provider_message_id: Mapped[str | None] = mapped_column(String(100), nullable=True)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now(), onupdate=func.now()
    )

    __table_args__ = (
        Index("idx_sms_requests_to_created", "to_number", "created_at"),
        Index("idx_sms_requests_status", "status"),
        Index("idx_sms_requests_provider_msg", "provider_message_id"),
    )


class SMSEvent(Base):
    """SMS event model - event sourcing for status changes."""

    __tablename__ = "sms_events"

    id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid4
    )
    request_id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), nullable=False, index=True
    )
    event_type: Mapped[str] = mapped_column(String(50), nullable=False)
    payload: Mapped[dict[str, Any]] = mapped_column(JSONB, nullable=False)
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )

    __table_args__ = (
        Index("idx_sms_events_request_created", "request_id", "created_at"),
    )


class OTPChallenge(Base):
    """OTP challenge model - tracks OTP generation and validation."""

    __tablename__ = "otp_challenges"

    id: Mapped[UUID] = mapped_column(
        UUID(as_uuid=True), primary_key=True, default=uuid4
    )
    to_number: Mapped[str] = mapped_column(String(20), nullable=False)
    otp_hash: Mapped[str] = mapped_column(String(100), nullable=False)
    sms_request_id: Mapped[UUID | None] = mapped_column(
        UUID(as_uuid=True), nullable=True
    )
    ttl_expires_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False
    )
    attempts: Mapped[int] = mapped_column(default=0, nullable=False)
    max_attempts: Mapped[int] = mapped_column(default=3, nullable=False)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="pending")
    created_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now()
    )
    updated_at: Mapped[datetime] = mapped_column(
        DateTime(timezone=True), nullable=False, server_default=func.now(), onupdate=func.now()
    )

    __table_args__ = (
        Index("idx_otp_challenges_to_created", "to_number", "created_at"),
        Index("idx_otp_challenges_status", "status"),
    )
