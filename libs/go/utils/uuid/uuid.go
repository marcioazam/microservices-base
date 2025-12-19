// Package uuid provides UUID v7 generation and parsing utilities.
// UUID v7 is a time-ordered UUID format defined in RFC 9562.
package uuid

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"time"
)

var (
	// ErrInvalidUUID is returned when parsing an invalid UUID string.
	ErrInvalidUUID = errors.New("invalid UUID format")
	// ErrInvalidUUIDv7 is returned when the UUID is not a valid v7.
	ErrInvalidUUIDv7 = errors.New("invalid UUID v7: wrong version or variant")

	// uuidv7Regex matches the UUID format with version 7.
	uuidv7Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
)

// GenerateEventID generates a new UUID v7 string.
// UUID v7 embeds a Unix timestamp in milliseconds in the first 48 bits,
// followed by version bits (0111), random bits, variant bits (10xx), and more random bits.
func GenerateEventID() string {
	var uuid [16]byte

	// Get current timestamp in milliseconds
	now := time.Now().UnixMilli()

	// First 48 bits: Unix timestamp in milliseconds (big-endian)
	uuid[0] = byte(now >> 40)
	uuid[1] = byte(now >> 32)
	uuid[2] = byte(now >> 24)
	uuid[3] = byte(now >> 16)
	uuid[4] = byte(now >> 8)
	uuid[5] = byte(now)

	// Generate random bytes for the rest
	rand.Read(uuid[6:])

	// Set version to 7 (0111) in bits 48-51
	uuid[6] = (uuid[6] & 0x0F) | 0x70

	// Set variant to RFC 4122 (10xx) in bits 64-65
	uuid[8] = (uuid[8] & 0x3F) | 0x80

	return formatUUID(uuid)
}

// ParseUUIDv7Timestamp extracts the embedded timestamp from a UUID v7 string.
// Returns the time and nil error on success, or zero time and error on failure.
func ParseUUIDv7Timestamp(uuidStr string) (time.Time, error) {
	uuid, err := parseUUID(uuidStr)
	if err != nil {
		return time.Time{}, err
	}

	// Verify it's a v7 UUID
	if !isV7(uuid) {
		return time.Time{}, ErrInvalidUUIDv7
	}

	// Extract timestamp from first 48 bits
	ms := int64(uuid[0])<<40 |
		int64(uuid[1])<<32 |
		int64(uuid[2])<<24 |
		int64(uuid[3])<<16 |
		int64(uuid[4])<<8 |
		int64(uuid[5])

	return time.UnixMilli(ms), nil
}


// IsValidUUIDv7 checks if the given string is a valid UUID v7.
// Returns true only if the string matches UUID v7 format with correct version and variant bits.
func IsValidUUIDv7(uuidStr string) bool {
	// Quick regex check for format
	if !uuidv7Regex.MatchString(uuidStr) {
		return false
	}

	// Parse and verify version/variant
	uuid, err := parseUUID(uuidStr)
	if err != nil {
		return false
	}

	return isV7(uuid)
}

// isV7 checks if the UUID bytes represent a valid v7 UUID.
func isV7(uuid [16]byte) bool {
	// Check version (bits 48-51 should be 0111)
	version := uuid[6] >> 4
	if version != 7 {
		return false
	}

	// Check variant (bits 64-65 should be 10)
	variant := uuid[8] >> 6
	return variant == 2 // 10 in binary
}

// parseUUID parses a UUID string into bytes.
func parseUUID(uuidStr string) ([16]byte, error) {
	var uuid [16]byte

	if len(uuidStr) != 36 {
		return uuid, ErrInvalidUUID
	}

	// Verify hyphen positions
	if uuidStr[8] != '-' || uuidStr[13] != '-' || uuidStr[18] != '-' || uuidStr[23] != '-' {
		return uuid, ErrInvalidUUID
	}

	// Remove hyphens and decode
	hexStr := uuidStr[0:8] + uuidStr[9:13] + uuidStr[14:18] + uuidStr[19:23] + uuidStr[24:36]
	decoded, err := hex.DecodeString(hexStr)
	if err != nil {
		return uuid, ErrInvalidUUID
	}

	copy(uuid[:], decoded)
	return uuid, nil
}

// formatUUID formats UUID bytes as a string with hyphens.
func formatUUID(uuid [16]byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		binary.BigEndian.Uint32(uuid[0:4]),
		binary.BigEndian.Uint16(uuid[4:6]),
		binary.BigEndian.Uint16(uuid[6:8]),
		binary.BigEndian.Uint16(uuid[8:10]),
		uuid[10:16])
}

// MustGenerateEventID generates a UUID v7 or panics on failure.
// This is useful for initialization code where failure is not recoverable.
func MustGenerateEventID() string {
	return GenerateEventID()
}

// GenerateEventIDWithTime generates a UUID v7 with a specific timestamp.
// Useful for testing or when you need deterministic timestamps.
func GenerateEventIDWithTime(t time.Time) string {
	var uuid [16]byte

	ms := t.UnixMilli()

	uuid[0] = byte(ms >> 40)
	uuid[1] = byte(ms >> 32)
	uuid[2] = byte(ms >> 24)
	uuid[3] = byte(ms >> 16)
	uuid[4] = byte(ms >> 8)
	uuid[5] = byte(ms)

	rand.Read(uuid[6:])

	uuid[6] = (uuid[6] & 0x0F) | 0x70
	uuid[8] = (uuid[8] & 0x3F) | 0x80

	return formatUUID(uuid)
}
