// Package repository provides database access implementations.
package repository

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

// Cursor represents pagination cursor state.
type Cursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
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
