package loggingclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/auth-platform/cache-service/internal/loggingclient"
	"github.com/auth-platform/cache-service/internal/observability"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewNoop(t *testing.T) {
	client := loggingclient.NewNoop()
	require.NotNil(t, client)

	ctx := context.Background()
	client.Info(ctx, "test message")
	client.Debug(ctx, "debug message")
	client.Warn(ctx, "warn message")
	client.Error(ctx, "error message")

	// Noop client buffers but discards on flush
	assert.False(t, client.IsCircuitOpen())
	assert.NoError(t, client.Close())
}

func TestClientWithDisabledConfig(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     10,
		FlushInterval: 100 * time.Millisecond,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, client)
	defer client.Close()

	ctx := context.Background()
	client.Info(ctx, "test message", loggingclient.String("key", "value"))

	assert.GreaterOrEqual(t, client.BufferSize(), 0)
}

func TestClientBuffering(t *testing.T) {
	cfg := loggingclient.Config{
		Enabled:       false,
		ServiceID:     "test-service",
		BatchSize:     5,
		FlushInterval: 1 * time.Hour,
		BufferSize:    100,
	}

	client, err := loggingclient.New(cfg)
	require.NoError(t, err)
	defer client.Close()

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		client.Info(ctx, "message", loggingclient.Int("index", i))
	}

	assert.Equal(t, 3, client.BufferSize())
}

func TestClientSync(t *testing.T) {
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

	ctx := context.Background()
	client.Info(ctx, "test message")

	err = client.Sync()
	assert.NoError(t, err)
	assert.Equal(t, 0, client.BufferSize())
}

func TestClientWithContextValues(t *testing.T) {
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

	ctx := context.Background()
	ctx = observability.WithCorrelationID(ctx, "corr-123")
	ctx = observability.WithRequestID(ctx, "req-456")

	client.Info(ctx, "test with context")
	assert.Equal(t, 1, client.BufferSize())
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     loggingclient.Config
		wantErr bool
	}{
		{
			name: "valid disabled config",
			cfg: loggingclient.Config{
				Enabled:       false,
				ServiceID:     "test",
				BatchSize:     10,
				FlushInterval: time.Second,
				BufferSize:    100,
			},
			wantErr: false,
		},
		{
			name: "missing address when enabled",
			cfg: loggingclient.Config{
				Enabled:       true,
				Address:       "",
				ServiceID:     "test",
				BatchSize:     10,
				FlushInterval: time.Second,
				BufferSize:    100,
			},
			wantErr: true,
		},
		{
			name: "zero batch size uses default",
			cfg: loggingclient.Config{
				Enabled:       false,
				ServiceID:     "test",
				BatchSize:     0,
				FlushInterval: time.Second,
				BufferSize:    100,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loggingclient.New(tt.cfg)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLogLevels(t *testing.T) {
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

	ctx := context.Background()

	client.Debug(ctx, "debug")
	client.Info(ctx, "info")
	client.Warn(ctx, "warn")
	client.Error(ctx, "error")

	assert.Equal(t, 4, client.BufferSize())
}

func TestFieldTypes(t *testing.T) {
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

	ctx := context.Background()

	client.Info(ctx, "test fields",
		loggingclient.String("str", "value"),
		loggingclient.Int("int", 42),
		loggingclient.Int64("int64", 123456789),
		loggingclient.Float64("float", 3.14),
		loggingclient.Bool("bool", true),
		loggingclient.Duration("dur", time.Second),
		loggingclient.Time("time", time.Now()),
		loggingclient.Error(assert.AnError),
		loggingclient.Strings("strings", []string{"a", "b"}),
		loggingclient.Any("any", map[string]int{"x": 1}),
	)

	assert.Equal(t, 1, client.BufferSize())
}
