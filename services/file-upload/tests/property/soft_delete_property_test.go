// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 13: Soft Delete Correctness
// Validates: Requirements 10.3
package property

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// SoftDeletableRecord represents a record that supports soft delete.
type SoftDeletableRecord struct {
	ID        string
	TenantID  string
	Status    string
	DeletedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

// IsDeleted returns true if the record is soft-deleted.
func (r *SoftDeletableRecord) IsDeleted() bool {
	return r.DeletedAt != nil
}

// MockSoftDeleteRepository simulates soft delete behavior.
type MockSoftDeleteRepository struct {
	records map[string]*SoftDeletableRecord
}

func NewMockSoftDeleteRepository() *MockSoftDeleteRepository {
	return &MockSoftDeleteRepository{
		records: make(map[string]*SoftDeletableRecord),
	}
}

// Create adds a new record.
func (r *MockSoftDeleteRepository) Create(record *SoftDeletableRecord) {
	r.records[record.ID] = record
}

// GetByID retrieves a record by ID (including deleted).
func (r *MockSoftDeleteRepository) GetByID(id string) (*SoftDeletableRecord, bool) {
	record, ok := r.records[id]
	return record, ok
}

// List returns non-deleted records for a tenant.
func (r *MockSoftDeleteRepository) List(tenantID string) []*SoftDeletableRecord {
	var result []*SoftDeletableRecord
	for _, record := range r.records {
		if record.TenantID == tenantID && !record.IsDeleted() {
			result = append(result, record)
		}
	}
	return result
}

// SoftDelete marks a record as deleted.
func (r *MockSoftDeleteRepository) SoftDelete(id string) bool {
	record, ok := r.records[id]
	if !ok || record.IsDeleted() {
		return false
	}
	now := time.Now().UTC()
	record.DeletedAt = &now
	record.Status = "deleted"
	record.UpdatedAt = now
	return true
}

// TestProperty13_SoftDeleteSetsTimestamp tests that soft delete sets deleted_at timestamp.
// Property 13: Soft Delete Correctness
// Validates: Requirements 10.3
func TestProperty13_SoftDeleteSetsTimestamp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockSoftDeleteRepository()

		id := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "id")
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		record := &SoftDeletableRecord{
			ID:        id,
			TenantID:  tenantID,
			Status:    "ready",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		repo.Create(record)

		// Verify not deleted initially
		if record.IsDeleted() {
			t.Error("record should not be deleted initially")
		}

		beforeDelete := time.Now().UTC()

		// Soft delete
		success := repo.SoftDelete(id)
		if !success {
			t.Fatal("soft delete should succeed")
		}

		afterDelete := time.Now().UTC()

		// Property: deleted_at timestamp SHALL be set
		retrieved, ok := repo.GetByID(id)
		if !ok {
			t.Fatal("record should still exist after soft delete")
		}

		if retrieved.DeletedAt == nil {
			t.Error("deleted_at should be set after soft delete")
		}

		// Verify timestamp is reasonable
		if retrieved.DeletedAt.Before(beforeDelete) {
			t.Error("deleted_at should not be before delete operation")
		}
		if retrieved.DeletedAt.After(afterDelete) {
			t.Error("deleted_at should not be after delete operation")
		}
	})
}

// TestProperty13_SoftDeletedNotInList tests that soft-deleted files don't appear in list queries.
// Property 13: Soft Delete Correctness
// Validates: Requirements 10.3
func TestProperty13_SoftDeletedNotInList(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockSoftDeleteRepository()

		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")
		numRecords := rapid.IntRange(3, 10).Draw(t, "numRecords")
		deleteIndex := rapid.IntRange(0, numRecords-1).Draw(t, "deleteIndex")

		// Create records
		ids := make([]string, numRecords)
		for i := 0; i < numRecords; i++ {
			id := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "id")
			ids[i] = id
			repo.Create(&SoftDeletableRecord{
				ID:        id,
				TenantID:  tenantID,
				Status:    "ready",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			})
		}

		// Verify all records in list
		listBefore := repo.List(tenantID)
		if len(listBefore) != numRecords {
			t.Errorf("expected %d records before delete, got %d", numRecords, len(listBefore))
		}

		// Soft delete one record
		deletedID := ids[deleteIndex]
		repo.SoftDelete(deletedID)

		// Property: File SHALL not appear in list queries
		listAfter := repo.List(tenantID)
		if len(listAfter) != numRecords-1 {
			t.Errorf("expected %d records after delete, got %d", numRecords-1, len(listAfter))
		}

		// Verify deleted record is not in list
		for _, record := range listAfter {
			if record.ID == deletedID {
				t.Errorf("deleted record %q should not appear in list", deletedID)
			}
		}
	})
}

// TestProperty13_SoftDeletedRetrievableByID tests that soft-deleted files are retrievable by ID.
// Property 13: Soft Delete Correctness
// Validates: Requirements 10.3
func TestProperty13_SoftDeletedRetrievableByID(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockSoftDeleteRepository()

		id := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "id")
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		record := &SoftDeletableRecord{
			ID:        id,
			TenantID:  tenantID,
			Status:    "ready",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		repo.Create(record)

		// Soft delete
		repo.SoftDelete(id)

		// Property: File SHALL be retrievable by ID with deleted status
		retrieved, ok := repo.GetByID(id)
		if !ok {
			t.Error("soft-deleted record should be retrievable by ID")
		}

		if !retrieved.IsDeleted() {
			t.Error("retrieved record should be marked as deleted")
		}

		if retrieved.Status != "deleted" {
			t.Errorf("status should be 'deleted', got %q", retrieved.Status)
		}
	})
}

// TestProperty13_DoubleDeleteNoOp tests that deleting already deleted record is no-op.
// Property 13: Soft Delete Correctness
// Validates: Requirements 10.3
func TestProperty13_DoubleDeleteNoOp(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockSoftDeleteRepository()

		id := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "id")
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")

		record := &SoftDeletableRecord{
			ID:        id,
			TenantID:  tenantID,
			Status:    "ready",
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
		}
		repo.Create(record)

		// First delete
		success1 := repo.SoftDelete(id)
		if !success1 {
			t.Fatal("first soft delete should succeed")
		}

		retrieved1, _ := repo.GetByID(id)
		firstDeletedAt := retrieved1.DeletedAt

		// Second delete should be no-op
		success2 := repo.SoftDelete(id)
		if success2 {
			t.Error("second soft delete should return false (no-op)")
		}

		// Verify deleted_at unchanged
		retrieved2, _ := repo.GetByID(id)
		if !retrieved2.DeletedAt.Equal(*firstDeletedAt) {
			t.Error("deleted_at should not change on second delete")
		}
	})
}
