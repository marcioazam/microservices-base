// Package resilience provides shared resilience primitives for distributed systems.
// This package contains domain primitives like event ID generation, correlation functions,
// and time serialization helpers that are used across multiple services.
package resilience

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateEventID generates a unique event identifier.
// Format: timestamp-random (e.g., "20241216150405-a1b2c3d4")
// The timestamp provides temporal ordering while the random suffix ensures uniqueness.
func GenerateEventID() string {
	ts := time.Now().Format("20060102150405")
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return ts + "-" + hex.EncodeToString(b)
}

// GenerateEventIDWithPrefix generates a unique event identifier with a custom prefix.
// Format: prefix-timestamp-random (e.g., "cb-20241216150405-a1b2c3d4")
func GenerateEventIDWithPrefix(prefix string) string {
	if prefix == "" {
		return GenerateEventID()
	}
	ts := time.Now().Format("20060102150405")
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return prefix + "-" + ts + "-" + hex.EncodeToString(b)
}
