package crypto_test

import (
	"testing"
	"time"

	"github.com/auth-platform/iam-policy-service/internal/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCryptoMetrics_RecordEncrypt(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record successful encryption
	metrics.RecordEncrypt("success", 10*time.Millisecond)
	metrics.RecordEncrypt("success", 15*time.Millisecond)
	metrics.RecordEncrypt("error", 5*time.Millisecond)

	// Verify counter
	counter := metrics.GetEncryptTotal()
	successCount := testutil.ToFloat64(counter.WithLabelValues("success"))
	errorCount := testutil.ToFloat64(counter.WithLabelValues("error"))

	if successCount != 2 {
		t.Errorf("expected 2 successful encrypts, got %f", successCount)
	}
	if errorCount != 1 {
		t.Errorf("expected 1 error encrypt, got %f", errorCount)
	}
}

func TestCryptoMetrics_RecordDecrypt(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record decryption operations
	metrics.RecordDecrypt("success", 8*time.Millisecond)
	metrics.RecordDecrypt("error", 3*time.Millisecond)

	// Verify counter
	counter := metrics.GetDecryptTotal()
	successCount := testutil.ToFloat64(counter.WithLabelValues("success"))
	errorCount := testutil.ToFloat64(counter.WithLabelValues("error"))

	if successCount != 1 {
		t.Errorf("expected 1 successful decrypt, got %f", successCount)
	}
	if errorCount != 1 {
		t.Errorf("expected 1 error decrypt, got %f", errorCount)
	}
}

func TestCryptoMetrics_RecordSign(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record signing operations
	metrics.RecordSign("success", 20*time.Millisecond)
	metrics.RecordSign("success", 25*time.Millisecond)
	metrics.RecordSign("success", 18*time.Millisecond)

	// Verify counter
	counter := metrics.GetSignTotal()
	successCount := testutil.ToFloat64(counter.WithLabelValues("success"))

	if successCount != 3 {
		t.Errorf("expected 3 successful signs, got %f", successCount)
	}
}

func TestCryptoMetrics_RecordVerify(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record verification operations
	metrics.RecordVerify("success", 12*time.Millisecond)
	metrics.RecordVerify("invalid", 10*time.Millisecond)

	// Verify counter
	counter := metrics.GetVerifyTotal()
	successCount := testutil.ToFloat64(counter.WithLabelValues("success"))
	invalidCount := testutil.ToFloat64(counter.WithLabelValues("invalid"))

	if successCount != 1 {
		t.Errorf("expected 1 successful verify, got %f", successCount)
	}
	if invalidCount != 1 {
		t.Errorf("expected 1 invalid verify, got %f", invalidCount)
	}
}

func TestCryptoMetrics_RecordFallback(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record fallbacks
	metrics.RecordFallback()
	metrics.RecordFallback()
	metrics.RecordFallback()

	// Verify we can record without panic
	// Note: fallbackTotal is not exposed via getter, so we just verify no panic
}

func TestCryptoMetrics_RecordError(t *testing.T) {
	registry := prometheus.NewRegistry()
	metrics := crypto.NewMetrics(registry)

	// Record errors
	metrics.RecordError("ENCRYPTION_FAILED")
	metrics.RecordError("DECRYPTION_FAILED")
	metrics.RecordError("ENCRYPTION_FAILED")

	// Verify we can record without panic
}

func TestCryptoMetrics_NilSafe(t *testing.T) {
	var metrics *crypto.Metrics

	// All methods should be nil-safe
	metrics.RecordEncrypt("success", time.Millisecond)
	metrics.RecordDecrypt("success", time.Millisecond)
	metrics.RecordSign("success", time.Millisecond)
	metrics.RecordVerify("success", time.Millisecond)
	metrics.RecordError("TEST")
	metrics.RecordFallback()

	// No panic means success
}
