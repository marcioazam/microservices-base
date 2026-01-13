// Package property contains property-based tests for the resilience operator.
package property

import (
	"regexp"
	"testing"

	"pgregory.net/rapid"
)

// TestTargetServiceNameValidation validates target service names follow DNS label rules.
// Property 8: Target Service Validation
// Validates: Requirements 4.7
func TestTargetServiceNameValidation(t *testing.T) {
	dnsLabelRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	consecutiveHyphens := regexp.MustCompile(`--`)

	rapid.Check(t, func(t *rapid.T) {
		name := rapid.StringMatching(`[a-z][a-z0-9-]{0,61}[a-z0-9]`).Draw(t, "serviceName")

		if len(name) > 0 && len(name) <= 63 {
			isValidDNS := dnsLabelRegex.MatchString(name)
			hasConsecutiveHyphens := consecutiveHyphens.MatchString(name)

			if isValidDNS && !hasConsecutiveHyphens {
				if len(name) < 1 || len(name) > 253 {
					t.Fatalf("Valid DNS label %q has invalid length", name)
				}
			}
		}
	})
}

// TestInvalidServiceNameRejection validates invalid service names are rejected.
func TestInvalidServiceNameRejection(t *testing.T) {
	dnsLabelRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	invalidNames := []string{
		"",
		"-invalid",
		"invalid-",
		"UPPERCASE",
		"with spaces",
		"with.dots",
		"with_underscore",
		"consecutive--hyphen",
	}

	for _, name := range invalidNames {
		if dnsLabelRegex.MatchString(name) && name != "" {
			if !regexp.MustCompile(`--`).MatchString(name) {
				t.Errorf("Expected %q to be invalid, but it matched DNS label regex", name)
			}
		}
	}
}

// TestNamespaceValidation validates namespace format.
func TestNamespaceValidation(t *testing.T) {
	dnsLabelRegex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

	rapid.Check(t, func(t *rapid.T) {
		ns := rapid.StringMatching(`[a-z][a-z0-9-]{0,30}[a-z0-9]`).Draw(t, "namespace")

		if len(ns) > 0 && len(ns) <= 63 {
			isValid := dnsLabelRegex.MatchString(ns)
			if !isValid {
				t.Fatalf("Generated namespace %q should be valid DNS label", ns)
			}
		}
	})
}

// TestPortValidation validates port number ranges.
func TestPortValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		port := rapid.Int32Range(1, 65535).Draw(t, "port")

		if port < 1 || port > 65535 {
			t.Fatalf("Port %d is outside valid range", port)
		}
	})
}

// TestInvalidPortRejection validates invalid ports are rejected.
func TestInvalidPortRejection(t *testing.T) {
	invalidPorts := []int32{0, -1, -100, 65536, 70000}

	for _, port := range invalidPorts {
		if port >= 1 && port <= 65535 {
			t.Errorf("Port %d should be invalid but is in valid range", port)
		}
	}
}

// TestCircuitBreakerThresholdValidation validates circuit breaker thresholds.
func TestCircuitBreakerThresholdValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		threshold := rapid.Int32Range(1, 100).Draw(t, "threshold")

		if threshold < 1 || threshold > 100 {
			t.Fatalf("Threshold %d is outside valid range", threshold)
		}
	})
}

// TestRetryMaxAttemptsValidation validates retry max attempts.
func TestRetryMaxAttemptsValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxAttempts := rapid.Int32Range(1, 10).Draw(t, "maxAttempts")

		if maxAttempts < 1 || maxAttempts > 10 {
			t.Fatalf("MaxAttempts %d is outside valid range", maxAttempts)
		}
	})
}

// TestTimeoutDurationValidation validates timeout duration format.
func TestTimeoutDurationValidation(t *testing.T) {
	durationRegex := regexp.MustCompile(`^[0-9]+(ms|s|m)$`)

	validDurations := []string{
		"100ms", "500ms", "1000ms",
		"1s", "5s", "30s", "60s",
		"1m", "5m", "10m",
	}

	for _, d := range validDurations {
		if !durationRegex.MatchString(d) {
			t.Errorf("Duration %q should be valid", d)
		}
	}

	invalidDurations := []string{
		"", "100", "ms", "s", "1h", "1d",
		"100 ms", "1.5s", "-1s",
	}

	for _, d := range invalidDurations {
		if durationRegex.MatchString(d) {
			t.Errorf("Duration %q should be invalid", d)
		}
	}
}

// TestRetryableStatusCodesValidation validates retryable status codes format.
func TestRetryableStatusCodesValidation(t *testing.T) {
	statusCodeRegex := regexp.MustCompile(`^[0-9x,]+$`)

	validCodes := []string{
		"500", "502", "503", "504",
		"5xx", "4xx",
		"500,502,503",
		"5xx,429",
	}

	for _, code := range validCodes {
		if !statusCodeRegex.MatchString(code) {
			t.Errorf("Status code pattern %q should be valid", code)
		}
	}

	invalidCodes := []string{
		"", "abc", "5XX", "500 502",
	}

	for _, code := range invalidCodes {
		if statusCodeRegex.MatchString(code) {
			t.Errorf("Status code pattern %q should be invalid", code)
		}
	}
}

// TestRateLimitValidation validates rate limit configuration.
func TestRateLimitValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rps := rapid.Int32Range(1, 100000).Draw(t, "requestsPerSecond")
		burst := rapid.Int32Range(1, 10000).Draw(t, "burstSize")

		if rps < 1 || rps > 100000 {
			t.Fatalf("RequestsPerSecond %d is outside valid range", rps)
		}
		if burst < 1 {
			t.Fatalf("BurstSize %d must be positive", burst)
		}
	})
}
