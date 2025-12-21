package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// e164Regex validates E.164 phone number format.
var e164Regex = regexp.MustCompile(`^\+[1-9]\d{1,14}$`)

// PhoneNumber represents a validated E.164 phone number.
type PhoneNumber struct {
	value string
}

// NewPhoneNumber creates a new PhoneNumber with E.164 validation.
func NewPhoneNumber(value string) (PhoneNumber, error) {
	normalized := normalizePhone(value)
	if normalized == "" {
		return PhoneNumber{}, fmt.Errorf("phone number cannot be empty")
	}
	if !e164Regex.MatchString(normalized) {
		return PhoneNumber{}, fmt.Errorf("invalid E.164 phone number: %s", value)
	}
	return PhoneNumber{value: normalized}, nil
}

// MustNewPhoneNumber creates a new PhoneNumber, panicking on invalid input.
func MustNewPhoneNumber(value string) PhoneNumber {
	phone, err := NewPhoneNumber(value)
	if err != nil {
		panic(err)
	}
	return phone
}

// String returns the phone number as a string.
func (p PhoneNumber) String() string {
	return p.value
}

// CountryCode returns the country code (first 1-3 digits after +).
func (p PhoneNumber) CountryCode() string {
	if len(p.value) < 2 {
		return ""
	}
	// Common country code lengths: 1 (US), 2 (UK), 3 (Brazil)
	for i := 4; i >= 2; i-- {
		if len(p.value) >= i {
			return p.value[1:i]
		}
	}
	return p.value[1:2]
}

// IsEmpty returns true if the phone number is empty.
func (p PhoneNumber) IsEmpty() bool {
	return p.value == ""
}

// Equals checks if two phone numbers are equal.
func (p PhoneNumber) Equals(other PhoneNumber) bool {
	return p.value == other.value
}

// MarshalJSON implements json.Marshaler.
func (p PhoneNumber) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *PhoneNumber) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	phone, err := NewPhoneNumber(s)
	if err != nil {
		return err
	}
	*p = phone
	return nil
}

func normalizePhone(value string) string {
	// Remove spaces, dashes, parentheses
	normalized := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' || r == '+' {
			return r
		}
		return -1
	}, value)
	return normalized
}
