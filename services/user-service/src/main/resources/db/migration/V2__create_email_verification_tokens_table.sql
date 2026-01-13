-- V2: Create email verification tokens table
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT uk_tokens_hash UNIQUE (token_hash)
);

CREATE INDEX idx_tokens_user_id ON email_verification_tokens(user_id);
CREATE INDEX idx_tokens_expires_at ON email_verification_tokens(expires_at);
CREATE INDEX idx_tokens_unused ON email_verification_tokens(user_id) WHERE used_at IS NULL;

COMMENT ON TABLE email_verification_tokens IS 'Email verification tokens with SHA-256 hash storage';
COMMENT ON COLUMN email_verification_tokens.token_hash IS 'SHA-256 hash of the verification token (64 hex chars)';
COMMENT ON COLUMN email_verification_tokens.used_at IS 'Timestamp when token was used, NULL if unused';
