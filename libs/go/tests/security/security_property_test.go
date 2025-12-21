package security_test

import (
	"testing"

	"github.com/auth-platform/libs/go/security"
	"pgregory.net/rapid"
)

// Property 12: Random Token Uniqueness
func TestRandomTokenUniqueness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(10, 100).Draw(t, "count")
		length := rapid.IntRange(16, 64).Draw(t, "length")
		seen := make(map[string]bool)

		for i := 0; i < count; i++ {
			token, err := security.GenerateToken(length)
			if err != nil {
				t.Fatalf("token generation failed: %v", err)
			}
			if seen[token] {
				t.Fatalf("duplicate token generated: %s", token)
			}
			seen[token] = true
		}
	})
}

// Property 13: Constant Time Comparison
func TestConstantTimeComparison(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		s1 := rapid.StringMatching(`[a-zA-Z0-9]{16,64}`).Draw(t, "s1")
		s2 := rapid.StringMatching(`[a-zA-Z0-9]{16,64}`).Draw(t, "s2")

		// Same strings should be equal
		if !security.ConstantTimeCompare(s1, s1) {
			t.Fatal("same string should be equal")
		}

		// Different strings should not be equal (unless randomly same)
		result := security.ConstantTimeCompare(s1, s2)
		expected := s1 == s2
		if result != expected {
			t.Fatalf("comparison mismatch: got %v, want %v", result, expected)
		}
	})
}

// Property 14: HTML Sanitization
func TestHTMLSanitization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dangerous := rapid.SampledFrom([]string{
			"<script>alert('xss')</script>",
			"<img src=x onerror=alert(1)>",
			"<a href='javascript:void(0)'>",
			"<div onclick='evil()'>",
		}).Draw(t, "dangerous")

		sanitized := security.SanitizeHTML(dangerous)

		// Should not contain raw < or >
		if containsUnescaped(sanitized, '<') || containsUnescaped(sanitized, '>') {
			t.Fatalf("HTML not properly escaped: %s", sanitized)
		}
	})
}

// Property 15: SQL Injection Detection
func TestSQLInjectionDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		injection := rapid.SampledFrom([]string{
			"'; DROP TABLE users; --",
			"1 OR 1=1",
			"1' OR '1'='1",
			"UNION SELECT * FROM passwords",
			"'; DELETE FROM users WHERE '1'='1",
		}).Draw(t, "injection")

		if !security.DetectSQLInjection(injection) {
			t.Fatalf("SQL injection not detected: %s", injection)
		}
	})
}

// Property 16: Safe Input Validation
func TestSafeInputValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		safeInput := rapid.StringMatching(`[a-zA-Z0-9 ]{5,50}`).Draw(t, "safe")

		if !security.ValidateInput(safeInput) {
			t.Fatalf("safe input rejected: %s", safeInput)
		}
	})
}

// Property 17: Shell Sanitization
func TestShellSanitization(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		dangerous := rapid.SampledFrom([]string{
			"; rm -rf /",
			"| cat /etc/passwd",
			"$(whoami)",
			"`id`",
			"&& curl evil.com",
		}).Draw(t, "dangerous")

		sanitized := security.SanitizeShell(dangerous)

		// Should not contain dangerous chars
		dangerousChars := []string{";", "&", "|", "`", "$", "(", ")"}
		for _, char := range dangerousChars {
			if containsChar(sanitized, char) {
				t.Fatalf("dangerous char %s not removed: %s", char, sanitized)
			}
		}
	})
}

// Property 18: Sensitive Data Masking
func TestSensitiveDataMasking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		length := rapid.IntRange(10, 50).Draw(t, "length")
		visibleChars := rapid.IntRange(2, 4).Draw(t, "visible")

		// Generate random string
		data := ""
		for i := 0; i < length; i++ {
			data += string(rune('a' + i%26))
		}

		masked := security.MaskSensitive(data, visibleChars)

		// Should contain asterisks
		if len(masked) != len(data) {
			t.Fatalf("masked length mismatch: got %d, want %d", len(masked), len(data))
		}

		// Middle should be masked
		if length > visibleChars*2 {
			middle := masked[visibleChars : len(masked)-visibleChars]
			for _, c := range middle {
				if c != '*' {
					t.Fatalf("middle not masked: %s", masked)
				}
			}
		}
	})
}

func containsUnescaped(s string, c rune) bool {
	for _, r := range s {
		if r == c {
			return true
		}
	}
	return false
}

func containsChar(s, char string) bool {
	for i := 0; i < len(s); i++ {
		if string(s[i]) == char {
			return true
		}
	}
	return false
}
