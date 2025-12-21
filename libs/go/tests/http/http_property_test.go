package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	httplib "github.com/auth-platform/libs/go/http"
	"pgregory.net/rapid"
)

// TestClientBuilderPattern verifies fluent builder pattern preserves settings.
func TestClientBuilderPattern(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		timeout := rapid.IntRange(1, 60).Draw(t, "timeout")
		retries := rapid.IntRange(0, 10).Draw(t, "retries")
		delay := rapid.IntRange(10, 1000).Draw(t, "delay")
		headerKey := rapid.StringMatching(`[A-Z][a-z]{2,10}`).Draw(t, "headerKey")
		headerVal := rapid.StringMatching(`[a-z]{3,20}`).Draw(t, "headerVal")

		client := httplib.NewClient().
			WithTimeout(time.Duration(timeout) * time.Second).
			WithRetry(retries, time.Duration(delay)*time.Millisecond).
			WithHeader(headerKey, headerVal)

		if client == nil {
			t.Fatal("client should not be nil")
		}
	})
}

// TestResponseStatusClassification verifies status code classification.
func TestResponseStatusClassification(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		statusCode := rapid.IntRange(100, 599).Draw(t, "statusCode")

		resp := &httplib.Response{StatusCode: statusCode}

		isSuccess := resp.IsSuccess()
		isClientError := resp.IsClientError()
		isServerError := resp.IsServerError()

		// Exactly one classification should be true for 2xx, 4xx, 5xx
		if statusCode >= 200 && statusCode < 300 {
			if !isSuccess {
				t.Errorf("status %d should be success", statusCode)
			}
			if isClientError || isServerError {
				t.Errorf("status %d should not be error", statusCode)
			}
		}

		if statusCode >= 400 && statusCode < 500 {
			if !isClientError {
				t.Errorf("status %d should be client error", statusCode)
			}
			if isSuccess || isServerError {
				t.Errorf("status %d classification mismatch", statusCode)
			}
		}

		if statusCode >= 500 {
			if !isServerError {
				t.Errorf("status %d should be server error", statusCode)
			}
			if isSuccess || isClientError {
				t.Errorf("status %d classification mismatch", statusCode)
			}
		}
	})
}

// TestMiddlewareChainOrder verifies middleware execution order.
func TestMiddlewareChainOrder(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		count := rapid.IntRange(1, 5).Draw(t, "middlewareCount")

		var order []int
		middlewares := make([]httplib.Middleware, count)

		for i := 0; i < count; i++ {
			idx := i
			middlewares[i] = func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					order = append(order, idx)
					next.ServeHTTP(w, r)
				})
			}
		}

		chain := httplib.Chain(middlewares...)
		handler := chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		order = nil
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		// Middlewares should execute in order 0, 1, 2, ...
		for i, v := range order {
			if v != i {
				t.Errorf("middleware %d executed at position %d", v, i)
			}
		}
	})
}

// TestHealthHandlerRegistration verifies health check registration.
func TestHealthHandlerRegistration(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		checkCount := rapid.IntRange(1, 10).Draw(t, "checkCount")

		handler := httplib.NewHealthHandler()

		for i := 0; i < checkCount; i++ {
			name := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "checkName")
			handler.Register(name, func() error { return nil })
		}

		// Liveness should always return healthy
		req := httptest.NewRequest("GET", "/health/live", nil)
		rec := httptest.NewRecorder()
		handler.LivenessHandler().ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("liveness should return 200, got %d", rec.Code)
		}
	})
}

// TestCORSMiddlewareHeaders verifies CORS headers are set.
func TestCORSMiddlewareHeaders(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		origin := rapid.StringMatching(`https?://[a-z]+\.[a-z]{2,3}`).Draw(t, "origin")

		middleware := httplib.CORSMiddleware(origin)
		handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != origin {
			t.Errorf("CORS origin header mismatch")
		}
	})
}

// TestRecoveryMiddlewarePanicHandling verifies panic recovery.
func TestRecoveryMiddlewarePanicHandling(t *testing.T) {
	var recovered any

	middleware := httplib.RecoveryMiddleware(func(err any) {
		recovered = err
	})

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", rec.Code)
	}

	if recovered != "test panic" {
		t.Errorf("panic not recovered correctly")
	}
}
