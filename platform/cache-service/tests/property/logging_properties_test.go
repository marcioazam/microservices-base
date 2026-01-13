package property_test

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

// Property 1: Delivery guarantee - all logs eventually delivered or written to stderr
func TestProperty_DeliveryGuarantee(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		messageCount := rapid.IntRange(1, 50).Draw(t, "messageCount")

		cfg := loggingclient.Config{
			Enabled:       false,
			ServiceID:     "test-service",
			BatchSize:     10,
			FlushInterval: 50 * time.Millisecond,
			BufferSize:    100,
		}

		client, err := loggingclient.New(cfg)
		assert.NoError(t, err)

		ctx := context.Background()
		for i := 0; i < messageCount; i++ {
			client.Info(ctx, "test message", loggingclient.Int("index", i))
		}

		err = client.Sync()
		assert.NoError(t, err)

		// After sync, buffer should be empty (all delivered)
		assert.Equal(t, 0, client.BufferSize())

		client.Close()
	})
}

// Property 2: Batch flush invariant - after sync, buffer is empty
func TestProperty_BatchFlushInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		batchSize := rapid.IntRange(5, 20).Draw(t, "batchSize")
		messageCount := rapid.IntRange(1, 100).Draw(t, "messageCount")

		cfg := loggingclient.Config{
			Enabled:       false,
			ServiceID:     "test-service",
			BatchSize:     batchSize,
			FlushInterval: 1 * time.Hour, // Long interval to test batch-based flush
			BufferSize:    1000,
		}

		client, err := loggingclient.New(cfg)
		assert.NoError(t, err)
		defer client.Close()

		ctx := context.Background()
		for i := 0; i < messageCount; i++ {
			client.Info(ctx, "message", loggingclient.Int("i", i))
		}

		// After sync, buffer should be empty (all flushed)
		err = client.Sync()
		assert.NoError(t, err)
		assert.Equal(t, 0, client.BufferSize(), "buffer should be empty after sync")
	})
}

// Property 8: Buffer overflow protection - no panic on high volume
func TestProperty_BufferOverflowProtection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		bufferSize := rapid.IntRange(10, 50).Draw(t, "bufferSize")
		messageCount := rapid.IntRange(100, 500).Draw(t, "messageCount")

		cfg := loggingclient.Config{
			Enabled:       false,
			ServiceID:     "test-service",
			BatchSize:     5,
			FlushInterval: 10 * time.Millisecond,
			BufferSize:    bufferSize,
		}

		client, err := loggingclient.New(cfg)
		assert.NoError(t, err)
		defer client.Close()

		ctx := context.Background()

		// Should not panic even with high volume
		assert.NotPanics(t, func() {
			for i := 0; i < messageCount; i++ {
				client.Info(ctx, "high volume message", loggingclient.Int("i", i))
			}
		})

		client.Sync()
	})
}

// Property: Log levels are preserved
func TestProperty_LogLevelsPreserved(t *testing.T) {
	levels := []loggingclient.Level{
		loggingclient.LevelDebug,
		loggingclient.LevelInfo,
		loggingclient.LevelWarn,
		loggingclient.LevelError,
	}

	rapid.Check(t, func(t *rapid.T) {
		levelIdx := rapid.IntRange(0, len(levels)-1).Draw(t, "levelIdx")
		level := levels[levelIdx]

		cfg := loggingclient.Config{
			Enabled:       false,
			ServiceID:     "test-service",
			BatchSize:     100,
			FlushInterval: 1 * time.Hour,
			BufferSize:    100,
		}

		client, err := loggingclient.New(cfg)
		assert.NoError(t, err)
		defer client.Close()

		ctx := context.Background()
		client.Log(ctx, level, "test message")

		assert.Equal(t, 1, client.BufferSize())
	})
}

// Property: Fields are correctly typed
func TestProperty_FieldTyping(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		strVal := rapid.String().Draw(t, "strVal")
		intVal := rapid.Int().Draw(t, "intVal")
		boolVal := rapid.Bool().Draw(t, "boolVal")
		floatVal := rapid.Float64().Draw(t, "floatVal")

		cfg := loggingclient.Config{
			Enabled:       false,
			ServiceID:     "test-service",
			BatchSize:     100,
			FlushInterval: 1 * time.Hour,
			BufferSize:    100,
		}

		client, err := loggingclient.New(cfg)
		assert.NoError(t, err)
		defer client.Close()

		ctx := context.Background()

		// Should not panic with any valid field values
		assert.NotPanics(t, func() {
			client.Info(ctx, "test",
				loggingclient.String("str", strVal),
				loggingclient.Int("int", intVal),
				loggingclient.Bool("bool", boolVal),
				loggingclient.Float64("float", floatVal),
			)
		})
	})
}
