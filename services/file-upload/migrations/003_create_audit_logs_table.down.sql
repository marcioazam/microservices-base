-- Drop audit_logs table and indexes
DROP INDEX IF EXISTS idx_audit_logs_request_id;
DROP INDEX IF EXISTS idx_audit_logs_tenant_created;
DROP INDEX IF EXISTS idx_audit_logs_created;
DROP INDEX IF EXISTS idx_audit_logs_operation;
DROP INDEX IF EXISTS idx_audit_logs_file;
DROP INDEX IF EXISTS idx_audit_logs_user;
DROP INDEX IF EXISTS idx_audit_logs_tenant;
DROP TABLE IF EXISTS audit_logs;
