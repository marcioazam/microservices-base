package property

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/auth-platform/file-upload/internal/observability"
	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: file-upload-service, Property 13: Structured Logging Format
// Validates: Requirements 11.2
// For any API request, the generated log entry SHALL be valid JSON containing
// at minimum: timestamp, level, correlation_id, tenant_id, and message fields.

func TestLoggerStructuredFormat(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	parameters.MaxSize = 50

	properties := gopter.NewProperties(parameters)

	// Property: All log entries are valid JSON
	properties.Property("log entries are valid JSON", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			return json.Unmarshal(buf.Bytes(), &logEntry) == nil
		},
		gen.AlphaString(),
	))

	// Property: Log entries contain timestamp field
	properties.Property("log entries contain timestamp", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			_, hasTime := logEntry["time"]
			return hasTime
		},
		gen.AlphaString(),
	))

	// Property: Log entries contain level field
	properties.Property("log entries contain level", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			_, hasLevel := logEntry["level"]
			return hasLevel
		},
		gen.AlphaString(),
	))

	// Property: Log entries contain message field
	properties.Property("log entries contain message", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			msg, hasMessage := logEntry["message"]
			if !hasMessage {
				return false
			}
			return msg == message
		},
		gen.AlphaString(),
	))

	// Property: Context values are included in log entries
	properties.Property("context values are included", prop.ForAll(
		func(correlationID, tenantID, userID, message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")

			ctx := context.Background()
			ctx = observability.WithCorrelationID(ctx, correlationID)
			ctx = observability.WithTenantID(ctx, tenantID)
			ctx = observability.WithUserID(ctx, userID)

			logger.WithContext(ctx).Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			// Check correlation_id if provided
			if correlationID != "" {
				if cid, ok := logEntry["correlation_id"].(string); !ok || cid != correlationID {
					return false
				}
			}

			// Check tenant_id if provided
			if tenantID != "" {
				if tid, ok := logEntry["tenant_id"].(string); !ok || tid != tenantID {
					return false
				}
			}

			// Check user_id if provided
			if userID != "" {
				if uid, ok := logEntry["user_id"].(string); !ok || uid != userID {
					return false
				}
			}

			return true
		},
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
		gen.AlphaString(),
	))

	// Property: Timestamp is valid ISO 8601 format
	properties.Property("timestamp is valid ISO 8601", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			timeStr, ok := logEntry["time"].(string)
			if !ok {
				return false
			}

			_, err := time.Parse(time.RFC3339Nano, timeStr)
			return err == nil
		},
		gen.AlphaString(),
	))

	// Property: Level values are valid
	properties.Property("level values are valid", prop.ForAll(
		func(message string) bool {
			validLevels := map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
				"fatal": true,
			}

			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			var logEntry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
				return false
			}

			level, ok := logEntry["level"].(string)
			if !ok {
				return false
			}

			return validLevels[level]
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}

// TestLoggerLevelFiltering tests that log levels are properly filtered
func TestLoggerLevelFiltering(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Debug messages are not logged at info level
	properties.Property("debug filtered at info level", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Debug(message)

			return buf.Len() == 0
		},
		gen.AlphaString(),
	))

	// Property: Info messages are logged at info level
	properties.Property("info logged at info level", prop.ForAll(
		func(message string) bool {
			var buf bytes.Buffer
			logger := observability.NewLoggerWithWriter(&buf, "info")
			logger.Info(message)

			return buf.Len() > 0
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t)
}
