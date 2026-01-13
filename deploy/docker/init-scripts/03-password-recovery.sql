-- Password Recovery Service Database Schema
-- Migration: Initial schema for recovery_tokens and password_recovery_audit tables

-- Recovery Tokens Table
CREATE TABLE IF NOT EXISTS recovery_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    ip_address VARCHAR(45),
    CONSTRAINT uk_token_hash UNIQUE (token_hash)
);

CREATE INDEX IF NOT EXISTS idx_recovery_tokens_user_id ON recovery_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_recovery_tokens_expires_at ON recovery_tokens(expires_at) WHERE NOT is_used;
CREATE INDEX IF NOT EXISTS idx_recovery_tokens_token_hash ON recovery_tokens(token_hash);

-- Audit Log Table
CREATE TABLE IF NOT EXISTS password_recovery_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(50) NOT NULL,
    user_id UUID,
    email VARCHAR(255),
    ip_address VARCHAR(45),
    correlation_id UUID NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_user_id ON password_recovery_audit(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON password_recovery_audit(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_correlation_id ON password_recovery_audit(correlation_id);
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON password_recovery_audit(event_type);

-- Comments for documentation
COMMENT ON TABLE recovery_tokens IS 'Stores hashed password recovery tokens';
COMMENT ON COLUMN recovery_tokens.token_hash IS 'SHA-256 hash of the recovery token';
COMMENT ON COLUMN recovery_tokens.is_used IS 'Flag indicating if token has been used';
COMMENT ON COLUMN recovery_tokens.ip_address IS 'IP address that requested the token';

COMMENT ON TABLE password_recovery_audit IS 'Audit log for password recovery operations';
COMMENT ON COLUMN password_recovery_audit.event_type IS 'Type of event: RECOVERY_REQUESTED, TOKEN_VALIDATED, PASSWORD_CHANGED';
COMMENT ON COLUMN password_recovery_audit.correlation_id IS 'Correlation ID for request tracing';
