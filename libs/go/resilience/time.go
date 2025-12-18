package resilience

import "time"

// TimeFormat is the standard time format for JSON serialization.
// Uses RFC3339Nano for maximum precision.
const TimeFormat = time.RFC3339Nano

// MarshalTime formats a time value for JSON serialization.
func MarshalTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// UnmarshalTime parses a time value from JSON serialization.
func UnmarshalTime(s string) (time.Time, error) {
	return time.Parse(TimeFormat, s)
}

// MarshalTimePtr formats a time pointer for JSON serialization.
// Returns empty string if the pointer is nil.
func MarshalTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(TimeFormat)
}

// UnmarshalTimePtr parses a time value and returns a pointer.
// Returns nil if the string is empty.
func UnmarshalTimePtr(s string) (*time.Time, error) {
	if s == "" {
		return nil, nil
	}
	t, err := time.Parse(TimeFormat, s)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

// NowUTC returns the current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// DurationToMillis converts a duration to milliseconds.
func DurationToMillis(d time.Duration) int64 {
	return d.Milliseconds()
}

// MillisToDuration converts milliseconds to a duration.
func MillisToDuration(ms int64) time.Duration {
	return time.Duration(ms) * time.Millisecond
}
