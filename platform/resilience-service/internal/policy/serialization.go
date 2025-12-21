package policy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/auth-platform/libs/go/resilience"
)

// MarshalPolicy serializes a policy to JSON.
func MarshalPolicy(policy *resilience.ResiliencePolicy) ([]byte, error) {
	return json.Marshal(policy)
}

// UnmarshalPolicy deserializes a policy from JSON.
func UnmarshalPolicy(data []byte) (*resilience.ResiliencePolicy, error) {
	var policy resilience.ResiliencePolicy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("unmarshal policy: %w", err)
	}
	return &policy, nil
}

// PrettyPrint returns a human-readable representation of the policy.
func PrettyPrint(policy *resilience.ResiliencePolicy) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Policy: %s (v%d)\n", policy.Name, policy.Version))
	sb.WriteString(strings.Repeat("-", 40) + "\n")

	if policy.CircuitBreaker != nil {
		sb.WriteString("Circuit Breaker:\n")
		sb.WriteString(fmt.Sprintf("  Failure Threshold: %d\n", policy.CircuitBreaker.FailureThreshold))
		sb.WriteString(fmt.Sprintf("  Success Threshold: %d\n", policy.CircuitBreaker.SuccessThreshold))
		sb.WriteString(fmt.Sprintf("  Timeout:           %v\n", policy.CircuitBreaker.Timeout))
		sb.WriteString("\n")
	}

	if policy.Retry != nil {
		sb.WriteString("Retry:\n")
		sb.WriteString(fmt.Sprintf("  Max Attempts:   %d\n", policy.Retry.MaxAttempts))
		sb.WriteString(fmt.Sprintf("  Base Delay:     %v\n", policy.Retry.BaseDelay))
		sb.WriteString(fmt.Sprintf("  Max Delay:      %v\n", policy.Retry.MaxDelay))
		sb.WriteString(fmt.Sprintf("  Multiplier:     %.2f\n", policy.Retry.Multiplier))
		sb.WriteString(fmt.Sprintf("  Jitter:         %.0f%%\n", policy.Retry.JitterPercent*100))
		sb.WriteString("\n")
	}

	if policy.Timeout != nil {
		sb.WriteString("Timeout:\n")
		sb.WriteString(fmt.Sprintf("  Default: %v\n", policy.Timeout.Default))
		if policy.Timeout.Max > 0 {
			sb.WriteString(fmt.Sprintf("  Max:     %v\n", policy.Timeout.Max))
		}
		sb.WriteString("\n")
	}

	if policy.RateLimit != nil {
		sb.WriteString("Rate Limit:\n")
		sb.WriteString(fmt.Sprintf("  Algorithm:  %s\n", policy.RateLimit.Algorithm))
		sb.WriteString(fmt.Sprintf("  Limit:      %d\n", policy.RateLimit.Limit))
		sb.WriteString(fmt.Sprintf("  Window:     %v\n", policy.RateLimit.Window))
		sb.WriteString(fmt.Sprintf("  Burst Size: %d\n", policy.RateLimit.BurstSize))
		sb.WriteString("\n")
	}

	if policy.Bulkhead != nil {
		sb.WriteString("Bulkhead:\n")
		sb.WriteString(fmt.Sprintf("  Max Concurrent: %d\n", policy.Bulkhead.MaxConcurrent))
		sb.WriteString(fmt.Sprintf("  Max Queue:      %d\n", policy.Bulkhead.MaxQueue))
		sb.WriteString(fmt.Sprintf("  Queue Timeout:  %v\n", policy.Bulkhead.QueueTimeout))
	}

	return sb.String()
}
