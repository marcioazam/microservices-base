package domain

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// uuidRegex validates UUID format.
var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// UUID represents a validated UUID (v4).
type UUID struct {
	value string
}

// NewUUID generates a new random UUID v4.
func NewUUID() UUID {
	uuid := make([]byte, 16)
	_, _ = rand.Read(uuid)
	// Set version 4
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	// Set variant RFC 4122
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return UUID{value: formatUUID(uuid)}
}

// ParseUUID parses a UUID string.
func ParseUUID(value string) (UUID, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return UUID{}, fmt.Errorf("UUID cannot be empty")
	}
	if !uuidRegex.MatchString(normalized) {
		return UUID{}, fmt.Errorf("invalid UUID format: %s", value)
	}
	return UUID{value: normalized}, nil
}

// MustParseUUID parses a UUID string, panicking on invalid input.
func MustParseUUID(value string) UUID {
	uuid, err := ParseUUID(value)
	if err != nil {
		panic(err)
	}
	return uuid
}

// NilUUID returns the nil UUID (all zeros).
func NilUUID() UUID {
	return UUID{value: "00000000-0000-0000-0000-000000000000"}
}

// String returns the UUID as a string.
func (u UUID) String() string {
	return u.value
}

// IsNil returns true if this is the nil UUID.
func (u UUID) IsNil() bool {
	return u.value == "" || u.value == "00000000-0000-0000-0000-000000000000"
}

// Equals checks if two UUIDs are equal.
func (u UUID) Equals(other UUID) bool {
	return u.value == other.value
}

// Bytes returns the UUID as a 16-byte array.
func (u UUID) Bytes() [16]byte {
	var result [16]byte
	hex.Decode(result[:4], []byte(u.value[0:8]))
	hex.Decode(result[4:6], []byte(u.value[9:13]))
	hex.Decode(result[6:8], []byte(u.value[14:18]))
	hex.Decode(result[8:10], []byte(u.value[19:23]))
	hex.Decode(result[10:16], []byte(u.value[24:36]))
	return result
}

// MarshalJSON implements json.Marshaler.
func (u UUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *UUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	uuid, err := ParseUUID(s)
	if err != nil {
		return err
	}
	*u = uuid
	return nil
}

func formatUUID(uuid []byte) string {
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
