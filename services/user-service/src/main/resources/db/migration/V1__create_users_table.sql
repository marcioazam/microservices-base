-- V1: Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) NOT NULL,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    password_hash VARCHAR(255) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'PENDING_EMAIL',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    CONSTRAINT uk_users_email UNIQUE (email),
    CONSTRAINT chk_users_status CHECK (status IN ('PENDING_EMAIL', 'ACTIVE', 'DISABLED'))
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_created_at ON users(created_at);

COMMENT ON TABLE users IS 'User accounts for the Auth Platform';
COMMENT ON COLUMN users.email IS 'Normalized email address (lowercase, trimmed)';
COMMENT ON COLUMN users.password_hash IS 'Argon2id password hash';
COMMENT ON COLUMN users.status IS 'Account status: PENDING_EMAIL, ACTIVE, DISABLED';
