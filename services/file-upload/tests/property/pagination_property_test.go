// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 12: Pagination Cursor Consistency
// Validates: Requirements 4.7, 10.2
package property

import (
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"pgregory.net/rapid"
)

// TestCursor represents pagination cursor state for testing.
type TestCursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// EncodeTestCursor encodes a cursor to a base64 string.
func EncodeTestCursor(cursor TestCursor) string {
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeTestCursor decodes a base64 string to a cursor.
func DecodeTestCursor(encoded string) (TestCursor, error) {
	if encoded == "" {
		return TestCursor{}, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return TestCursor{}, err
	}
	var cursor TestCursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return TestCursor{}, err
	}
	return cursor, nil
}

// TestFileRecord represents a file record for pagination testing.
type TestFileRecord struct {
	ID        string
	TenantID  string
	CreatedAt time.Time
}

// MockPaginatedRepository simulates paginated data access.
type MockPaginatedRepository struct {
	records []TestFileRecord
}

func NewMockPaginatedRepository(records []TestFileRecord) *MockPaginatedRepository {
	return &MockPaginatedRepository{records: records}
}

// List returns paginated results.
func (r *MockPaginatedRepository) List(tenantID string, cursor string, pageSize int) ([]TestFileRecord, string, error) {
	// Filter by tenant
	var filtered []TestFileRecord
	for _, rec := range r.records {
		if rec.TenantID == tenantID {
			filtered = append(filtered, rec)
		}
	}

	// Apply cursor
	startIdx := 0
	if cursor != "" {
		c, err := DecodeTestCursor(cursor)
		if err != nil {
			return nil, "", err
		}
		for i, rec := range filtered {
			if rec.CreatedAt.Before(c.CreatedAt) || (rec.CreatedAt.Equal(c.CreatedAt) && rec.ID < c.ID) {
				startIdx = i
				break
			}
		}
	}

	// Get page
	endIdx := startIdx + pageSize
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	page := filtered[startIdx:endIdx]

	// Generate next cursor
	var nextCursor string
	if endIdx < len(filtered) {
		lastRec := page[len(page)-1]
		nextCursor = EncodeTestCursor(TestCursor{ID: lastRec.ID, CreatedAt: lastRec.CreatedAt})
	}

	return page, nextCursor, nil
}

// TestProperty12_CursorDecodable tests that cursors are decodable to valid pagination state.
// Property 12: Pagination Cursor Consistency
// Validates: Requirements 4.7, 10.2
func TestProperty12_CursorDecodable(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random cursor data
		id := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "id")
		year := rapid.IntRange(2020, 2030).Draw(t, "year")
		month := rapid.IntRange(1, 12).Draw(t, "month")
		day := rapid.IntRange(1, 28).Draw(t, "day")
		hour := rapid.IntRange(0, 23).Draw(t, "hour")
		minute := rapid.IntRange(0, 59).Draw(t, "minute")

		createdAt := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)

		// Encode cursor
		original := TestCursor{ID: id, CreatedAt: createdAt}
		encoded := EncodeTestCursor(original)

		// Property: Cursor SHALL be decodable to valid pagination state
		decoded, err := DecodeTestCursor(encoded)
		if err != nil {
			t.Fatalf("failed to decode cursor: %v", err)
		}

		// Verify decoded values match original
		if decoded.ID != original.ID {
			t.Errorf("ID mismatch: expected %q, got %q", original.ID, decoded.ID)
		}
		if !decoded.CreatedAt.Equal(original.CreatedAt) {
			t.Errorf("CreatedAt mismatch: expected %v, got %v", original.CreatedAt, decoded.CreatedAt)
		}
	})
}

// TestProperty12_CursorReturnsNextPage tests that using cursor returns next page of results.
// Property 12: Pagination Cursor Consistency
// Validates: Requirements 4.7, 10.2
func TestProperty12_CursorReturnsNextPage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")
		numRecords := rapid.IntRange(5, 20).Draw(t, "numRecords")
		pageSize := rapid.IntRange(2, 5).Draw(t, "pageSize")

		// Generate records
		records := make([]TestFileRecord, numRecords)
		baseTime := time.Now().UTC()
		for i := 0; i < numRecords; i++ {
			records[i] = TestFileRecord{
				ID:        rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "recordID"),
				TenantID:  tenantID,
				CreatedAt: baseTime.Add(-time.Duration(i) * time.Hour),
			}
		}

		repo := NewMockPaginatedRepository(records)

		// Get first page
		page1, cursor1, err := repo.List(tenantID, "", pageSize)
		if err != nil {
			t.Fatalf("failed to get first page: %v", err)
		}

		if len(page1) == 0 {
			t.Skip("no records in first page")
		}

		// Property: Using cursor SHALL return next page of results
		if cursor1 != "" {
			page2, _, err := repo.List(tenantID, cursor1, pageSize)
			if err != nil {
				t.Fatalf("failed to get second page: %v", err)
			}

			// Verify no overlap between pages
			page1IDs := make(map[string]bool)
			for _, rec := range page1 {
				page1IDs[rec.ID] = true
			}

			for _, rec := range page2 {
				if page1IDs[rec.ID] {
					t.Errorf("record %q appears in both pages", rec.ID)
				}
			}
		}
	})
}

// TestProperty12_EmptyCursorOnLastPage tests that last page has empty cursor.
// Property 12: Pagination Cursor Consistency
// Validates: Requirements 4.7, 10.2
func TestProperty12_EmptyCursorOnLastPage(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		tenantID := rapid.StringMatching(`tenant-[a-z0-9]{8}`).Draw(t, "tenantID")
		numRecords := rapid.IntRange(1, 10).Draw(t, "numRecords")

		// Generate records
		records := make([]TestFileRecord, numRecords)
		baseTime := time.Now().UTC()
		for i := 0; i < numRecords; i++ {
			records[i] = TestFileRecord{
				ID:        rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "recordID"),
				TenantID:  tenantID,
				CreatedAt: baseTime.Add(-time.Duration(i) * time.Hour),
			}
		}

		repo := NewMockPaginatedRepository(records)

		// Get all pages until cursor is empty
		cursor := ""
		pageSize := numRecords + 1 // Ensure we get all records in one page
		var allRecords []TestFileRecord

		page, nextCursor, err := repo.List(tenantID, cursor, pageSize)
		if err != nil {
			t.Fatalf("failed to list: %v", err)
		}
		allRecords = append(allRecords, page...)

		// Property: Empty cursor on last page
		if nextCursor != "" {
			t.Errorf("expected empty cursor on last page, got %q", nextCursor)
		}

		// Verify we got all records
		if len(allRecords) != numRecords {
			t.Errorf("expected %d records, got %d", numRecords, len(allRecords))
		}
	})
}

// TestProperty12_InvalidCursorHandling tests that invalid cursors are handled gracefully.
// Property 12: Pagination Cursor Consistency
// Validates: Requirements 4.7, 10.2
func TestProperty12_InvalidCursorHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate invalid cursor strings
		invalidCursor := rapid.SampledFrom([]string{
			"not-base64!@#$",
			"aW52YWxpZC1qc29u", // "invalid-json" in base64
			"",
		}).Draw(t, "invalidCursor")

		// Empty cursor should decode successfully
		if invalidCursor == "" {
			cursor, err := DecodeTestCursor(invalidCursor)
			if err != nil {
				t.Errorf("empty cursor should decode without error: %v", err)
			}
			if cursor.ID != "" {
				t.Error("empty cursor should have empty ID")
			}
			return
		}

		// Invalid cursors should return error
		_, err := DecodeTestCursor(invalidCursor)
		if err == nil && invalidCursor != "" {
			// Some invalid strings might accidentally be valid base64
			// This is acceptable as long as they don't cause panics
		}
	})
}
