package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/auth-platform/file-upload/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository implements the MetadataStore interface
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new metadata repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// fileRow represents a database row for files
type fileRow struct {
	ID           string         `db:"id"`
	TenantID     string         `db:"tenant_id"`
	UserID       string         `db:"user_id"`
	Filename     string         `db:"filename"`
	OriginalName string         `db:"original_name"`
	MIMEType     string         `db:"mime_type"`
	Size         int64          `db:"size"`
	Hash         string         `db:"hash"`
	StoragePath  string         `db:"storage_path"`
	StorageURL   sql.NullString `db:"storage_url"`
	Status       string         `db:"status"`
	ScanStatus   string         `db:"scan_status"`
	Metadata     []byte         `db:"metadata"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
	DeletedAt    sql.NullTime   `db:"deleted_at"`
}

func (r *fileRow) toDomain() *domain.FileMetadata {
	f := &domain.FileMetadata{
		ID:           r.ID,
		TenantID:     r.TenantID,
		UserID:       r.UserID,
		Filename:     r.Filename,
		OriginalName: r.OriginalName,
		MIMEType:     r.MIMEType,
		Size:         r.Size,
		Hash:         r.Hash,
		StoragePath:  r.StoragePath,
		Status:       domain.FileStatus(r.Status),
		ScanStatus:   domain.ScanStatus(r.ScanStatus),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}

	if r.StorageURL.Valid {
		f.StorageURL = r.StorageURL.String
	}

	if r.DeletedAt.Valid {
		f.DeletedAt = &r.DeletedAt.Time
	}

	if len(r.Metadata) > 0 {
		_ = json.Unmarshal(r.Metadata, &f.Metadata)
	}

	return f
}

// Create stores new file metadata
func (r *Repository) Create(ctx context.Context, f *domain.FileMetadata) error {
	metadata, err := json.Marshal(f.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	query := `
		INSERT INTO files (
			id, tenant_id, user_id, filename, original_name, mime_type,
			size, hash, storage_path, storage_url, status, scan_status, metadata
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	var storageURL sql.NullString
	if f.StorageURL != "" {
		storageURL = sql.NullString{String: f.StorageURL, Valid: true}
	}

	_, err = r.db.ExecContext(ctx, query,
		f.ID, f.TenantID, f.UserID, f.Filename, f.OriginalName, f.MIMEType,
		f.Size, f.Hash, f.StoragePath, storageURL, f.Status, f.ScanStatus, metadata,
	)

	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.NewDomainError(domain.ErrCodeDatabaseError, "file already exists", err)
		}
		return domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to create file", err)
	}

	return nil
}

// GetByID retrieves metadata by ID
func (r *Repository) GetByID(ctx context.Context, id string) (*domain.FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE id = $1`

	var row fileRow
	err := r.db.GetContext(ctx, &row, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrFileNotFound
		}
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to get file", err)
	}

	return row.toDomain(), nil
}

// GetByHash retrieves metadata by hash for deduplication
func (r *Repository) GetByHash(ctx context.Context, tenantID, hash string) (*domain.FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE tenant_id = $1 AND hash = $2 AND deleted_at IS NULL`

	var row fileRow
	err := r.db.GetContext(ctx, &row, query, tenantID, hash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found is not an error for deduplication check
		}
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to get file by hash", err)
	}

	return row.toDomain(), nil
}

// List retrieves files with pagination and filters
func (r *Repository) List(ctx context.Context, req *domain.ListRequest) (*domain.ListResponse, error) {
	// Build query
	baseQuery := `
		FROM files
		WHERE tenant_id = $1 AND deleted_at IS NULL`
	args := []interface{}{req.TenantID}
	argIdx := 2

	if req.Status != nil {
		baseQuery += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, string(*req.Status))
		argIdx++
	}

	if req.StartDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *req.StartDate)
		argIdx++
	}

	if req.EndDate != nil {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *req.EndDate)
		argIdx++
	}

	// Count total
	countQuery := "SELECT COUNT(*) " + baseQuery
	var totalCount int64
	err := r.db.GetContext(ctx, &totalCount, countQuery, args...)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to count files", err)
	}

	// Build order clause
	sortBy := "created_at"
	if req.SortBy != "" {
		sortBy = req.SortBy
	}
	sortOrder := "DESC"
	if req.SortOrder == "asc" {
		sortOrder = "ASC"
	}

	// Pagination
	pageSize := 20
	if req.PageSize > 0 && req.PageSize <= 100 {
		pageSize = req.PageSize
	}

	selectQuery := fmt.Sprintf(`
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		%s
		ORDER BY %s %s
		LIMIT $%d`, baseQuery, sortBy, sortOrder, argIdx)
	args = append(args, pageSize+1) // +1 to check if there's a next page

	var rows []fileRow
	err = r.db.SelectContext(ctx, &rows, selectQuery, args...)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to list files", err)
	}

	// Convert to domain objects
	files := make([]*domain.FileMetadata, 0, len(rows))
	var nextPageToken string

	for i, row := range rows {
		if i >= pageSize {
			// There's a next page
			nextPageToken = row.ID
			break
		}
		files = append(files, row.toDomain())
	}

	return &domain.ListResponse{
		Files:         files,
		NextPageToken: nextPageToken,
		TotalCount:    totalCount,
	}, nil
}

// Search searches files by name or hash
func (r *Repository) Search(ctx context.Context, tenantID, query string) ([]*domain.FileMetadata, error) {
	sqlQuery := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE tenant_id = $1 
		  AND deleted_at IS NULL
		  AND (filename ILIKE $2 OR original_name ILIKE $2 OR hash = $3)
		ORDER BY created_at DESC
		LIMIT 50`

	searchPattern := "%" + query + "%"
	var rows []fileRow
	err := r.db.SelectContext(ctx, &rows, sqlQuery, tenantID, searchPattern, query)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to search files", err)
	}

	files := make([]*domain.FileMetadata, len(rows))
	for i, row := range rows {
		files[i] = row.toDomain()
	}

	return files, nil
}

// Update updates file metadata
func (r *Repository) Update(ctx context.Context, f *domain.FileMetadata) error {
	metadata, err := json.Marshal(f.Metadata)
	if err != nil {
		metadata = []byte("{}")
	}

	query := `
		UPDATE files SET
			filename = $2,
			storage_url = $3,
			status = $4,
			scan_status = $5,
			metadata = $6,
			updated_at = NOW()
		WHERE id = $1`

	var storageURL sql.NullString
	if f.StorageURL != "" {
		storageURL = sql.NullString{String: f.StorageURL, Valid: true}
	}

	result, err := r.db.ExecContext(ctx, query,
		f.ID, f.Filename, storageURL, f.Status, f.ScanStatus, metadata,
	)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to update file", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrFileNotFound
	}

	return nil
}

// SoftDelete marks file as deleted
func (r *Repository) SoftDelete(ctx context.Context, id string) error {
	query := `
		UPDATE files SET
			status = $2,
			deleted_at = NOW(),
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.ExecContext(ctx, query, id, domain.FileStatusDeleted)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to delete file", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrFileNotFound
	}

	return nil
}

// HardDelete permanently removes metadata
func (r *Repository) HardDelete(ctx context.Context, id string) error {
	query := `DELETE FROM files WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to hard delete file", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrFileNotFound
	}

	return nil
}

// GetExpiredDeleted retrieves files past retention period
func (r *Repository) GetExpiredDeleted(ctx context.Context, before time.Time) ([]*domain.FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE deleted_at IS NOT NULL AND deleted_at < $1
		LIMIT 100`

	var rows []fileRow
	err := r.db.SelectContext(ctx, &rows, query, before)
	if err != nil {
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to get expired files", err)
	}

	files := make([]*domain.FileMetadata, len(rows))
	for i, row := range rows {
		files[i] = row.toDomain()
	}

	return files, nil
}

// GetByTenantAndID retrieves file ensuring tenant isolation
func (r *Repository) GetByTenantAndID(ctx context.Context, tenantID, id string) (*domain.FileMetadata, error) {
	query := `
		SELECT id, tenant_id, user_id, filename, original_name, mime_type,
			   size, hash, storage_path, storage_url, status, scan_status,
			   metadata, created_at, updated_at, deleted_at
		FROM files
		WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL`

	var row fileRow
	err := r.db.GetContext(ctx, &row, query, id, tenantID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrFileNotFound
		}
		return nil, domain.NewDomainError(domain.ErrCodeDatabaseError, "failed to get file", err)
	}

	return row.toDomain(), nil
}
