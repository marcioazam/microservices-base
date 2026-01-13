-- Drop chunk_sessions table and indexes
DROP INDEX IF EXISTS idx_chunk_sessions_tenant_status;
DROP INDEX IF EXISTS idx_chunk_sessions_status;
DROP INDEX IF EXISTS idx_chunk_sessions_expires;
DROP INDEX IF EXISTS idx_chunk_sessions_user;
DROP INDEX IF EXISTS idx_chunk_sessions_tenant;
DROP TABLE IF EXISTS chunk_sessions;
