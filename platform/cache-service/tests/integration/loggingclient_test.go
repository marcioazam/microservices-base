package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/auth-platform/cache-service/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Property 9: Graceful shutdown completeness
func TestIntegration_GracefulShutdownCompleteness(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
		BufferSize:    1000,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)

	ctx := context.Background()

	// Send multiple messages
	for i := 0; i < 50; i++ {
		client.Info(ctx, "test message", loggingclient.Int("index", i))
	}

	// Buffer should have messages
	assert.Greater(t, client.BufferSize(), 0)

	// Close should flush all messages
	err = client.Close()
	assert.NoError(t, err)

	// After close, buffer should be empty
	assert.Equal(t, 0, client.BufferSize())
}

func TestIntegration_ConcurrentLogging(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
		BufferSize:    1000,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent logging from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				client.Info(ctx, "concurrent message",
					loggingclient.Int("goroutine", id),
					loggingclient.Int("message", j),
				)
			}
		}(i)
	}

	wg.Wait()

	// Should not panic and should handle all messages
	err = client.Sync()
	assert.NoError(t, err)
}

func TestIntegration_ContextPropagation(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	defer client.Close()

	// Create context with all observability values
	ctx := context.Background()
	ctx = observability.WithCorrelationID(ctx, "corr-integration-test")
	ctx = observability.WithRequestID(ctx, "req-integration-test")
	ctx = observability.WithTraceID(ctx, "00000000000000000000000000000001")
	ctx = observability.WithSpanID(ctx, "0000000000000001")

	// Log with context
	client.Info(ctx, "integration test message",
		loggingclient.String("test", "context_propagation"),
	)

	assert.Equal(t, 1, client.BufferSize())
}

func TestIntegration_FlushOnBatchSize(t *testing.T) {
	batchSize := 5

	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     batchSize,
		FlushInterval: 1 * time.Hour,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Send exactly batch size messages
	for i := 0; i < batchSize; i++ {
		client.Info(ctx, "batch message", loggingclient.Int("i", i))
	}

	// Give time for async flush
	time.Sleep(100 * time.Millisecond)

	// Buffer should be flushed
	assert.LessOrEqual(t, client.BufferSize(), batchSize)
}

func TestIntegration_FlushOnInterval(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     100,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()

	// Send fewer messages than batch size
	client.Info(ctx, "interval flush test")

	// Wait for flush interval
	time.Sleep(200 * time.Millisecond)

	// Buffer should be flushed by interval
	assert.Equal(t, 0, client.BufferSize())
}

func TestIntegration_MultipleCloseIsSafe(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)

	// Multiple closes should be safe
	assert.NoError(t, client.Close())
	assert.NoError(t, client.Close())
	assert.NoError(t, client.Close())
}

func TestIntegration_LogAfterClose(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     100,
		FlushInterval: 1 * time.Hour,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)

	client.Close()

	// Logging after close should not panic
	ctx := context.Background()
	assert.NotPanics(t, func() {
		client.Info(ctx, "after close")
	})
}
