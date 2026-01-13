-- Create chunk_sessions table for managing chunked uploads
CREATE TABLE IF NOT EXISTS chunk_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    total_size BIGINT NOT NULL,
    chunk_size BIGINT NOT NULL,
    total_chunks INT NOT NULL,
    uploaded_chunks INT[] DEFAULT '{}',
    status VARCHAR(32) NOT NULL DEFAULT 'active',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for chunk sessions
CREATE INDEX idx_chunk_sessions_tenant ON chunk_sessions(tenant_id);
CREATE INDEX idx_chunk_sessions_user ON chunk_sessions(user_id);
CREATE INDEX idx_chunk_sessions_expires ON chunk_sessions(expires_at);
CREATE INDEX idx_chunk_sessions_status ON chunk_sessions(status);
CREATE INDEX idx_chunk_sessions_tenant_status ON chunk_sessions(tenant_id, status);
