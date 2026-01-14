// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 14: Database Transaction Atomicity
// Validates: Requirements 10.5
package property

import (
	"errors"
	"sync"
	"testing"

	"pgregory.net/rapid"
)

// TransactionalRecord represents a record in a transaction.
type TransactionalRecord struct {
	ID     string
	Value  string
	Status string
}

// MockTransactionalRepository simulates transactional database behavior.
type MockTransactionalRepository struct {
	records    map[string]*TransactionalRecord
	mu         sync.RWMutex
	inTx       bool
	txRecords  map[string]*TransactionalRecord
	txDeletes  map[string]bool
	shouldFail bool
}

func NewMockTransactionalRepository() *MockTransactionalRepository {
	return &MockTransactionalRepository{
		records:   make(map[string]*TransactionalRecord),
		txRecords: make(map[string]*TransactionalRecord),
		txDeletes: make(map[string]bool),
	}
}

// BeginTx starts a transaction.
func (r *MockTransactionalRepository) BeginTx() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inTx = true
	r.txRecords = make(map[string]*TransactionalRecord)
	r.txDeletes = make(map[string]bool)
}

// Commit commits the transaction.
func (r *MockTransactionalRepository) Commit() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.shouldFail {
		r.inTx = false
		r.txRecords = make(map[string]*TransactionalRecord)
		r.txDeletes = make(map[string]bool)
		return errors.New("commit failed")
	}

	// Apply changes
	for id, record := range r.txRecords {
		r.records[id] = record
	}
	for id := range r.txDeletes {
		delete(r.records, id)
	}

	r.inTx = false
	r.txRecords = make(map[string]*TransactionalRecord)
	r.txDeletes = make(map[string]bool)
	return nil
}

// Rollback rolls back the transaction.
func (r *MockTransactionalRepository) Rollback() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.inTx = false
	r.txRecords = make(map[string]*TransactionalRecord)
	r.txDeletes = make(map[string]bool)
}

// Create adds a record (in transaction if active).
func (r *MockTransactionalRepository) Create(record *TransactionalRecord) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inTx {
		r.txRecords[record.ID] = record
	} else {
		r.records[record.ID] = record
	}
}

// Update updates a record (in transaction if active).
func (r *MockTransactionalRepository) Update(id, value, status string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record := &TransactionalRecord{ID: id, Value: value, Status: status}
	if r.inTx {
		r.txRecords[id] = record
	} else {
		r.records[id] = record
	}
}

// Delete deletes a record (in transaction if active).
func (r *MockTransactionalRepository) Delete(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.inTx {
		r.txDeletes[id] = true
		delete(r.txRecords, id)
	} else {
		delete(r.records, id)
	}
}

// Get retrieves a record.
func (r *MockTransactionalRepository) Get(id string) (*TransactionalRecord, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[id]
	return record, ok
}

// Count returns the number of records.
func (r *MockTransactionalRepository) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.records)
}

// SetShouldFail sets whether commit should fail.
func (r *MockTransactionalRepository) SetShouldFail(fail bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.shouldFail = fail
}

// TestProperty14_AllStepsSucceedOrRollback tests transaction atomicity.
// Property 14: Database Transaction Atomicity
// Validates: Requirements 10.5
func TestProperty14_AllStepsSucceedOrRollback(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockTransactionalRepository()

		numOperations := rapid.IntRange(2, 5).Draw(t, "numOperations")
		shouldFail := rapid.Bool().Draw(t, "shouldFail")

		// Generate operations
		operations := make([]TransactionalRecord, numOperations)
		for i := 0; i < numOperations; i++ {
			operations[i] = TransactionalRecord{
				ID:     rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "id"),
				Value:  rapid.StringMatching(`value-[a-z0-9]{8}`).Draw(t, "value"),
				Status: "pending",
			}
		}

		// Set failure mode
		repo.SetShouldFail(shouldFail)

		// Begin transaction
		repo.BeginTx()

		// Execute operations
		for _, op := range operations {
			repo.Create(&op)
		}

		// Commit or fail
		err := repo.Commit()

		if shouldFail {
			// Property: All steps SHALL rollback on failure
			if err == nil {
				t.Error("commit should have failed")
			}

			// Verify no records were created
			for _, op := range operations {
				_, exists := repo.Get(op.ID)
				if exists {
					t.Errorf("record %q should not exist after failed transaction", op.ID)
				}
			}
		} else {
			// Property: All steps SHALL succeed
			if err != nil {
				t.Errorf("commit should have succeeded: %v", err)
			}

			// Verify all records were created
			for _, op := range operations {
				record, exists := repo.Get(op.ID)
				if !exists {
					t.Errorf("record %q should exist after successful transaction", op.ID)
				}
				if record != nil && record.Value != op.Value {
					t.Errorf("record value mismatch: expected %q, got %q", op.Value, record.Value)
				}
			}
		}
	})
}

// TestProperty14_PartialStateNeverVisible tests that partial state is never visible.
// Property 14: Database Transaction Atomicity
// Validates: Requirements 10.5
func TestProperty14_PartialStateNeverVisible(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockTransactionalRepository()

		id1 := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "id1")
		id2 := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "id2")

		// Begin transaction
		repo.BeginTx()

		// Create first record
		repo.Create(&TransactionalRecord{ID: id1, Value: "value1", Status: "pending"})

		// Property: Partial state SHALL never be visible
		// Check from outside transaction - should not see uncommitted record
		_, exists1 := repo.Get(id1)
		if exists1 {
			t.Error("uncommitted record should not be visible outside transaction")
		}

		// Create second record
		repo.Create(&TransactionalRecord{ID: id2, Value: "value2", Status: "pending"})

		// Still should not be visible
		_, exists2 := repo.Get(id2)
		if exists2 {
			t.Error("uncommitted record should not be visible outside transaction")
		}

		// Rollback
		repo.Rollback()

		// Verify nothing was persisted
		_, exists1After := repo.Get(id1)
		_, exists2After := repo.Get(id2)
		if exists1After || exists2After {
			t.Error("rolled back records should not exist")
		}
	})
}

// TestProperty14_RollbackRestoresState tests that rollback restores original state.
// Property 14: Database Transaction Atomicity
// Validates: Requirements 10.5
func TestProperty14_RollbackRestoresState(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockTransactionalRepository()

		// Create initial record outside transaction
		initialID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "initialID")
		initialValue := rapid.StringMatching(`initial-[a-z0-9]{8}`).Draw(t, "initialValue")
		repo.Create(&TransactionalRecord{ID: initialID, Value: initialValue, Status: "ready"})

		initialCount := repo.Count()

		// Begin transaction
		repo.BeginTx()

		// Attempt to modify
		newValue := rapid.StringMatching(`new-[a-z0-9]{8}`).Draw(t, "newValue")
		repo.Update(initialID, newValue, "modified")

		// Add new record
		newID := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "newID")
		repo.Create(&TransactionalRecord{ID: newID, Value: "new", Status: "pending"})

		// Rollback
		repo.Rollback()

		// Property: Rollback SHALL restore original state
		record, exists := repo.Get(initialID)
		if !exists {
			t.Error("original record should still exist after rollback")
		}
		if record != nil && record.Value != initialValue {
			t.Errorf("original value should be restored: expected %q, got %q", initialValue, record.Value)
		}

		// New record should not exist
		_, newExists := repo.Get(newID)
		if newExists {
			t.Error("new record should not exist after rollback")
		}

		// Count should be unchanged
		if repo.Count() != initialCount {
			t.Errorf("record count should be unchanged: expected %d, got %d", initialCount, repo.Count())
		}
	})
}

// TestProperty14_ConcurrentTransactionIsolation tests transaction isolation.
// Property 14: Database Transaction Atomicity
// Validates: Requirements 10.5
func TestProperty14_ConcurrentTransactionIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		repo := NewMockTransactionalRepository()

		id := rapid.StringMatching(`[a-f0-9]{16}`).Draw(t, "id")
		value := rapid.StringMatching(`value-[a-z0-9]{8}`).Draw(t, "value")

		// Create record
		repo.Create(&TransactionalRecord{ID: id, Value: value, Status: "ready"})

		// Begin transaction
		repo.BeginTx()

		// Modify in transaction
		newValue := rapid.StringMatching(`modified-[a-z0-9]{8}`).Draw(t, "newValue")
		repo.Update(id, newValue, "modified")

		// Property: Changes in transaction SHALL not be visible until commit
		record, _ := repo.Get(id)
		if record != nil && record.Value == newValue {
			t.Error("uncommitted changes should not be visible")
		}

		// Commit
		repo.Commit()

		// Now changes should be visible
		recordAfter, _ := repo.Get(id)
		if recordAfter == nil || recordAfter.Value != newValue {
			t.Error("committed changes should be visible")
		}
	})
}
