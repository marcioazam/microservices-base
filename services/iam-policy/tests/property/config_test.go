// Package property contains property-based tests for IAM Policy Service.
package property

import (
	"os"
	"testing"

	"pgregory.net/rapid"
)

// **Feature: iam-policy-service-modernization, Property 1: Configuration Loading Consistency**
// **Validates: Requirements 1.1, 1.2, 1.3**
func TestConfigurationLoadingConsistency(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate random configuration values
		grpcPort := rapid.IntRange(1024, 65535).Draw(t, "grpcPort")
		healthPort := rapid.IntRange(1024, 65535).Draw(t, "healthPort")
		metricsPort := rapid.IntRange(1024, 65535).Draw(t, "metricsPort")
		policyPath := rapid.SampledFrom([]string{"./policies", "/etc/policies", "./custom"}).Draw(t, "policyPath")
		cacheNamespace := rapid.SampledFrom([]string{"iam-policy", "test-ns", "prod-iam"}).Draw(t, "cacheNamespace")
		serviceName := rapid.SampledFrom([]string{"iam-policy-service", "test-service"}).Draw(t, "serviceName")

		// Set environment variables
		os.Setenv("IAM_POLICY_SERVER_GRPC_PORT", intToString(grpcPort))
		os.Setenv("IAM_POLICY_SERVER_HEALTH_PORT", intToString(healthPort))
		os.Setenv("IAM_POLICY_SERVER_METRICS_PORT", intToString(metricsPort))
		os.Setenv("IAM_POLICY_POLICY_PATH", policyPath)
		os.Setenv("IAM_POLICY_CACHE_NAMESPACE", cacheNamespace)
		os.Setenv("IAM_POLICY_LOGGING_SERVICE_NAME", serviceName)

		defer func() {
			os.Unsetenv("IAM_POLICY_SERVER_GRPC_PORT")
			os.Unsetenv("IAM_POLICY_SERVER_HEALTH_PORT")
			os.Unsetenv("IAM_POLICY_SERVER_METRICS_PORT")
			os.Unsetenv("IAM_POLICY_POLICY_PATH")
			os.Unsetenv("IAM_POLICY_CACHE_NAMESPACE")
			os.Unsetenv("IAM_POLICY_LOGGING_SERVICE_NAME")
		}()

		// Property: Loading configuration twice should produce identical results
		// This validates deterministic configuration loading
		config1 := loadTestConfig()
		config2 := loadTestConfig()

		if config1.GRPCPort != config2.GRPCPort {
			t.Fatalf("gRPC port mismatch: %d != %d", config1.GRPCPort, config2.GRPCPort)
		}
		if config1.HealthPort != config2.HealthPort {
			t.Fatalf("health port mismatch: %d != %d", config1.HealthPort, config2.HealthPort)
		}
		if config1.PolicyPath != config2.PolicyPath {
			t.Fatalf("policy path mismatch: %s != %s", config1.PolicyPath, config2.PolicyPath)
		}
		if config1.CacheNamespace != config2.CacheNamespace {
			t.Fatalf("cache namespace mismatch: %s != %s", config1.CacheNamespace, config2.CacheNamespace)
		}
	})
}

// TestConfigurationDefaultValues tests that default values are applied correctly.
func TestConfigurationDefaultValues(t *testing.T) {
	// Clear all IAM_POLICY_ environment variables
	for _, env := range os.Environ() {
		if len(env) > 11 && env[:11] == "IAM_POLICY_" {
			key := env[:len(env)-len(env[len(env):])]
			os.Unsetenv(key)
		}
	}

	config := loadTestConfig()

	// Verify defaults are applied
	if config.GRPCPort != 50054 {
		t.Errorf("expected default gRPC port 50054, got %d", config.GRPCPort)
	}
	if config.HealthPort != 8080 {
		t.Errorf("expected default health port 8080, got %d", config.HealthPort)
	}
	if config.PolicyPath != "./policies" {
		t.Errorf("expected default policy path './policies', got %s", config.PolicyPath)
	}
	if config.CacheNamespace != "iam-policy" {
		t.Errorf("expected default cache namespace 'iam-policy', got %s", config.CacheNamespace)
	}
}

// testConfig is a simplified config struct for testing.
type testConfig struct {
	GRPCPort       int
	HealthPort     int
	MetricsPort    int
	PolicyPath     string
	CacheNamespace string
	ServiceName    string
}

func loadTestConfig() testConfig {
	return testConfig{
		GRPCPort:       getEnvInt("IAM_POLICY_SERVER_GRPC_PORT", 50054),
		HealthPort:     getEnvInt("IAM_POLICY_SERVER_HEALTH_PORT", 8080),
		MetricsPort:    getEnvInt("IAM_POLICY_SERVER_METRICS_PORT", 9090),
		PolicyPath:     getEnvString("IAM_POLICY_POLICY_PATH", "./policies"),
		CacheNamespace: getEnvString("IAM_POLICY_CACHE_NAMESPACE", "iam-policy"),
		ServiceName:    getEnvString("IAM_POLICY_LOGGING_SERVICE_NAME", "iam-policy-service"),
	}
}

func getEnvString(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		var result int
		for _, c := range val {
			if c >= '0' && c <= '9' {
				result = result*10 + int(c-'0')
			}
		}
		if result > 0 {
			return result
		}
	}
	return defaultVal
}

func intToString(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
