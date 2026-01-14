// Package repository provides database access implementations.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// FileStatus represents the lifecycle status of a file.
type FileStatus string

const (
	FileStatusPending    FileStatus = "pending"
	FileStatusUploaded   FileStatus = "uploaded"
	FileStatusProcessing FileStatus = "processing"
	FileStatusReady      FileStatus = "ready"
	FileStatusFailed     FileStatus = "failed"
	FileStatusDeleted    FileStatus = "deleted"
)

// ScanStatus represents the malware scan status.
type ScanStatus string

const (
	ScanStatusPending  ScanStatus = "pending"
	ScanStatusScanning ScanStatus = "scanning"
	ScanStatusClean    ScanStatus = "clean"
	ScanStatusInfected ScanStatus = "infected"
	ScanStatusFailed   ScanStatus = "failed"
)

// FileMetadata represents stored file information.
type FileMetadata struct {
	ID           string            `db:"id"`
	TenantID     string            `db:"tenant_id"`
	UserID       string            `db:"user_id"`
	Filename     string            `db:"filename"`
	OriginalName string            `db:"original_name"`
	MIMEType     string            `db:"mime_type"`
	Size         int64             `db:"size"`
	Hash         string            `db:"hash"`
	StoragePath  string            `db:"storage_path"`
	StorageURL   string            `db:"storage_url"`
	Status       FileStatus        `db:"status"`
	ScanStatus   ScanStatus        `db:"scan_status"`
	Metadata     map[string]string `db:"-"`
	MetadataJSON sql.NullString    `db:"metadata"`
	CreatedAt    time.Time         `db:"created_at"`
	UpdatedAt    time.Time         `db:"updated_at"`
	DeletedAt    *time.Time        `db:"deleted_at"`
}

// ListRequest represents a list query request.
type ListRequest struct {
	TenantID  string
	PageSize  int
	Cursor    string
	Status    *FileStatus
	StartDate *time.Time
	EndDate   *time.Time
}

// ListResponse represents a paginated list response.
type ListResponse struct {
	Files      []*FileMetadata
	NextCursor string
	TotalCount int64
}

// MetadataRepository defines the contract for file metadata storage.
type MetadataRepository interface {
	Create(ctx context.Context, file *FileMetadata) error
	GetByID(ctx context.Context, id string) (*FileMetadata, error)
	GetByHash(ctx context.Context, tenantID, hash string) (*FileMetadata, error)
	List(ctx context.Context, req *ListRequest) (*ListResponse, error)
	Update(ctx context.Context, file *FileMetadata) error
	SoftDelete(ctx context.Context, id string) error
	UpdateScanStatus(ctx context.Context, id string, status ScanStatus) error
}

// PostgresMetadataRepository implements MetadataRepository using PostgreSQL.
type PostgresMetadataRepository struct {
	db *sqlx.DB
}

// NewPostgresMetadataRepository creates a new PostgreSQL metadata repository.
func NewPostgresMetadataRepository(db *sqlx.DB) *PostgresMetadataRepository {
	return &PostgresMetadataRepository{db: db}
}

// Create inserts a new file metadata record.
func (r *PostgresMetadataRepository) Create(ctx context.Context, file *FileMetadata) error {
	if file.Metadata != nil {
		metadataJSON, err := json.Marshal(file.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		file.MetadataJSON = sql.NullString{String: string(metadataJSON), Valid: true}
	}

	query := `
		INSERT INTO files (
			id, tenant_id, user_id, filename, original_name, mime_type,
			size, hash, storage_path, storage_url, status, scan_status,
			metadata, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :user_id, :filename, :original_name, :mime_type,
			:size, :hash, :storage_path, :storage_url, :status, :scan_status,
			:metadata, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, file)
	if err != nil {
		return fmt.Errorf("failed to create file metadata: %w", err)
	}
	return nil
}

// GetByID retrieves file metadata by ID.
func (r *PostgresMetadataRepository) GetByID(ctx context.Context, id string) (*FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE id = $1`

	var file FileMetadata
	if err := r.db.GetContext(ctx, &file, query, id); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get file by ID: %w", err)
	}

	if err := r.unmarshalMetadata(&file); err != nil {
		return nil, err
	}
	return &file, nil
}

// GetByHash retrieves file metadata by tenant ID and hash.
func (r *PostgresMetadataRepository) GetByHash(ctx context.Context, tenantID, hash string) (*FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE tenant_id = $1 AND hash = $2 AND deleted_at IS NULL`

	var file FileMetadata
	if err := r.db.GetContext(ctx, &file, query, tenantID, hash); err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get file by hash: %w", err)
	}

	if err := r.unmarshalMetadata(&file); err != nil {
		return nil, err
	}
	return &file, nil
}

// List retrieves paginated file metadata.
func (r *PostgresMetadataRepository) List(ctx context.Context, req *ListRequest) (*ListResponse, error) {
	// Decode cursor if provided
	var cursorID string
	var cursorTime time.Time
	if req.Cursor != "" {
		cursor, err := DecodeCursor(req.Cursor)
		if err != nil {
			return nil, fmt.Errorf("invalid cursor: %w", err)
		}
		cursorID = cursor.ID
		cursorTime = cursor.CreatedAt
	}

	// Build query with cursor-based pagination
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	// Fetch one extra to determine if there are more results
	query := r.buildListQuery(req, cursorID, cursorTime, pageSize+1)
	args := r.buildListArgs(req, cursorID, cursorTime, pageSize+1)

	var files []*FileMetadata
	if err := r.db.SelectContext(ctx, &files, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Unmarshal metadata for each file
	for _, file := range files {
		if err := r.unmarshalMetadata(file); err != nil {
			return nil, err
		}
	}

	// Determine if there are more results
	var nextCursor string
	if len(files) > pageSize {
		files = files[:pageSize]
		lastFile := files[len(files)-1]
		nextCursor = EncodeCursor(Cursor{ID: lastFile.ID, CreatedAt: lastFile.CreatedAt})
	}

	// Get total count
	totalCount, err := r.countFiles(ctx, req)
	if err != nil {
		return nil, err
	}

	return &ListResponse{
		Files:      files,
		NextCursor: nextCursor,
		TotalCount: totalCount,
	}, nil
}

func (r *PostgresMetadataRepository) buildListQuery(req *ListRequest, cursorID string, cursorTime time.Time, limit int) string {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE tenant_id = $1 AND deleted_at IS NULL`

	argNum := 2
	if cursorID != "" {
		query += fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argNum, argNum+1)
		argNum += 2
	}
	if req.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		argNum++
	}
	if req.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		argNum++
	}
	if req.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC, id DESC LIMIT $%d", argNum)
	return query
}

func (r *PostgresMetadataRepository) buildListArgs(req *ListRequest, cursorID string, cursorTime time.Time, limit int) []any {
	args := []any{req.TenantID}
	if cursorID != "" {
		args = append(args, cursorTime, cursorID)
	}
	if req.Status != nil {
		args = append(args, *req.Status)
	}
	if req.StartDate != nil {
		args = append(args, *req.StartDate)
	}
	if req.EndDate != nil {
		args = append(args, *req.EndDate)
	}
	args = append(args, limit)
	return args
}

func (r *PostgresMetadataRepository) countFiles(ctx context.Context, req *ListRequest) (int64, error) {
	query := `SELECT COUNT(*) FROM files WHERE tenant_id = $1 AND deleted_at IS NULL`
	args := []any{req.TenantID}
	argNum := 2

	if req.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *req.Status)
		argNum++
	}
	if req.StartDate != nil {
		query += fmt.Sprintf(" AND created_at >= $%d", argNum)
		args = append(args, *req.StartDate)
		argNum++
	}
	if req.EndDate != nil {
		query += fmt.Sprintf(" AND created_at <= $%d", argNum)
		args = append(args, *req.EndDate)
	}

	var count int64
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return 0, fmt.Errorf("failed to count files: %w", err)
	}
	return count, nil
}

// Update updates file metadata.
func (r *PostgresMetadataRepository) Update(ctx context.Context, file *FileMetadata) error {
	if file.Metadata != nil {
		metadataJSON, err := json.Marshal(file.Metadata)
		if err != nil {
			return fmt.Errorf("failed to marshal metadata: %w", err)
		}
		file.MetadataJSON = sql.NullString{String: string(metadataJSON), Valid: true}
	}

	file.UpdatedAt = time.Now().UTC()

	query := `
		UPDATE files SET
			filename = :filename,
			original_name = :original_name,
			mime_type = :mime_type,
			size = :size,
			storage_path = :storage_path,
			storage_url = :storage_url,
			status = :status,
			scan_status = :scan_status,
			metadata = :metadata,
			updated_at = :updated_at
		WHERE id = :id AND deleted_at IS NULL`

	result, err := r.db.NamedExecContext(ctx, query, file)
	if err != nil {
		return fmt.Errorf("failed to update file metadata: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// SoftDelete marks a file as deleted.
func (r *PostgresMetadataRepository) SoftDelete(ctx context.Context, id string) error {
	now := time.Now().UTC()
	query := `
		UPDATE files SET
			deleted_at = $1,
			updated_at = $1,
			status = $2
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, now, FileStatusDeleted, id)
	if err != nil {
		return fmt.Errorf("failed to soft delete file: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// UpdateScanStatus updates the scan status of a file.
func (r *PostgresMetadataRepository) UpdateScanStatus(ctx context.Context, id string, status ScanStatus) error {
	query := `
		UPDATE files SET
			scan_status = $1,
			updated_at = $2
		WHERE id = $3 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("failed to update scan status: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresMetadataRepository) unmarshalMetadata(file *FileMetadata) error {
	if file.MetadataJSON.Valid && file.MetadataJSON.String != "" {
		if err := json.Unmarshal([]byte(file.MetadataJSON.String), &file.Metadata); err != nil {
			return fmt.Errorf("failed to unmarshal metadata: %w", err)
		}
	}
	return nil
}

// Errors
var (
	ErrNotFound = &RepositoryError{Code: "NOT_FOUND", Message: "record not found"}
)

// RepositoryError represents a repository operation error.
type RepositoryError struct {
	Code    string
	Message string
}

func (e *RepositoryError) Error() string {
	return e.Code + ": " + e.Message
}
