package fault_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/authcorp/libs/go/src/fault"
	"pgregory.net/rapid"
)

// Property 2: Resilience Error Type Hierarchy
// All specific error types embed ResilienceError and satisfy error interface.
func TestProperty_ErrorTypeHierarchy(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		correlationID := rapid.String().Draw(t, "correlationID")
		
		// CircuitOpenError
		circuitErr := fault.NewCircuitOpenError(service, correlationID, time.Now(), time.Second, 0.5)
		var _ error = circuitErr
		if circuitErr.Service != service {
			t.Fatalf("CircuitOpenError service mismatch")
		}
		if circuitErr.Code != fault.ErrCodeCircuitOpen {
			t.Fatalf("CircuitOpenError code mismatch")
		}
		
		// RateLimitError
		rateErr := fault.NewRateLimitError(service, correlationID, 100, time.Minute, time.Second*30)
		var _ error = rateErr
		if rateErr.Service != service {
			t.Fatalf("RateLimitError service mismatch")
		}
		if rateErr.Code != fault.ErrCodeRateLimited {
			t.Fatalf("RateLimitError code mismatch")
		}
		
		// TimeoutError
		timeoutErr := fault.NewTimeoutError(service, correlationID, time.Second*5, time.Second*6, nil)
		var _ error = timeoutErr
		if timeoutErr.Service != service {
			t.Fatalf("TimeoutError service mismatch")
		}
		if timeoutErr.Code != fault.ErrCodeTimeout {
			t.Fatalf("TimeoutError code mismatch")
		}
		
		// BulkheadFullError
		bulkheadErr := fault.NewBulkheadFullError(service, correlationID, 10, 5, 15)
		var _ error = bulkheadErr
		if bulkheadErr.Service != service {
			t.Fatalf("BulkheadFullError service mismatch")
		}
		if bulkheadErr.Code != fault.ErrCodeBulkheadFull {
			t.Fatalf("BulkheadFullError code mismatch")
		}
	})
}

// Property 3: Error Type Checking Functions
// IsXxx functions correctly identify error types.
func TestProperty_ErrorTypeChecking(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		correlationID := rapid.String().Draw(t, "correlationID")
		
		circuitErr := fault.NewCircuitOpenError(service, correlationID, time.Now(), time.Second, 0.5)
		rateErr := fault.NewRateLimitError(service, correlationID, 100, time.Minute, time.Second*30)
		timeoutErr := fault.NewTimeoutError(service, correlationID, time.Second*5, time.Second*6, nil)
		bulkheadErr := fault.NewBulkheadFullError(service, correlationID, 10, 5, 15)
		
		// Each check function should only match its type
		if !fault.IsCircuitOpen(circuitErr) {
			t.Fatalf("IsCircuitOpen should match CircuitOpenError")
		}
		if fault.IsCircuitOpen(rateErr) {
			t.Fatalf("IsCircuitOpen should not match RateLimitError")
		}
		
		if !fault.IsRateLimited(rateErr) {
			t.Fatalf("IsRateLimited should match RateLimitError")
		}
		if fault.IsRateLimited(circuitErr) {
			t.Fatalf("IsRateLimited should not match CircuitOpenError")
		}
		
		if !fault.IsTimeout(timeoutErr) {
			t.Fatalf("IsTimeout should match TimeoutError")
		}
		if fault.IsTimeout(circuitErr) {
			t.Fatalf("IsTimeout should not match CircuitOpenError")
		}
		
		if !fault.IsBulkheadFull(bulkheadErr) {
			t.Fatalf("IsBulkheadFull should match BulkheadFullError")
		}
		if fault.IsBulkheadFull(circuitErr) {
			t.Fatalf("IsBulkheadFull should not match CircuitOpenError")
		}
	})
}

// Property 4: Error JSON Round Trip
// Serializing and deserializing preserves error data.
func TestProperty_ErrorJSONRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9-]*`).Draw(t, "service")
		correlationID := rapid.StringMatching(`[a-zA-Z0-9-]+`).Draw(t, "correlationID")
		limit := rapid.IntRange(1, 10000).Draw(t, "limit")
		
		original := fault.NewRateLimitError(service, correlationID, limit, time.Minute, time.Second*30)
		
		// Serialize
		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("Failed to marshal: %v", err)
		}
		
		// Deserialize to map to verify JSON structure
		var restored map[string]interface{}
		if err := json.Unmarshal(data, &restored); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}
		
		// Verify fields preserved in JSON
		if restored["code"] != string(original.Code) {
			t.Fatalf("Code mismatch: %v != %v", restored["code"], original.Code)
		}
		if restored["service"] != original.Service {
			t.Fatalf("Service mismatch: %v != %v", restored["service"], original.Service)
		}
		if restored["correlation_id"] != original.CorrelationID {
			t.Fatalf("CorrelationID mismatch: %v != %v", restored["correlation_id"], original.CorrelationID)
		}
		// JSON numbers are float64
		if int(restored["limit"].(float64)) != original.Limit {
			t.Fatalf("Limit mismatch: %v != %v", restored["limit"], original.Limit)
		}
	})
}

// Property: GetErrorCode extracts correct code
func TestProperty_GetErrorCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		service := rapid.String().Draw(t, "service")
		correlationID := rapid.String().Draw(t, "correlationID")
		
		testCases := []struct {
			err      error
			expected fault.ErrorCode
		}{
			{fault.NewCircuitOpenError(service, correlationID, time.Now(), time.Second, 0.5), fault.ErrCodeCircuitOpen},
			{fault.NewRateLimitError(service, correlationID, 100, time.Minute, time.Second), fault.ErrCodeRateLimited},
			{fault.NewTimeoutError(service, correlationID, time.Second, time.Second, nil), fault.ErrCodeTimeout},
			{fault.NewBulkheadFullError(service, correlationID, 10, 5, 15), fault.ErrCodeBulkheadFull},
		}
		
		for _, tc := range testCases {
			code, ok := fault.GetErrorCode(tc.err)
			if !ok {
				t.Fatalf("GetErrorCode should return true for resilience errors")
			}
			if code != tc.expected {
				t.Fatalf("GetErrorCode returned %v, expected %v", code, tc.expected)
			}
		}
	})
}
