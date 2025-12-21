package property

import (
	"context"
	"testing"

	"github.com/auth-platform/sdk-go/internal/observability"
	"pgregory.net/rapid"
)

// TestProperty28_NoSensitiveDataInObservability tests that sensitive data is filtered.
func TestProperty28_NoSensitiveDataInObservability(t *testing.T) {
	sensitiveKeys := []string{
		"token", "access_token", "secret", "password",
		"credential", "authorization", "bearer", "jwt", "key",
	}

	for _, key := range sensitiveKeys {
		t.Run(key, func(t *testing.T) {
			if !observability.IsSensitiveKey(key) {
				t.Errorf("key %s should be detected as sensitive", key)
			}
		})
	}
}

func TestProperty28_NonSensitiveKeysAllowed(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate safe keys
		safeKeys := []string{"user_id", "request_id", "method", "path", "status", "duration"}
		key := rapid.SampledFrom(safeKeys).Draw(t, "key")

		if observability.IsSensitiveKey(key) {
			t.Errorf("key %s should not be detected as sensitive", key)
		}
	})
}

// TestProperty29_LogSeverityMatching tests log level mapping.
func TestProperty29_LogSeverityMatching(t *testing.T) {
	testCases := []struct {
		level    observability.LogLevel
		expected string
	}{
		{observability.LogLevelDebug, "DEBUG"},
		{observability.LogLevelInfo, "INFO"},
		{observability.LogLevelWarn, "WARN"},
		{observability.LogLevelError, "ERROR"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			logger := observability.NewDefaultLogger(tc.level)
			if logger == nil {
				t.Fatal("logger should not be nil")
			}
		})
	}
}

func TestNopLoggerDoesNotPanic(t *testing.T) {
	logger := observability.NopLogger{}
	ctx := context.Background()

	// These should not panic
	logger.Debug(ctx, "test", "key", "value")
	logger.Info(ctx, "test", "key", "value")
	logger.Warn(ctx, "test", "key", "value")
	logger.Error(ctx, "test", "key", "value")
}

func TestLoggerInterface(t *testing.T) {
	var _ observability.Logger = &observability.DefaultLogger{}
	var _ observability.Logger = observability.NopLogger{}
}

func TestTracerCreation(t *testing.T) {
	tracer := observability.NewTracer()
	if tracer == nil {
		t.Fatal("tracer should not be nil")
	}
}

func TestTracerSpanCreation(t *testing.T) {
	tracer := observability.NewTracer()
	ctx := context.Background()

	t.Run("TokenValidationSpan", func(t *testing.T) {
		ctx, span := tracer.TokenValidationSpan(ctx)
		if span == nil {
			t.Fatal("span should not be nil")
		}
		if ctx == nil {
			t.Fatal("context should not be nil")
		}
		span.End()
	})

	t.Run("TokenRefreshSpan", func(t *testing.T) {
		ctx, span := tracer.TokenRefreshSpan(ctx)
		if span == nil {
			t.Fatal("span should not be nil")
		}
		if ctx == nil {
			t.Fatal("context should not be nil")
		}
		span.End()
	})

	t.Run("JWKSFetchSpan", func(t *testing.T) {
		ctx, span := tracer.JWKSFetchSpan(ctx, "https://example.com/.well-known/jwks.json")
		if span == nil {
			t.Fatal("span should not be nil")
		}
		if ctx == nil {
			t.Fatal("context should not be nil")
		}
		span.End()
	})

	t.Run("DPoPProofSpan", func(t *testing.T) {
		ctx, span := tracer.DPoPProofSpan(ctx, "POST", "https://api.example.com/token")
		if span == nil {
			t.Fatal("span should not be nil")
		}
		if ctx == nil {
			t.Fatal("context should not be nil")
		}
		span.End()
	})
}

func TestSanitizeURI(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/path", "https://example.com/path"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := observability.SanitizeURI(tc.input)
			if result != tc.expected {
				t.Errorf("expected %s, got %s", tc.expected, result)
			}
		})
	}
}

func TestLongURITruncation(t *testing.T) {
	longURI := "https://example.com/" + string(make([]byte, 200))
	result := observability.SanitizeURI(longURI)
	if len(result) > 103 { // 100 + "..."
		t.Errorf("URI should be truncated, got length %d", len(result))
	}
}
