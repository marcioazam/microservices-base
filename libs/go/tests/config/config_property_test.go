package config_test

import (
	"os"
	"testing"

	"github.com/auth-platform/libs/go/config"
	"pgregory.net/rapid"
)

// Property 15: Configuration Error Completeness
func TestConfigurationErrorCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		requiredCount := rapid.IntRange(1, 5).Draw(t, "requiredCount")
		presentCount := rapid.IntRange(0, requiredCount).Draw(t, "presentCount")

		cfg := config.New()

		// Set some required keys
		required := make([]string, requiredCount)
		for i := 0; i < requiredCount; i++ {
			required[i] = "key" + itoa(i)
		}

		// Only set some of them
		for i := 0; i < presentCount; i++ {
			cfg.Set(required[i], "value")
		}

		err := cfg.Validate(required...)

		if presentCount == requiredCount {
			if err != nil {
				t.Fatal("validation should pass when all required keys present")
			}
		} else {
			if err == nil {
				t.Fatal("validation should fail when required keys missing")
			}
			valErr, ok := err.(*config.ValidationError)
			if !ok {
				t.Fatal("error should be ValidationError")
			}
			expectedMissing := requiredCount - presentCount
			if len(valErr.MissingKeys) != expectedMissing {
				t.Fatalf("missing keys count wrong: got %d, want %d",
					len(valErr.MissingKeys), expectedMissing)
			}
		}
	})
}

// Property 16: Configuration Defaults
func TestConfigurationDefaults(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		key := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "key")
		defaultValue := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "default")
		overrideValue := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "override")

		cfg := config.New().WithDefaults(map[string]any{
			key: defaultValue,
		})

		// Should return default
		if cfg.GetString(key) != defaultValue {
			t.Fatal("should return default value")
		}

		// Override should take precedence
		cfg.Set(key, overrideValue)
		if cfg.GetString(key) != overrideValue {
			t.Fatal("override should take precedence over default")
		}
	})
}

// Property 17: Configuration Type Coercion
func TestConfigurationTypeCoercion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		intValue := rapid.IntRange(1, 10000).Draw(t, "int")

		cfg := config.New()
		cfg.Set("int_as_int", intValue)
		cfg.Set("int_as_string", itoa(intValue))
		cfg.Set("int_as_float", float64(intValue))

		// All should return same int value
		if cfg.GetInt("int_as_int") != intValue {
			t.Fatal("int should be returned as-is")
		}
		if cfg.GetInt("int_as_string") != intValue {
			t.Fatal("string should be coerced to int")
		}
		if cfg.GetInt("int_as_float") != intValue {
			t.Fatal("float should be coerced to int")
		}
	})
}

// Property 18: Configuration Bool Coercion
func TestConfigurationBoolCoercion(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		trueValue := rapid.SampledFrom([]string{"true", "1", "yes"}).Draw(t, "true")
		falseValue := rapid.SampledFrom([]string{"false", "0", "no", ""}).Draw(t, "false")

		cfg := config.New()
		cfg.Set("true_bool", true)
		cfg.Set("true_string", trueValue)
		cfg.Set("false_bool", false)
		cfg.Set("false_string", falseValue)

		if !cfg.GetBool("true_bool") {
			t.Fatal("true bool should be true")
		}
		if !cfg.GetBool("true_string") {
			t.Fatalf("'%s' should be coerced to true", trueValue)
		}
		if cfg.GetBool("false_bool") {
			t.Fatal("false bool should be false")
		}
		if cfg.GetBool("false_string") && falseValue != "" {
			t.Fatalf("'%s' should be coerced to false", falseValue)
		}
	})
}

// Property 19: Configuration Env Loading
func TestConfigurationEnvLoading(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prefix := "TEST"
		key := rapid.StringMatching(`[A-Z]{3,8}`).Draw(t, "key")
		value := rapid.StringMatching(`[a-z]{5,15}`).Draw(t, "value")

		envKey := prefix + "_" + key
		os.Setenv(envKey, value)
		defer os.Unsetenv(envKey)

		cfg := config.New().LoadEnv(prefix)

		// Key should be lowercase with dots
		configKey := stringToLower(key)
		if cfg.GetString(configKey) != value {
			t.Fatalf("env var should be loaded: key=%s, got=%s, want=%s",
				configKey, cfg.GetString(configKey), value)
		}
	})
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	result := ""
	for n > 0 {
		result = string(rune('0'+n%10)) + result
		n /= 10
	}
	return result
}

func stringToLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}
