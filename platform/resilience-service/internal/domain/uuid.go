// Package domain provides UUID v7 generation for event IDs.
// This package re-exports functions from libs/go/uuid for backward compatibility.
package domain

import (
	"time"

	libuuid "github.com/auth-platform/libs/go/utils/uuid"
)

// ErrInvalidUUID indicates a malformed UUID string.
var ErrInvalidUUID = libuuid.ErrInvalidUUID

// GenerateEventID generates a UUID v7 compliant event ID.
// UUID v7 provides time-ordered, cryptographically random identifiers per RFC 9562.
func GenerateEventID() string {
	return libuuid.GenerateEventID()
}

// ParseUUIDv7Timestamp extracts the timestamp from a UUID v7 string.
func ParseUUIDv7Timestamp(id string) (time.Time, error) {
	return libuuid.ParseUUIDv7Timestamp(id)
}

// IsValidUUIDv7 checks if a string is a valid UUID v7.
func IsValidUUIDv7(id string) bool {
	return libuuid.IsValidUUIDv7(id)
}
