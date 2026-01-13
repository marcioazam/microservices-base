-- Password Recovery Service Database Initialization
-- Creates the database and tables for the password recovery microservice

-- Create database if not exists
SELECT 'CREATE DATABASE password_recovery_db'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'password_recovery_db')\gexec

\c password_recovery_db;

-- Recovery Tokens Table
CREATE TABLE IF NOT EXISTS recovery_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    token_hash VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    is_used BOOLEAN NOT NULL DEFAULT FALSE,
    used_at TIMESTAMPTZ,
    ip_address INET,
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
    ip_address INET,
    correlation_id UUID NOT NULL,
    event_data JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_user_id ON password_recovery_audit(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created_at ON password_recovery_audit(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_correlation_id ON password_recovery_audit(correlation_id);

-- Grant permissions
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;
