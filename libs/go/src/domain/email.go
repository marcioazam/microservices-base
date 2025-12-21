package domain

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// emailRegex is a simplified RFC 5322 compliant email regex.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)

// Email represents a validated email address.
type Email struct {
	value string
}

// NewEmail creates a new Email from a string, validating RFC 5322 compliance.
func NewEmail(value string) (Email, error) {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if normalized == "" {
		return Email{}, fmt.Errorf("email cannot be empty")
	}
	if len(normalized) > 254 {
		return Email{}, fmt.Errorf("email exceeds maximum length of 254 characters")
	}
	if !emailRegex.MatchString(normalized) {
		return Email{}, fmt.Errorf("invalid email format: %s", value)
	}
	return Email{value: normalized}, nil
}

// MustNewEmail creates a new Email, panicking on invalid input.
func MustNewEmail(value string) Email {
	email, err := NewEmail(value)
	if err != nil {
		panic(err)
	}
	return email
}

// String returns the email as a string.
func (e Email) String() string {
	return e.value
}

// LocalPart returns the local part of the email (before @).
func (e Email) LocalPart() string {
	parts := strings.Split(e.value, "@")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Domain returns the domain part of the email (after @).
func (e Email) Domain() string {
	parts := strings.Split(e.value, "@")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// IsEmpty returns true if the email is empty.
func (e Email) IsEmpty() bool {
	return e.value == ""
}

// Equals checks if two emails are equal.
func (e Email) Equals(other Email) bool {
	return e.value == other.value
}

// MarshalJSON implements json.Marshaler.
func (e Email) MarshalJSON() ([]byte, error) {
	return json.Marshal(e.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (e *Email) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	email, err := NewEmail(s)
	if err != nil {
		return err
	}
	*e = email
	return nil
}

// MarshalText implements encoding.TextMarshaler.
func (e Email) MarshalText() ([]byte, error) {
	return []byte(e.value), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (e *Email) UnmarshalText(data []byte) error {
	email, err := NewEmail(string(data))
	if err != nil {
		return err
	}
	*e = email
	return nil
}
