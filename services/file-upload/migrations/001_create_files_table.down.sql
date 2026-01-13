-- Drop files table and related objects
DROP TRIGGER IF EXISTS update_files_updated_at ON files;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_files_filename_search;
DROP INDEX IF EXISTS idx_files_tenant_created;
DROP INDEX IF EXISTS idx_files_deleted_at;
DROP INDEX IF EXISTS idx_files_created_at;
DROP INDEX IF EXISTS idx_files_scan_status;
DROP INDEX IF EXISTS idx_files_status;
DROP INDEX IF EXISTS idx_files_hash;
DROP INDEX IF EXISTS idx_files_user_id;
DROP INDEX IF EXISTS idx_files_tenant_id;
DROP TABLE IF EXISTS files;
