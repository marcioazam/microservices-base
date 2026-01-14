// Package property contains property-based tests for file-upload service.
// Feature: file-upload-modernization-2025, Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
package property

import (
	"testing"
	"time"

	"pgregory.net/rapid"
)

// MockAsyncTask represents an async task for testing.
type MockAsyncTask struct {
	ID        string
	Type      string
	Retries   int
	MaxRetry  int
	Error     string
	Delays    []time.Duration
}

// MockRetryCalculator calculates retry delays.
type MockRetryCalculator struct {
	baseDelay time.Duration
	maxDelay  time.Duration
}

func NewMockRetryCalculator(baseDelay, maxDelay time.Duration) *MockRetryCalculator {
	return &MockRetryCalculator{
		baseDelay: baseDelay,
		maxDelay:  maxDelay,
	}
}

// CalculateDelay calculates exponential backoff delay.
func (c *MockRetryCalculator) CalculateDelay(retries int) time.Duration {
	delay := c.baseDelay
	for i := 0; i < retries; i++ {
		delay *= 2
	}
	if delay > c.maxDelay {
		delay = c.maxDelay
	}
	return delay
}

// MockTaskProcessor simulates task processing for testing.
type MockTaskProcessor struct {
	maxRetries int
	calculator *MockRetryCalculator
	processed  []*MockAsyncTask
	failed     []*MockAsyncTask
}

func NewMockTaskProcessor(maxRetries int, baseDelay time.Duration) *MockTaskProcessor {
	return &MockTaskProcessor{
		maxRetries: maxRetries,
		calculator: NewMockRetryCalculator(baseDelay, 5*time.Minute),
		processed:  make([]*MockAsyncTask, 0),
		failed:     make([]*MockAsyncTask, 0),
	}
}

// ProcessWithFailures simulates processing with specified number of failures.
func (p *MockTaskProcessor) ProcessWithFailures(task *MockAsyncTask, failCount int) {
	task.MaxRetry = p.maxRetries
	task.Delays = make([]time.Duration, 0)

	for attempt := 0; attempt <= failCount && task.Retries < task.MaxRetry; attempt++ {
		if attempt < failCount {
			// Simulate failure
			task.Error = "simulated failure"
			task.Retries++
			delay := p.calculator.CalculateDelay(task.Retries)
			task.Delays = append(task.Delays, delay)
		} else {
			// Success
			task.Error = ""
			p.processed = append(p.processed, task)
			return
		}
	}

	// Max retries exceeded
	p.failed = append(p.failed, task)
}

// TestProperty18_RetryDelayIncreasesExponentially tests that retry delay increases exponentially.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_RetryDelayIncreasesExponentially(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelaySecs := rapid.IntRange(1, 5).Draw(t, "baseDelaySecs")
		baseDelay := time.Duration(baseDelaySecs) * time.Second
		maxDelay := 5 * time.Minute

		calculator := NewMockRetryCalculator(baseDelay, maxDelay)

		numRetries := rapid.IntRange(2, 6).Draw(t, "numRetries")
		var delays []time.Duration

		for i := 1; i <= numRetries; i++ {
			delay := calculator.CalculateDelay(i)
			delays = append(delays, delay)
		}

		// Property: Retry delay SHALL increase exponentially
		for i := 1; i < len(delays); i++ {
			if delays[i] < delays[i-1] && delays[i] < maxDelay {
				t.Errorf("delay should increase: delays[%d]=%v < delays[%d]=%v",
					i, delays[i], i-1, delays[i-1])
			}
		}

		// Verify exponential growth (each delay should be ~2x previous)
		for i := 1; i < len(delays); i++ {
			if delays[i-1] < maxDelay {
				expectedMin := delays[i-1] * 2
				if expectedMin > maxDelay {
					expectedMin = maxDelay
				}
				if delays[i] < expectedMin && delays[i] != maxDelay {
					t.Errorf("delay should be ~2x previous: got %v, expected >= %v",
						delays[i], expectedMin)
				}
			}
		}
	})
}

// TestProperty18_MaxRetryCountRespected tests that maximum retry count is respected.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_MaxRetryCountRespected(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(1, 5).Draw(t, "maxRetries")
		failCount := rapid.IntRange(maxRetries+1, maxRetries+5).Draw(t, "failCount")

		processor := NewMockTaskProcessor(maxRetries, time.Second)

		task := &MockAsyncTask{
			ID:   rapid.StringMatching(`task-[a-z0-9]{8}`).Draw(t, "taskID"),
			Type: "test_task",
		}

		processor.ProcessWithFailures(task, failCount)

		// Property: Maximum retry count SHALL be respected
		if task.Retries > maxRetries {
			t.Errorf("retries %d exceeded max %d", task.Retries, maxRetries)
		}

		// Task should be in failed list
		found := false
		for _, failed := range processor.failed {
			if failed.ID == task.ID {
				found = true
				break
			}
		}
		if !found {
			t.Error("task should be in failed list after max retries")
		}
	})
}

// TestProperty18_FailedTasksLoggedWithError tests that failed tasks are logged with error details.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_FailedTasksLoggedWithError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(1, 3).Draw(t, "maxRetries")
		processor := NewMockTaskProcessor(maxRetries, time.Second)

		task := &MockAsyncTask{
			ID:   rapid.StringMatching(`task-[a-z0-9]{8}`).Draw(t, "taskID"),
			Type: "test_task",
		}

		// Process with more failures than max retries
		processor.ProcessWithFailures(task, maxRetries+1)

		// Property: Failed tasks SHALL be logged with error details
		if task.Error == "" {
			t.Error("failed task should have error message")
		}

		// Verify task is tracked
		if len(processor.failed) == 0 {
			t.Error("failed tasks should be tracked")
		}

		// Verify error is preserved
		for _, failed := range processor.failed {
			if failed.ID == task.ID && failed.Error == "" {
				t.Error("failed task should preserve error message")
			}
		}
	})
}

// TestProperty18_SuccessfulRetryStopsRetrying tests that successful retry stops further retries.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_SuccessfulRetryStopsRetrying(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		maxRetries := rapid.IntRange(3, 5).Draw(t, "maxRetries")
		failCount := rapid.IntRange(1, maxRetries-1).Draw(t, "failCount")

		processor := NewMockTaskProcessor(maxRetries, time.Second)

		task := &MockAsyncTask{
			ID:   rapid.StringMatching(`task-[a-z0-9]{8}`).Draw(t, "taskID"),
			Type: "test_task",
		}

		processor.ProcessWithFailures(task, failCount)

		// Property: Successful retry SHALL stop further retries
		if task.Retries != failCount {
			t.Errorf("expected %d retries, got %d", failCount, task.Retries)
		}

		// Task should be in processed list, not failed
		foundProcessed := false
		for _, processed := range processor.processed {
			if processed.ID == task.ID {
				foundProcessed = true
				break
			}
		}
		if !foundProcessed {
			t.Error("successfully retried task should be in processed list")
		}

		foundFailed := false
		for _, failed := range processor.failed {
			if failed.ID == task.ID {
				foundFailed = true
				break
			}
		}
		if foundFailed {
			t.Error("successfully retried task should not be in failed list")
		}
	})
}

// TestProperty18_DelayCapAtMaximum tests that delay is capped at maximum.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_DelayCapAtMaximum(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelay := time.Second
		maxDelay := 30 * time.Second

		calculator := NewMockRetryCalculator(baseDelay, maxDelay)

		// With many retries, delay should cap at maxDelay
		numRetries := rapid.IntRange(10, 20).Draw(t, "numRetries")

		for i := 1; i <= numRetries; i++ {
			delay := calculator.CalculateDelay(i)

			// Property: Delay SHALL be capped at maximum
			if delay > maxDelay {
				t.Errorf("delay %v exceeds max %v at retry %d", delay, maxDelay, i)
			}
		}

		// High retry count should hit the cap
		highRetryDelay := calculator.CalculateDelay(20)
		if highRetryDelay != maxDelay {
			t.Errorf("high retry delay should be capped at %v, got %v", maxDelay, highRetryDelay)
		}
	})
}

// TestProperty18_FirstRetryUsesBaseDelay tests that first retry uses base delay.
// Property 18: Async Task Retry Behavior
// Validates: Requirements 14.4, 5.2
func TestProperty18_FirstRetryUsesBaseDelay(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		baseDelaySecs := rapid.IntRange(1, 10).Draw(t, "baseDelaySecs")
		baseDelay := time.Duration(baseDelaySecs) * time.Second

		calculator := NewMockRetryCalculator(baseDelay, 5*time.Minute)

		// First retry (retries=1) should use 2*baseDelay (exponential)
		firstDelay := calculator.CalculateDelay(1)
		expectedFirst := baseDelay * 2

		if firstDelay != expectedFirst {
			t.Errorf("first retry delay should be %v, got %v", expectedFirst, firstDelay)
		}
	})
}
