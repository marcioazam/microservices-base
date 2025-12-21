// Package testing provides domain-specific generators for property-based testing.
package testing

import (
	"fmt"
	"strings"
	"time"

	"pgregory.net/rapid"
)

// EmailGen generates valid email addresses.
func EmailGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		local := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "local")
		domain := rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net", "io", "dev"}).Draw(t, "tld")
		return fmt.Sprintf("%s@%s.%s", local, domain, tld)
	})
}

// UUIDGen generates valid UUID v4 strings.
func UUIDGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// UUID v4 format: xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx
		hex := "0123456789abcdef"
		var b strings.Builder
		for i := 0; i < 36; i++ {
			switch i {
			case 8, 13, 18, 23:
				b.WriteByte('-')
			case 14:
				b.WriteByte('4') // Version 4
			case 19:
				// Variant bits (8, 9, a, or b)
				b.WriteByte(hex[rapid.IntRange(8, 11).Draw(t, "variant")])
			default:
				b.WriteByte(hex[rapid.IntRange(0, 15).Draw(t, "hex")])
			}
		}
		return b.String()
	})
}

// ULIDGen generates valid ULID strings.
func ULIDGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// ULID: 26 characters, Crockford's Base32
		chars := "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
		var b strings.Builder
		for i := 0; i < 26; i++ {
			b.WriteByte(chars[rapid.IntRange(0, 31).Draw(t, "char")])
		}
		return b.String()
	})
}

// Money represents a monetary value.
type Money struct {
	Amount   int64  // Amount in smallest unit (cents)
	Currency string // ISO 4217 currency code
}

// MoneyGen generates valid Money values.
func MoneyGen() *rapid.Generator[Money] {
	return rapid.Custom(func(t *rapid.T) Money {
		return Money{
			Amount:   rapid.Int64Range(0, 999999999).Draw(t, "amount"),
			Currency: rapid.SampledFrom([]string{"USD", "EUR", "GBP", "JPY", "CAD"}).Draw(t, "currency"),
		}
	})
}

// PhoneNumberGen generates valid phone numbers (E.164 format).
func PhoneNumberGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		countryCode := rapid.SampledFrom([]string{"1", "44", "49", "33", "81"}).Draw(t, "country")
		number := rapid.StringMatching(`[0-9]{10}`).Draw(t, "number")
		return fmt.Sprintf("+%s%s", countryCode, number)
	})
}

// URLGen generates valid HTTP/HTTPS URLs.
func URLGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		scheme := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "scheme")
		domain := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net", "io"}).Draw(t, "tld")
		path := rapid.StringMatching(`/[a-z]{0,10}`).Draw(t, "path")
		return fmt.Sprintf("%s://%s.%s%s", scheme, domain, tld, path)
	})
}

// IPAddressGen generates valid IPv4 addresses.
func IPAddressGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return fmt.Sprintf("%d.%d.%d.%d",
			rapid.IntRange(1, 255).Draw(t, "o1"),
			rapid.IntRange(0, 255).Draw(t, "o2"),
			rapid.IntRange(0, 255).Draw(t, "o3"),
			rapid.IntRange(1, 254).Draw(t, "o4"),
		)
	})
}

// TimestampGen generates timestamps within a range.
func TimestampGen(start, end time.Time) *rapid.Generator[time.Time] {
	return rapid.Custom(func(t *rapid.T) time.Time {
		delta := end.Sub(start)
		offset := rapid.Int64Range(0, int64(delta)).Draw(t, "offset")
		return start.Add(time.Duration(offset))
	})
}

// RecentTimestampGen generates timestamps within the last 30 days.
func RecentTimestampGen() *rapid.Generator[time.Time] {
	now := time.Now()
	return TimestampGen(now.AddDate(0, 0, -30), now)
}

// FutureTimestampGen generates timestamps within the next 30 days.
func FutureTimestampGen() *rapid.Generator[time.Time] {
	now := time.Now()
	return TimestampGen(now, now.AddDate(0, 0, 30))
}

// SlugGen generates URL-safe slugs.
func SlugGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9-]{2,20}[a-z0-9]`)
}

// UsernameGen generates valid usernames.
func UsernameGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-z][a-z0-9_]{2,15}`)
}

// PasswordGen generates passwords meeting common requirements.
func PasswordGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		// At least 8 chars, 1 upper, 1 lower, 1 digit, 1 special
		upper := rapid.StringMatching(`[A-Z]{2}`).Draw(t, "upper")
		lower := rapid.StringMatching(`[a-z]{4}`).Draw(t, "lower")
		digit := rapid.StringMatching(`[0-9]{2}`).Draw(t, "digit")
		special := rapid.SampledFrom([]string{"!", "@", "#", "$", "%"}).Draw(t, "special")
		return upper + lower + digit + special
	})
}

// HexColorGen generates valid hex color codes.
func HexColorGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		return fmt.Sprintf("#%s", rapid.StringMatching(`[0-9A-Fa-f]{6}`).Draw(t, "hex"))
	})
}

// CorrelationIDGen generates correlation IDs (32 hex chars).
func CorrelationIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-f0-9]{32}`)
}

// TraceIDGen generates W3C trace IDs (32 hex chars).
func TraceIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-f0-9]{32}`)
}

// SpanIDGen generates W3C span IDs (16 hex chars).
func SpanIDGen() *rapid.Generator[string] {
	return rapid.StringMatching(`[a-f0-9]{16}`)
}

// JWTGen generates JWT-like tokens (not cryptographically valid).
func JWTGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		header := rapid.StringMatching(`[A-Za-z0-9_-]{20,40}`).Draw(t, "header")
		payload := rapid.StringMatching(`[A-Za-z0-9_-]{50,100}`).Draw(t, "payload")
		sig := rapid.StringMatching(`[A-Za-z0-9_-]{40,60}`).Draw(t, "sig")
		return fmt.Sprintf("%s.%s.%s", header, payload, sig)
	})
}

// SemanticVersionGen generates semantic version strings.
func SemanticVersionGen() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		major := rapid.IntRange(0, 99).Draw(t, "major")
		minor := rapid.IntRange(0, 99).Draw(t, "minor")
		patch := rapid.IntRange(0, 99).Draw(t, "patch")
		return fmt.Sprintf("%d.%d.%d", major, minor, patch)
	})
}
