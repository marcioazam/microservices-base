// Package domain provides UUID v7 generation for event IDs.
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrInvalidUUID indicates a malformed UUID string.
var ErrInvalidUUID = errors.New("invalid UUID format")

// GenerateUUIDv7 generates a UUID v7 compliant ID.
func GenerateUUIDv7() string {
	var uuid [16]byte
	now := time.Now().UnixMilli()

	// First 48 bits: timestamp in milliseconds
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Fill remaining bytes with random data
	rand.Read(uuid[6:])

	// Set version (4 bits) to 7
	uuid[6] = (uuid[6] & 0x0f) | 0x70

	// Set variant (2 bits) to RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// ParseUUIDv7Timestamp extracts the timestamp from a UUID v7 string.
func ParseUUIDv7Timestamp(id string) (time.Time, error) {
	if len(id) != 36 {
		return time.Time{}, ErrInvalidUUID
	}

	// Validate hyphen positions
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return time.Time{}, ErrInvalidUUID
	}

	// Remove hyphens
	clean := strings.ReplaceAll(id, "-", "")
	if len(clean) != 32 {
		return time.Time{}, ErrInvalidUUID
	}

	// Decode hex
	bytes, err := hex.DecodeString(clean)
	if err != nil {
		return time.Time{}, ErrInvalidUUID
	}

	// Extract timestamp from first 48 bits
	ms := int64(bytes[0])<<40 | int64(bytes[1])<<32 | int64(bytes[2])<<24 |
		int64(bytes[3])<<16 | int64(bytes[4])<<8 | int64(bytes[5])

	return time.UnixMilli(ms), nil
}

// IsValidUUIDv7 checks if a string is a valid UUID v7.
func IsValidUUIDv7(id string) bool {
	if len(id) != 36 {
		return false
	}
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return false
	}
	if id[14] != '7' {
		return false
	}
	variant := id[19]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' && variant != 'A' && variant != 'B' {
		return false
	}
	_, err := ParseUUIDv7Timestamp(id)
	return err == nil
}
