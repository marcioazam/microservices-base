package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
)

// UUID represents a universally unique identifier.
type UUID [16]byte

// NewUUID generates a new random UUID (v4).
func NewUUID() (UUID, error) {
	var uuid UUID
	_, err := io.ReadFull(rand.Reader, uuid[:])
	if err != nil {
		return uuid, err
	}
	// Set version (4) and variant (RFC 4122)
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return uuid, nil
}

// MustNewUUID generates a new UUID or panics.
func MustNewUUID() UUID {
	uuid, err := NewUUID()
	if err != nil {
		panic(err)
	}
	return uuid
}

// String returns the string representation.
func (u UUID) String() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(u[0:4]),
		hex.EncodeToString(u[4:6]),
		hex.EncodeToString(u[6:8]),
		hex.EncodeToString(u[8:10]),
		hex.EncodeToString(u[10:16]),
	)
}

// ParseUUID parses a UUID string.
func ParseUUID(s string) (UUID, error) {
	var uuid UUID
	if len(s) != 36 {
		return uuid, fmt.Errorf("invalid UUID length: %d", len(s))
	}
	// Remove hyphens
	clean := s[0:8] + s[9:13] + s[14:18] + s[19:23] + s[24:36]
	bytes, err := hex.DecodeString(clean)
	if err != nil {
		return uuid, err
	}
	copy(uuid[:], bytes)
	return uuid, nil
}

// IsZero returns true if UUID is all zeros.
func (u UUID) IsZero() bool {
	for _, b := range u {
		if b != 0 {
			return false
		}
	}
	return true
}

// Bytes returns the UUID as a byte slice.
func (u UUID) Bytes() []byte {
	return u[:]
}
