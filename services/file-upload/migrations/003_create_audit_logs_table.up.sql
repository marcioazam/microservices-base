-- Create audit_logs table for tracking file operations
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    file_id UUID,
    operation VARCHAR(32) NOT NULL,
    filename VARCHAR(255),
    file_size BIGINT,
    file_hash VARCHAR(64),
    source_ip VARCHAR(45),
    user_agent VARCHAR(512),
    request_id VARCHAR(64),
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Indexes for audit logs
CREATE INDEX idx_audit_logs_tenant ON audit_logs(tenant_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_file ON audit_logs(file_id);
CREATE INDEX idx_audit_logs_operation ON audit_logs(operation);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);
CREATE INDEX idx_audit_logs_tenant_created ON audit_logs(tenant_id, created_at DESC);
CREATE INDEX idx_audit_logs_request_id ON audit_logs(request_id);

-- Partition by month for better performance (optional, for high-volume scenarios)
-- This is a comment showing how to implement partitioning if needed:
-- CREATE TABLE audit_logs_partitioned (
--     LIKE audit_logs INCLUDING ALL
-- ) PARTITION BY RANGE (created_at);
