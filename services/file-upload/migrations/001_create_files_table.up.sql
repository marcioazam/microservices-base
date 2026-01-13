-- Create files table for storing file metadata
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    mime_type VARCHAR(128) NOT NULL,
    size BIGINT NOT NULL,
    hash VARCHAR(64) NOT NULL,
    storage_path VARCHAR(512) NOT NULL,
    storage_url VARCHAR(1024),
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    scan_status VARCHAR(32) NOT NULL DEFAULT 'pending',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    CONSTRAINT unique_tenant_hash UNIQUE (tenant_id, hash)
);

-- Indexes for common queries
CREATE INDEX idx_files_tenant_id ON files(tenant_id);
CREATE INDEX idx_files_user_id ON files(user_id);
CREATE INDEX idx_files_hash ON files(hash);
CREATE INDEX idx_files_status ON files(status);
CREATE INDEX idx_files_scan_status ON files(scan_status);
CREATE INDEX idx_files_created_at ON files(created_at);
CREATE INDEX idx_files_deleted_at ON files(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX idx_files_tenant_created ON files(tenant_id, created_at DESC);
CREATE INDEX idx_files_filename_search ON files USING gin(filename gin_trgm_ops);

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_files_updated_at
    BEFORE UPDATE ON files
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
