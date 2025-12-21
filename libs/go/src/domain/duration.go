package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Duration represents a validated duration with human-readable parsing.
type Duration struct {
	value time.Duration
}

// durationRegex matches human-readable duration strings.
var durationRegex = regexp.MustCompile(`^(\d+)(ns|us|µs|ms|s|m|h|d|w)$`)

// NewDuration creates a Duration from time.Duration.
func NewDuration(d time.Duration) Duration {
	return Duration{value: d}
}

// ParseDuration parses a human-readable duration string.
// Supports: ns, us, µs, ms, s, m, h, d (days), w (weeks)
func ParseDuration(value string) (Duration, error) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return Duration{}, fmt.Errorf("duration cannot be empty")
	}

	// Try standard Go duration first
	if d, err := time.ParseDuration(trimmed); err == nil {
		return Duration{value: d}, nil
	}

	// Try extended format with days/weeks
	match := durationRegex.FindStringSubmatch(trimmed)
	if match == nil {
		return Duration{}, fmt.Errorf("invalid duration format: %s", value)
	}

	num, _ := strconv.ParseInt(match[1], 10, 64)
	unit := match[2]

	var d time.Duration
	switch unit {
	case "ns":
		d = time.Duration(num) * time.Nanosecond
	case "us", "µs":
		d = time.Duration(num) * time.Microsecond
	case "ms":
		d = time.Duration(num) * time.Millisecond
	case "s":
		d = time.Duration(num) * time.Second
	case "m":
		d = time.Duration(num) * time.Minute
	case "h":
		d = time.Duration(num) * time.Hour
	case "d":
		d = time.Duration(num) * 24 * time.Hour
	case "w":
		d = time.Duration(num) * 7 * 24 * time.Hour
	default:
		return Duration{}, fmt.Errorf("unknown duration unit: %s", unit)
	}

	return Duration{value: d}, nil
}

// MustParseDuration parses a duration, panicking on invalid input.
func MustParseDuration(value string) Duration {
	d, err := ParseDuration(value)
	if err != nil {
		panic(err)
	}
	return d
}

// Seconds creates a Duration from seconds.
func Seconds(n int64) Duration {
	return Duration{value: time.Duration(n) * time.Second}
}

// Minutes creates a Duration from minutes.
func Minutes(n int64) Duration {
	return Duration{value: time.Duration(n) * time.Minute}
}

// Hours creates a Duration from hours.
func Hours(n int64) Duration {
	return Duration{value: time.Duration(n) * time.Hour}
}

// Days creates a Duration from days.
func Days(n int64) Duration {
	return Duration{value: time.Duration(n) * 24 * time.Hour}
}

// Value returns the underlying time.Duration.
func (d Duration) Value() time.Duration {
	return d.value
}

// Nanoseconds returns the duration in nanoseconds.
func (d Duration) Nanoseconds() int64 {
	return d.value.Nanoseconds()
}

// Milliseconds returns the duration in milliseconds.
func (d Duration) Milliseconds() int64 {
	return d.value.Milliseconds()
}

// Seconds returns the duration in seconds.
func (d Duration) SecondsFloat() float64 {
	return d.value.Seconds()
}

// Minutes returns the duration in minutes.
func (d Duration) MinutesFloat() float64 {
	return d.value.Minutes()
}

// Hours returns the duration in hours.
func (d Duration) HoursFloat() float64 {
	return d.value.Hours()
}

// String returns the duration as a string.
func (d Duration) String() string {
	return d.value.String()
}

// HumanReadable returns a human-readable representation.
func (d Duration) HumanReadable() string {
	if d.value < time.Minute {
		return d.value.String()
	}
	if d.value < time.Hour {
		return fmt.Sprintf("%.0fm", d.value.Minutes())
	}
	if d.value < 24*time.Hour {
		return fmt.Sprintf("%.0fh", d.value.Hours())
	}
	days := d.value.Hours() / 24
	return fmt.Sprintf("%.0fd", days)
}

// IsZero returns true if the duration is zero.
func (d Duration) IsZero() bool {
	return d.value == 0
}

// IsPositive returns true if the duration is positive.
func (d Duration) IsPositive() bool {
	return d.value > 0
}

// IsNegative returns true if the duration is negative.
func (d Duration) IsNegative() bool {
	return d.value < 0
}

// Equals checks if two durations are equal.
func (d Duration) Equals(other Duration) bool {
	return d.value == other.value
}

// Add adds two durations.
func (d Duration) Add(other Duration) Duration {
	return Duration{value: d.value + other.value}
}

// Multiply multiplies the duration by a factor.
func (d Duration) Multiply(factor int64) Duration {
	return Duration{value: d.value * time.Duration(factor)}
}

// MarshalJSON implements json.Marshaler.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.value.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := ParseDuration(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
