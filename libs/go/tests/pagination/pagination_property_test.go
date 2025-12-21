package pagination_test

import (
	"testing"

	"github.com/auth-platform/libs/go/pagination"
	"pgregory.net/rapid"
)

// Property 19: Pagination Parameter Validation
func TestPaginationParameterValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		offset := rapid.IntRange(-10, 100).Draw(t, "offset")
		limit := rapid.IntRange(-10, 200).Draw(t, "limit")

		page, err := pagination.NewPage(offset, limit)

		// Negative offset should fail
		if offset < 0 && err == nil {
			t.Fatal("negative offset should fail")
		}

		// Non-positive limit should fail
		if limit <= 0 && err == nil {
			t.Fatal("non-positive limit should fail")
		}

		// Limit > MaxLimit should fail
		if limit > pagination.MaxLimit && err == nil {
			t.Fatal("limit exceeding max should fail")
		}

		// Valid params should succeed
		if offset >= 0 && limit > 0 && limit <= pagination.MaxLimit && err != nil {
			t.Fatalf("valid params should succeed: offset=%d, limit=%d, err=%v", offset, limit, err)
		}

		// Verify values when valid
		if err == nil {
			if page.Offset != offset {
				t.Fatalf("offset mismatch: got %d, want %d", page.Offset, offset)
			}
			if page.Limit != limit {
				t.Fatalf("limit mismatch: got %d, want %d", page.Limit, limit)
			}
		}
	})
}

// Property 20: Cursor Encoding Round-Trip
func TestCursorEncodingRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.StringMatching(`[a-zA-Z0-9]{10,50}`).Draw(t, "value")
		direction := rapid.SampledFrom([]string{"next", "prev"}).Draw(t, "direction")

		original := pagination.Cursor{
			Value:     value,
			Direction: direction,
		}

		// Encode
		encoded := pagination.EncodeCursor(original)

		// Decode
		decoded, err := pagination.DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		// Verify
		if decoded.Value != original.Value {
			t.Fatalf("value mismatch: got %s, want %s", decoded.Value, original.Value)
		}
		if decoded.Direction != original.Direction {
			t.Fatalf("direction mismatch: got %s, want %s", decoded.Direction, original.Direction)
		}
	})
}

// Property 21: Page Navigation Consistency
func TestPageNavigationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		offset := rapid.IntRange(0, 1000).Draw(t, "offset")
		limit := rapid.IntRange(1, pagination.MaxLimit).Draw(t, "limit")

		page, _ := pagination.NewPage(offset, limit)

		// Next page should have offset + limit
		next := page.Next()
		if next.Offset != offset+limit {
			t.Fatalf("next offset wrong: got %d, want %d", next.Offset, offset+limit)
		}
		if next.Limit != limit {
			t.Fatalf("next limit changed: got %d, want %d", next.Limit, limit)
		}

		// Previous of next should return to original (if offset >= limit)
		if offset >= limit {
			prev := next.Previous()
			if prev.Offset != offset {
				t.Fatalf("prev offset wrong: got %d, want %d", prev.Offset, offset)
			}
		}
	})
}

// Property 22: Page Result HasMore Correctness
func TestPageResultHasMore(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.Int64Range(0, 1000).Draw(t, "total")
		offset := rapid.IntRange(0, int(total)+10).Draw(t, "offset")
		limit := rapid.IntRange(1, 50).Draw(t, "limit")
		itemCount := rapid.IntRange(0, limit).Draw(t, "itemCount")

		items := make([]int, itemCount)
		page := pagination.Page{Offset: offset, Limit: limit}

		result := pagination.NewPageResult(items, total, page)

		// HasMore should be true if there are more items after current page
		expectedHasMore := int64(offset+itemCount) < total
		if result.HasMore != expectedHasMore {
			t.Fatalf("HasMore wrong: got %v, want %v (offset=%d, items=%d, total=%d)",
				result.HasMore, expectedHasMore, offset, itemCount, total)
		}
	})
}

// Property 23: PageInfo Calculation
func TestPageInfoCalculation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		total := rapid.Int64Range(0, 10000).Draw(t, "total")
		limit := rapid.IntRange(1, 100).Draw(t, "limit")
		pageNum := rapid.IntRange(1, 100).Draw(t, "pageNum")
		offset := (pageNum - 1) * limit

		info := pagination.NewPageInfo(offset, limit, total)

		// Page number should match
		if info.Page != pageNum {
			t.Fatalf("page number wrong: got %d, want %d", info.Page, pageNum)
		}

		// PerPage should match limit
		if info.PerPage != limit {
			t.Fatalf("per page wrong: got %d, want %d", info.PerPage, limit)
		}

		// TotalPages calculation
		expectedTotalPages := int((total + int64(limit) - 1) / int64(limit))
		if info.TotalPages != expectedTotalPages {
			t.Fatalf("total pages wrong: got %d, want %d", info.TotalPages, expectedTotalPages)
		}

		// TotalItems should match
		if info.TotalItems != total {
			t.Fatalf("total items wrong: got %d, want %d", info.TotalItems, total)
		}
	})
}

// Property 24: Empty Cursor Handling
func TestEmptyCursorHandling(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Empty string should return empty cursor without error
		cursor, err := pagination.DecodeCursor("")
		if err != nil {
			t.Fatalf("empty cursor should not error: %v", err)
		}
		if cursor.Value != "" || cursor.Direction != "" {
			t.Fatal("empty cursor should have empty fields")
		}
	})
}
