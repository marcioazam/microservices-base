package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// URL represents a validated URL.
type URL struct {
	value  string
	parsed *url.URL
}

// AllowedSchemes defines the allowed URL schemes.
var AllowedSchemes = map[string]bool{
	"http":  true,
	"https": true,
	"ftp":   true,
	"ftps":  true,
}

// NewURL creates a new URL with validation.
func NewURL(value string) (URL, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return URL{}, fmt.Errorf("URL cannot be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return URL{}, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme == "" {
		return URL{}, fmt.Errorf("URL must have a scheme")
	}
	if !AllowedSchemes[strings.ToLower(parsed.Scheme)] {
		return URL{}, fmt.Errorf("unsupported URL scheme: %s", parsed.Scheme)
	}
	if parsed.Host == "" {
		return URL{}, fmt.Errorf("URL must have a host")
	}
	return URL{value: trimmed, parsed: parsed}, nil
}

// MustNewURL creates a new URL, panicking on invalid input.
func MustNewURL(value string) URL {
	u, err := NewURL(value)
	if err != nil {
		panic(err)
	}
	return u
}

// String returns the URL as a string.
func (u URL) String() string {
	return u.value
}

// Scheme returns the URL scheme.
func (u URL) Scheme() string {
	if u.parsed == nil {
		return ""
	}
	return u.parsed.Scheme
}

// Host returns the URL host.
func (u URL) Host() string {
	if u.parsed == nil {
		return ""
	}
	return u.parsed.Host
}

// Path returns the URL path.
func (u URL) Path() string {
	if u.parsed == nil {
		return ""
	}
	return u.parsed.Path
}

// Query returns the URL query string.
func (u URL) Query() string {
	if u.parsed == nil {
		return ""
	}
	return u.parsed.RawQuery
}

// IsSecure returns true if the URL uses HTTPS or FTPS.
func (u URL) IsSecure() bool {
	scheme := strings.ToLower(u.Scheme())
	return scheme == "https" || scheme == "ftps"
}

// IsEmpty returns true if the URL is empty.
func (u URL) IsEmpty() bool {
	return u.value == ""
}

// Equals checks if two URLs are equal.
func (u URL) Equals(other URL) bool {
	return u.value == other.value
}

// MarshalJSON implements json.Marshaler.
func (u URL) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.value)
}

// UnmarshalJSON implements json.Unmarshaler.
func (u *URL) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	parsed, err := NewURL(s)
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}
