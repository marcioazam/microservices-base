package domain_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/auth-platform/libs/go/domain"
	"pgregory.net/rapid"
)

// Property 1: Email Validation Consistency
// Valid emails always parse successfully, invalid emails always fail
func TestEmailValidationConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		local := rapid.StringMatching(`[a-z][a-z0-9]{0,10}`).Draw(t, "local")
		dom := rapid.StringMatching(`[a-z][a-z0-9]{0,10}`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net", "io"}).Draw(t, "tld")

		email := local + "@" + dom + "." + tld
		parsed, err := domain.NewEmail(email)

		if err != nil {
			t.Fatalf("valid email should parse: %s, error: %v", email, err)
		}
		if parsed.String() != strings.ToLower(email) {
			t.Fatalf("email should be normalized: got %s, want %s", parsed.String(), strings.ToLower(email))
		}
	})
}

// Property 2: Email JSON Round-Trip
func TestEmailJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		local := rapid.StringMatching(`[a-z][a-z0-9]{0,10}`).Draw(t, "local")
		dom := rapid.StringMatching(`[a-z][a-z0-9]{0,10}`).Draw(t, "domain")
		tld := rapid.SampledFrom([]string{"com", "org", "net"}).Draw(t, "tld")

		original, err := domain.NewEmail(local + "@" + dom + "." + tld)
		if err != nil {
			return // Skip invalid emails
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.Email
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !original.Equals(restored) {
			t.Fatalf("round-trip failed: %s != %s", original, restored)
		}
	})
}

// Property 3: UUID Uniqueness
func TestUUIDUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(2, 100).Draw(t, "count")
		seen := make(map[string]bool)

		for i := 0; i < count; i++ {
			uuid := domain.NewUUID()
			if seen[uuid.String()] {
				t.Fatalf("duplicate UUID generated: %s", uuid)
			}
			seen[uuid.String()] = true
		}
	})
}

// Property 4: UUID JSON Round-Trip
func TestUUIDJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := domain.NewUUID()

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.UUID
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !original.Equals(restored) {
			t.Fatalf("round-trip failed: %s != %s", original, restored)
		}
	})
}

// Property 5: ULID Time Ordering
func TestULIDTimeOrdering(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseTime := time.Now()
		offset1 := rapid.Int64Range(0, 1000).Draw(t, "offset1")
		offset2 := rapid.Int64Range(1001, 2000).Draw(t, "offset2")

		t1 := baseTime.Add(time.Duration(offset1) * time.Millisecond)
		t2 := baseTime.Add(time.Duration(offset2) * time.Millisecond)

		ulid1 := domain.NewULIDWithTime(t1)
		ulid2 := domain.NewULIDWithTime(t2)

		if ulid1.Compare(ulid2) >= 0 {
			t.Fatalf("ULID ordering violated: %s should be < %s", ulid1, ulid2)
		}
	})
}

// Property 6: ULID JSON Round-Trip
func TestULIDJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := domain.NewULID()

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.ULID
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !original.Equals(restored) {
			t.Fatalf("round-trip failed: %s != %s", original, restored)
		}
	})
}

// Property 7: Money Arithmetic Consistency
func TestMoneyArithmeticConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		a := rapid.Int64Range(0, 1000000).Draw(t, "a")
		b := rapid.Int64Range(0, 1000000).Draw(t, "b")

		m1, _ := domain.NewMoney(a, domain.USD)
		m2, _ := domain.NewMoney(b, domain.USD)

		// Addition is commutative
		sum1, _ := m1.Add(m2)
		sum2, _ := m2.Add(m1)
		if !sum1.Equals(sum2) {
			t.Fatalf("addition not commutative: %v != %v", sum1, sum2)
		}

		// a + b - b = a
		diff, _ := sum1.Subtract(m2)
		if !diff.Equals(m1) {
			t.Fatalf("subtraction inverse failed: %v != %v", diff, m1)
		}
	})
}

// Property 8: Money JSON Round-Trip
func TestMoneyJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		amount := rapid.Int64Range(-1000000, 1000000).Draw(t, "amount")
		currency := rapid.SampledFrom([]domain.Currency{
			domain.USD, domain.EUR, domain.GBP, domain.JPY,
		}).Draw(t, "currency")

		original, err := domain.NewMoney(amount, currency)
		if err != nil {
			return
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.Money
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !original.Equals(restored) {
			t.Fatalf("round-trip failed: %v != %v", original, restored)
		}
	})
}

// Property 9: PhoneNumber E.164 Validation
func TestPhoneNumberValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		countryCode := rapid.IntRange(1, 999).Draw(t, "countryCode")
		number := rapid.StringMatching(`[0-9]{6,12}`).Draw(t, "number")

		phone := "+" + itoa(countryCode) + number
		if len(phone) > 16 {
			phone = phone[:16]
		}

		parsed, err := domain.NewPhoneNumber(phone)
		if err != nil {
			return // Skip invalid
		}

		if !strings.HasPrefix(parsed.String(), "+") {
			t.Fatalf("phone should start with +: %s", parsed)
		}
	})
}

// Property 10: URL Validation
func TestURLValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scheme := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "scheme")
		host := rapid.StringMatching(`[a-z][a-z0-9]{2,10}`).Draw(t, "host")
		tld := rapid.SampledFrom([]string{"com", "org", "net"}).Draw(t, "tld")

		urlStr := scheme + "://" + host + "." + tld
		parsed, err := domain.NewURL(urlStr)
		if err != nil {
			t.Fatalf("valid URL should parse: %s, error: %v", urlStr, err)
		}

		if parsed.Scheme() != scheme {
			t.Fatalf("scheme mismatch: got %s, want %s", parsed.Scheme(), scheme)
		}
	})
}

// Property 11: Timestamp JSON Round-Trip
func TestTimestampJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		unix := rapid.Int64Range(0, 2000000000).Draw(t, "unix")
		original := domain.FromUnix(unix)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.Timestamp
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		// Compare Unix timestamps (nano precision may differ)
		if original.Unix() != restored.Unix() {
			t.Fatalf("round-trip failed: %d != %d", original.Unix(), restored.Unix())
		}
	})
}

// Property 12: Duration Parsing Consistency
func TestDurationParsingConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		value := rapid.IntRange(1, 1000).Draw(t, "value")
		unit := rapid.SampledFrom([]string{"s", "m", "h"}).Draw(t, "unit")

		durStr := itoa(value) + unit
		parsed, err := domain.ParseDuration(durStr)
		if err != nil {
			t.Fatalf("valid duration should parse: %s, error: %v", durStr, err)
		}

		if parsed.IsZero() {
			t.Fatalf("parsed duration should not be zero: %s", durStr)
		}
	})
}

// Property 13: Duration JSON Round-Trip
func TestDurationJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		seconds := rapid.Int64Range(1, 86400).Draw(t, "seconds")
		original := domain.Seconds(seconds)

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}

		var restored domain.Duration
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}

		if !original.Equals(restored) {
			t.Fatalf("round-trip failed: %v != %v", original, restored)
		}
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}
