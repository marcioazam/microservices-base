package property

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/auth-platform/platform/resilience-service/internal/application/services"
	"github.com/auth-platform/platform/resilience-service/tests/testutil"
	"pgregory.net/rapid"
)

// TestServiceCreationWithoutGlobalsProperty tests service creation.
func TestServiceCreationWithoutGlobalsProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		tracer := testutil.GetMockTracer()

		service1 := services.NewResilienceService(
			&testutil.MockResilienceExecutor{},
			&testutil.MockMetricsRecorder{},
			logger,
			tracer,
		)

		service2 := services.NewResilienceService(
			&testutil.MockResilienceExecutor{},
			&testutil.MockMetricsRecorder{},
			logger,
			tracer,
		)

		if service1 == service2 {
			t.Fatal("Services should be different instances")
		}

		ctx := context.Background()

		err1 := service1.Execute(ctx, "test-policy", func() error { return nil })
		err2 := service2.Execute(ctx, "test-policy", func() error { return nil })

		if err1 != nil {
			t.Fatalf("Service1 execution failed: %v", err1)
		}

		if err2 != nil {
			t.Fatalf("Service2 execution failed: %v", err2)
		}
	})
}

// TestResilienceServiceExecutionProperty tests resilience service execution.
func TestResilienceServiceExecutionProperty(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		policyName := rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9_-]{1,20}`).Draw(t, "policy_name")
		shouldSucceed := rapid.Bool().Draw(t, "should_succeed")

		logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
		tracer := testutil.GetMockTracer()
		executor := &testutil.MockResilienceExecutor{}
		metrics := &testutil.MockMetricsRecorder{}

		service := services.NewResilienceService(executor, metrics, logger, tracer)

		ctx := context.Background()

		var operationCalled bool
		operation := func() error {
			operationCalled = true
			if shouldSucceed {
				return nil
			}
			return errors.New("test error")
		}

		err := service.Execute(ctx, policyName, operation)

		if !operationCalled {
			t.Fatal("Operation should have been called")
		}

		if shouldSucceed && err != nil {
			t.Fatalf("Expected success but got error: %v", err)
		}

		if !shouldSucceed && err == nil {
			t.Fatal("Expected error but got success")
		}
	})
}
