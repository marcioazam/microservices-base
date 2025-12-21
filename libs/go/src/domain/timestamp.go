package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

// Timestamp represents a validated timestamp with ISO 8601 support.
type Timestamp struct {
	value time.Time
}

// Common time formats for parsing.
var timeFormats = []string{
	time.RFC3339,
	time.RFC3339Nano,
	"2006-01-02T15:04:05Z",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Now returns the current timestamp.
func Now() Timestamp {
	return Timestamp{value: time.Now().UTC()}
}

// NewTimestamp creates a Timestamp from a time.Time.
func NewTimestamp(t time.Time) Timestamp {
	return Timestamp{value: t.UTC()}
}

// ParseTimestamp parses an ISO 8601 timestamp string.
func ParseTimestamp(value string) (Timestamp, error) {
	if value == "" {
		return Timestamp{}, fmt.Errorf("timestamp cannot be empty")
	}
	for _, format := range timeFormats {
		if t, err := time.Parse(format, value); err == nil {
			return Timestamp{value: t.UTC()}, nil
		}
	}
	return Timestamp{}, fmt.Errorf("invalid timestamp format: %s", value)
}

// MustParseTimestamp parses a timestamp, panicking on invalid input.
func MustParseTimestamp(value string) Timestamp {
	ts, err := ParseTimestamp(value)
	if err != nil {
		panic(err)
	}
	return ts
}

// FromUnix creates a Timestamp from Unix seconds.
func FromUnix(sec int64) Timestamp {
	return Timestamp{value: time.Unix(sec, 0).UTC()}
}

// FromUnixMilli creates a Timestamp from Unix milliseconds.
func FromUnixMilli(ms int64) Timestamp {
	return Timestamp{value: time.UnixMilli(ms).UTC()}
}

// Time returns the underlying time.Time.
func (t Timestamp) Time() time.Time {
	return t.value
}

// Unix returns the Unix timestamp in seconds.
func (t Timestamp) Unix() int64 {
	return t.value.Unix()
}

// UnixMilli returns the Unix timestamp in milliseconds.
func (t Timestamp) UnixMilli() int64 {
	return t.value.UnixMilli()
}

// String returns the timestamp in RFC3339 format.
func (t Timestamp) String() string {
	return t.value.Format(time.RFC3339)
}

// Format returns the timestamp in the specified format.
func (t Timestamp) Format(layout string) string {
	return t.value.Format(layout)
}

// IsZero returns true if the timestamp is zero.
func (t Timestamp) IsZero() bool {
	return t.value.IsZero()
}

// Before returns true if t is before other.
func (t Timestamp) Before(other Timestamp) bool {
	return t.value.Before(other.value)
}

// After returns true if t is after other.
func (t Timestamp) After(other Timestamp) bool {
	return t.value.After(other.value)
}

// Equals checks if two timestamps are equal.
func (t Timestamp) Equals(other Timestamp) bool {
	return t.value.Equal(other.value)
}

// Add adds a duration to the timestamp.
func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp{value: t.value.Add(d)}
}

// Sub returns the duration between two timestamps.
func (t Timestamp) Sub(other Timestamp) time.Duration {
	return t.value.Sub(other.value)
}

// MarshalJSON implements json.Marshaler.
func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.value.Format(time.RFC3339Nano))
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	ts, err := ParseTimestamp(s)
	if err != nil {
		return err
	}
	*t = ts
	return nil
}
