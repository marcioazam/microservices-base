// Feature: go-libs-state-of-art-2025, Property 9: Generator Validity
// Validates: Requirements 11.1, 11.3
package testing_test

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/functional"
	testutil "github.com/authcorp/libs/go/src/testing"
	"pgregory.net/rapid"
)

// Property 20: Generator Validity
// Generated values satisfy their type constraints.
func TestProperty_GeneratorValidity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Option generator produces valid Options
		optGen := testutil.OptionGen(rapid.Int())
		opt := optGen.Draw(t, "option")
		
		// Must be either Some or None, not both
		if opt.IsSome() && !opt.IsSome() {
			t.Fatalf("Option cannot be both Some and None")
		}
		
		// Result generator produces valid Results
		resultGen := testutil.ResultGen(rapid.Int(), testutil.ErrorGen())
		result := resultGen.Draw(t, "result")
		
		// Must be either Ok or Err, not both
		if result.IsOk() == result.IsErr() {
			t.Fatalf("Result must be exactly one of Ok or Err")
		}
		
		// Either generator produces valid Eithers
		eitherGen := testutil.EitherGen(rapid.String(), rapid.Int())
		either := eitherGen.Draw(t, "either")
		
		// Must be either Left or Right, not both
		if either.IsLeft() == either.IsRight() {
			t.Fatalf("Either must be exactly one of Left or Right")
		}
	})
}

// Property: SomeGen always produces Some
func TestProperty_SomeGenAlwaysSome(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.SomeGen(rapid.Int())
		opt := gen.Draw(t, "some")
		
		if !opt.IsSome() {
			t.Fatalf("SomeGen should always produce Some")
		}
	})
}

// Property: NoneGen always produces None
func TestProperty_NoneGenAlwaysNone(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.NoneGen[int]()
		opt := gen.Draw(t, "none")
		
		if opt.IsSome() {
			t.Fatalf("NoneGen should always produce None")
		}
	})
}

// Property: OkGen always produces Ok
func TestProperty_OkGenAlwaysOk(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.OkGen(rapid.Int())
		result := gen.Draw(t, "ok")
		
		if !result.IsOk() {
			t.Fatalf("OkGen should always produce Ok")
		}
	})
}

// Property: ErrGen always produces Err
func TestProperty_ErrGenAlwaysErr(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.ErrGen[int](testutil.ErrorGen())
		result := gen.Draw(t, "err")
		
		if !result.IsErr() {
			t.Fatalf("ErrGen should always produce Err")
		}
	})
}

// Property: LeftGen always produces Left
func TestProperty_LeftGenAlwaysLeft(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.LeftGen[string, int](rapid.String())
		either := gen.Draw(t, "left")
		
		if !either.IsLeft() {
			t.Fatalf("LeftGen should always produce Left")
		}
	})
}

// Property: RightGen always produces Right
func TestProperty_RightGenAlwaysRight(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		gen := testutil.RightGen[string](rapid.Int())
		either := gen.Draw(t, "right")
		
		if !either.IsRight() {
			t.Fatalf("RightGen should always produce Right")
		}
	})
}

// Property: PairGen produces valid pairs
func TestProperty_PairGenValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		first := rapid.Int().Draw(t, "first")
		second := rapid.String().Draw(t, "second")
		
		pair := functional.NewPair(first, second)
		
		if pair.First != first {
			t.Fatalf("Pair First mismatch")
		}
		if pair.Second != second {
			t.Fatalf("Pair Second mismatch")
		}
	})
}

// Property: EmailGen produces valid emails
func TestProperty_EmailGenValid(t *testing.T) {
	emailRegex := regexp.MustCompile(`^[a-z][a-z0-9]{2,10}@[a-z]{3,8}\.(com|org|net|io|dev)$`)
	rapid.Check(t, func(t *rapid.T) {
		email := testutil.EmailGen().Draw(t, "email")
		if !emailRegex.MatchString(email) {
			t.Fatalf("invalid email format: %s", email)
		}
	})
}

// Property: UUIDGen produces valid UUIDs
func TestProperty_UUIDGenValid(t *testing.T) {
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	rapid.Check(t, func(t *rapid.T) {
		uuid := testutil.UUIDGen().Draw(t, "uuid")
		if !uuidRegex.MatchString(uuid) {
			t.Fatalf("invalid UUID format: %s", uuid)
		}
	})
}

// Property: ULIDGen produces valid ULIDs
func TestProperty_ULIDGenValid(t *testing.T) {
	ulidRegex := regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	rapid.Check(t, func(t *rapid.T) {
		ulid := testutil.ULIDGen().Draw(t, "ulid")
		if !ulidRegex.MatchString(ulid) {
			t.Fatalf("invalid ULID format: %s", ulid)
		}
	})
}

// Property: MoneyGen produces valid Money
func TestProperty_MoneyGenValid(t *testing.T) {
	validCurrencies := map[string]bool{"USD": true, "EUR": true, "GBP": true, "JPY": true, "CAD": true}
	rapid.Check(t, func(t *rapid.T) {
		money := testutil.MoneyGen().Draw(t, "money")
		if money.Amount < 0 {
			t.Fatalf("money amount should be non-negative: %d", money.Amount)
		}
		if !validCurrencies[money.Currency] {
			t.Fatalf("invalid currency: %s", money.Currency)
		}
	})
}

// Property: PhoneNumberGen produces valid E.164 format
func TestProperty_PhoneNumberGenValid(t *testing.T) {
	phoneRegex := regexp.MustCompile(`^\+\d{11,15}$`)
	rapid.Check(t, func(t *rapid.T) {
		phone := testutil.PhoneNumberGen().Draw(t, "phone")
		if !phoneRegex.MatchString(phone) {
			t.Fatalf("invalid phone format: %s", phone)
		}
	})
}

// Property: URLGen produces valid URLs
func TestProperty_URLGenValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		url := testutil.URLGen().Draw(t, "url")
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			t.Fatalf("URL should start with http:// or https://: %s", url)
		}
	})
}

// Property: IPAddressGen produces valid IPv4
func TestProperty_IPAddressGenValid(t *testing.T) {
	ipRegex := regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)
	rapid.Check(t, func(t *rapid.T) {
		ip := testutil.IPAddressGen().Draw(t, "ip")
		if !ipRegex.MatchString(ip) {
			t.Fatalf("invalid IP format: %s", ip)
		}
	})
}

// Property: RecentTimestampGen produces recent timestamps
func TestProperty_RecentTimestampGenValid(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		ts := testutil.RecentTimestampGen().Draw(t, "timestamp")
		now := time.Now().Add(time.Second) // Small buffer for timing
		thirtyOneDaysAgo := now.AddDate(0, 0, -31) // Extra day buffer
		if ts.Before(thirtyOneDaysAgo) || ts.After(now) {
			t.Fatalf("timestamp should be within last 30 days: %v", ts)
		}
	})
}

// Property: CorrelationIDGen produces valid format
func TestProperty_CorrelationIDGenValid(t *testing.T) {
	corrRegex := regexp.MustCompile(`^[a-f0-9]{32}$`)
	rapid.Check(t, func(t *rapid.T) {
		id := testutil.CorrelationIDGen().Draw(t, "correlationID")
		if !corrRegex.MatchString(id) {
			t.Fatalf("invalid correlation ID format: %s", id)
		}
	})
}

// Property: SemanticVersionGen produces valid semver
func TestProperty_SemanticVersionGenValid(t *testing.T) {
	semverRegex := regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	rapid.Check(t, func(t *rapid.T) {
		version := testutil.SemanticVersionGen().Draw(t, "version")
		if !semverRegex.MatchString(version) {
			t.Fatalf("invalid semver format: %s", version)
		}
	})
}
