// Package pagination provides cursor-based and offset pagination.
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Page represents pagination parameters.
type Page struct {
	Offset int `json:"offset"`
	Limit  int `json:"limit"`
}

// DefaultLimit is the default page size.
const DefaultLimit = 20

// MaxLimit is the maximum allowed page size.
const MaxLimit = 100

// NewPage creates a new Page with validation.
func NewPage(offset, limit int) (Page, error) {
	if offset < 0 {
		return Page{}, fmt.Errorf("offset must be non-negative, got %d", offset)
	}
	if limit <= 0 {
		return Page{}, fmt.Errorf("limit must be positive, got %d", limit)
	}
	if limit > MaxLimit {
		return Page{}, fmt.Errorf("limit exceeds maximum of %d, got %d", MaxLimit, limit)
	}
	return Page{Offset: offset, Limit: limit}, nil
}

// DefaultPage returns a page with default values.
func DefaultPage() Page {
	return Page{Offset: 0, Limit: DefaultLimit}
}

// Next returns the next page.
func (p Page) Next() Page {
	return Page{Offset: p.Offset + p.Limit, Limit: p.Limit}
}

// Previous returns the previous page.
func (p Page) Previous() Page {
	newOffset := p.Offset - p.Limit
	if newOffset < 0 {
		newOffset = 0
	}
	return Page{Offset: newOffset, Limit: p.Limit}
}

// Cursor represents an opaque cursor for cursor-based pagination.
type Cursor struct {
	Value     string `json:"v"`
	Direction string `json:"d"` // "next" or "prev"
}

// EncodeCursor encodes a cursor to a base64 string.
func EncodeCursor(cursor Cursor) string {
	data, _ := json.Marshal(cursor)
	return base64.URLEncoding.EncodeToString(data)
}

// DecodeCursor decodes a base64 string to a cursor.
func DecodeCursor(encoded string) (Cursor, error) {
	if encoded == "" {
		return Cursor{}, nil
	}
	data, err := base64.URLEncoding.DecodeString(encoded)
	if err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor encoding: %w", err)
	}
	var cursor Cursor
	if err := json.Unmarshal(data, &cursor); err != nil {
		return Cursor{}, fmt.Errorf("invalid cursor format: %w", err)
	}
	return cursor, nil
}

// PageResult represents a paginated result.
type PageResult[T any] struct {
	Items      []T    `json:"items"`
	Total      int64  `json:"total"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
	PrevCursor string `json:"prev_cursor,omitempty"`
}

// NewPageResult creates a new page result.
func NewPageResult[T any](items []T, total int64, page Page) PageResult[T] {
	hasMore := int64(page.Offset+len(items)) < total
	return PageResult[T]{
		Items:   items,
		Total:   total,
		HasMore: hasMore,
	}
}

// WithCursors adds cursor information to the result.
func (r PageResult[T]) WithCursors(nextCursor, prevCursor string) PageResult[T] {
	r.NextCursor = nextCursor
	r.PrevCursor = prevCursor
	return r
}

// CursorPageResult represents a cursor-based paginated result.
type CursorPageResult[T any] struct {
	Items      []T    `json:"items"`
	HasMore    bool   `json:"has_more"`
	NextCursor string `json:"next_cursor,omitempty"`
}

// NewCursorPageResult creates a new cursor-based page result.
func NewCursorPageResult[T any](items []T, hasMore bool, nextCursor string) CursorPageResult[T] {
	return CursorPageResult[T]{
		Items:      items,
		HasMore:    hasMore,
		NextCursor: nextCursor,
	}
}

// PageInfo provides pagination metadata.
type PageInfo struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
	TotalItems int64 `json:"total_items"`
}

// NewPageInfo creates pagination metadata.
func NewPageInfo(offset, limit int, total int64) PageInfo {
	page := (offset / limit) + 1
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return PageInfo{
		Page:       page,
		PerPage:    limit,
		TotalPages: totalPages,
		TotalItems: total,
	}
}
