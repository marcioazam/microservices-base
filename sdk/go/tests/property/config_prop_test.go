package property

import (
	"os"
	"testing"
	"time"

	authplatform "github.com/auth-platform/sdk-go"
	"pgregory.net/rapid"
)

// TestProperty23_RequiredFieldValidation validates Property 23:
// For any Config with empty BaseURL or empty ClientID, Validate()
// SHALL return an SDKError with code INVALID_CONFIG.
// **Validates: Requirements 11.1, 11.2**
func TestProperty23_RequiredFieldValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		testCase := rapid.IntRange(0, 2).Draw(t, "testCase")

		config := authplatform.Config{
			BaseURL:      "https://auth.example.com",
			ClientID:     "test-client",
			Timeout:      30 * time.Second,
			JWKSCacheTTL: time.Hour,
			MaxRetries:   3,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
		}

		switch testCase {
		case 0:
			config.BaseURL = ""
		case 1:
			config.ClientID = ""
		case 2:
			config.BaseURL = ""
			config.ClientID = ""
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for missing required field")
			return
		}

		if !authplatform.IsInvalidConfig(err) {
			t.Errorf("expected INVALID_CONFIG error, got: %v", err)
		}
	})
}

// TestProperty24_TimeoutValidation validates Property 24:
// For any Config with non-positive Timeout, Validate()
// SHALL return an SDKError with code INVALID_CONFIG.
// **Validates: Requirements 11.3**
func TestProperty24_TimeoutValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		invalidTimeout := rapid.Int64Range(-1000000000000, 0).Draw(t, "invalidTimeout")

		config := authplatform.Config{
			BaseURL:      "https://auth.example.com",
			ClientID:     "test-client",
			Timeout:      time.Duration(invalidTimeout),
			JWKSCacheTTL: time.Hour,
			MaxRetries:   3,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
		}

		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for non-positive timeout")
			return
		}

		if !authplatform.IsInvalidConfig(err) {
			t.Errorf("expected INVALID_CONFIG error, got: %v", err)
		}
	})
}


// TestProperty25_CacheTTLValidation validates Property 25:
// For any Config with JWKSCacheTTL less than 1 minute or greater than 24 hours,
// Validate() SHALL return an SDKError with code INVALID_CONFIG.
// **Validates: Requirements 11.4**
func TestProperty25_CacheTTLValidation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		testCase := rapid.IntRange(0, 1).Draw(t, "testCase")

		config := authplatform.Config{
			BaseURL:      "https://auth.example.com",
			ClientID:     "test-client",
			Timeout:      30 * time.Second,
			JWKSCacheTTL: time.Hour,
			MaxRetries:   3,
			BaseDelay:    time.Second,
			MaxDelay:     30 * time.Second,
		}

		switch testCase {
		case 0:
			invalidTTL := rapid.Int64Range(1, int64(time.Minute)-1).Draw(t, "tooSmallTTL")
			config.JWKSCacheTTL = time.Duration(invalidTTL)
		case 1:
			invalidTTL := rapid.Int64Range(int64(24*time.Hour)+1, int64(48*time.Hour)).Draw(t, "tooLargeTTL")
			config.JWKSCacheTTL = time.Duration(invalidTTL)
		}

		err := config.Validate()
		if err == nil {
			t.Errorf("expected validation error for invalid JWKSCacheTTL: %v", config.JWKSCacheTTL)
			return
		}

		if !authplatform.IsInvalidConfig(err) {
			t.Errorf("expected INVALID_CONFIG error, got: %v", err)
		}
	})
}

// TestProperty26_DefaultValuesApplication validates Property 26:
// For any Config with zero-value optional fields, after applying defaults,
// Timeout SHALL be 30s, JWKSCacheTTL SHALL be 1h, MaxRetries SHALL be 3.
// **Validates: Requirements 11.5**
func TestProperty26_DefaultValuesApplication(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		config := authplatform.Config{
			BaseURL:  "https://auth.example.com",
			ClientID: "test-client",
		}

		config.ApplyDefaults()

		if config.Timeout != 30*time.Second {
			t.Errorf("expected Timeout 30s, got %v", config.Timeout)
		}
		if config.JWKSCacheTTL != time.Hour {
			t.Errorf("expected JWKSCacheTTL 1h, got %v", config.JWKSCacheTTL)
		}
		if config.MaxRetries != 3 {
			t.Errorf("expected MaxRetries 3, got %d", config.MaxRetries)
		}
		if config.BaseDelay != time.Second {
			t.Errorf("expected BaseDelay 1s, got %v", config.BaseDelay)
		}
		if config.MaxDelay != 30*time.Second {
			t.Errorf("expected MaxDelay 30s, got %v", config.MaxDelay)
		}
	})
}


// TestProperty27_EnvironmentVariableConfiguration validates Property 27:
// For any environment variable AUTH_PLATFORM_* set, loading config from
// environment SHALL populate the corresponding field.
// **Validates: Requirements 11.6**
func TestProperty27_EnvironmentVariableConfiguration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseURL := rapid.StringMatching(`https://[a-z]+\.example\.com`).Draw(t, "baseURL")
		clientID := rapid.StringMatching(`[a-z]{5,10}-client`).Draw(t, "clientID")
		clientSecret := rapid.StringMatching(`secret-[a-z0-9]{8}`).Draw(t, "clientSecret")

		os.Setenv("AUTH_PLATFORM_BASE_URL", baseURL)
		os.Setenv("AUTH_PLATFORM_CLIENT_ID", clientID)
		os.Setenv("AUTH_PLATFORM_CLIENT_SECRET", clientSecret)
		os.Setenv("AUTH_PLATFORM_TIMEOUT", "45s")
		os.Setenv("AUTH_PLATFORM_JWKS_CACHE_TTL", "2h")
		os.Setenv("AUTH_PLATFORM_MAX_RETRIES", "5")
		os.Setenv("AUTH_PLATFORM_DPOP_ENABLED", "true")

		defer func() {
			os.Unsetenv("AUTH_PLATFORM_BASE_URL")
			os.Unsetenv("AUTH_PLATFORM_CLIENT_ID")
			os.Unsetenv("AUTH_PLATFORM_CLIENT_SECRET")
			os.Unsetenv("AUTH_PLATFORM_TIMEOUT")
			os.Unsetenv("AUTH_PLATFORM_JWKS_CACHE_TTL")
			os.Unsetenv("AUTH_PLATFORM_MAX_RETRIES")
			os.Unsetenv("AUTH_PLATFORM_DPOP_ENABLED")
		}()

		config := authplatform.LoadFromEnv()

		if config.BaseURL != baseURL {
			t.Errorf("expected BaseURL %q, got %q", baseURL, config.BaseURL)
		}
		if config.ClientID != clientID {
			t.Errorf("expected ClientID %q, got %q", clientID, config.ClientID)
		}
		if config.ClientSecret != clientSecret {
			t.Errorf("expected ClientSecret %q, got %q", clientSecret, config.ClientSecret)
		}
		if config.Timeout != 45*time.Second {
			t.Errorf("expected Timeout 45s, got %v", config.Timeout)
		}
		if config.JWKSCacheTTL != 2*time.Hour {
			t.Errorf("expected JWKSCacheTTL 2h, got %v", config.JWKSCacheTTL)
		}
		if config.MaxRetries != 5 {
			t.Errorf("expected MaxRetries 5, got %d", config.MaxRetries)
		}
		if !config.DPoPEnabled {
			t.Error("expected DPoPEnabled true")
		}
	})
}

// TestValidConfigPasses tests that valid configurations pass validation.
func TestValidConfigPasses(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := rapid.Int64Range(int64(time.Second), int64(5*time.Minute)).Draw(t, "timeout")
		cacheTTL := rapid.Int64Range(int64(time.Minute), int64(24*time.Hour)).Draw(t, "cacheTTL")
		maxRetries := rapid.IntRange(0, 10).Draw(t, "maxRetries")
		baseDelay := rapid.Int64Range(int64(100*time.Millisecond), int64(5*time.Second)).Draw(t, "baseDelay")
		maxDelay := rapid.Int64Range(int64(5*time.Second), int64(time.Minute)).Draw(t, "maxDelay")

		config := authplatform.Config{
			BaseURL:      "https://auth.example.com",
			ClientID:     "test-client",
			Timeout:      time.Duration(timeout),
			JWKSCacheTTL: time.Duration(cacheTTL),
			MaxRetries:   maxRetries,
			BaseDelay:    time.Duration(baseDelay),
			MaxDelay:     time.Duration(maxDelay),
		}

		err := config.Validate()
		if err != nil {
			t.Errorf("expected valid config to pass validation, got: %v", err)
		}
	})
}
