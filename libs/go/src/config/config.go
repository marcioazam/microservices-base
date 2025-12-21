// Package config provides configuration loading and validation.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds configuration values.
type Config struct {
	values   map[string]any
	defaults map[string]any
}

// New creates a new empty Config.
func New() *Config {
	return &Config{
		values:   make(map[string]any),
		defaults: make(map[string]any),
	}
}

// WithDefaults sets default values.
func (c *Config) WithDefaults(defaults map[string]any) *Config {
	for k, v := range defaults {
		c.defaults[k] = v
	}
	return c
}

// LoadFile loads configuration from a JSON or YAML file.
func (c *Config) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var values map[string]any
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &values); err != nil {
			return fmt.Errorf("failed to parse JSON: %w", err)
		}
	}

	for k, v := range values {
		c.values[k] = v
	}
	return nil
}

// LoadEnv loads configuration from environment variables with prefix.
func (c *Config) LoadEnv(prefix string) *Config {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]
		if prefix != "" && !strings.HasPrefix(key, prefix+"_") {
			continue
		}
		// Convert ENV_VAR_NAME to env.var.name
		configKey := strings.ToLower(strings.ReplaceAll(
			strings.TrimPrefix(key, prefix+"_"), "_", "."))
		c.values[configKey] = value
	}
	return c
}

// Set sets a configuration value.
func (c *Config) Set(key string, value any) {
	c.values[key] = value
}

// Get returns a configuration value.
func (c *Config) Get(key string) (any, bool) {
	if v, ok := c.values[key]; ok {
		return v, true
	}
	if v, ok := c.defaults[key]; ok {
		return v, true
	}
	return nil, false
}

// GetString returns a string configuration value.
func (c *Config) GetString(key string) string {
	v, ok := c.Get(key)
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// GetInt returns an int configuration value.
func (c *Config) GetInt(key string) int {
	v, ok := c.Get(key)
	if !ok {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

// GetBool returns a bool configuration value.
func (c *Config) GetBool(key string) bool {
	v, ok := c.Get(key)
	if !ok {
		return false
	}
	switch val := v.(type) {
	case bool:
		return val
	case string:
		return val == "true" || val == "1" || val == "yes"
	}
	return false
}

// GetStringSlice returns a string slice configuration value.
func (c *Config) GetStringSlice(key string) []string {
	v, ok := c.Get(key)
	if !ok {
		return nil
	}
	switch val := v.(type) {
	case []string:
		return val
	case []any:
		result := make([]string, len(val))
		for i, item := range val {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case string:
		return strings.Split(val, ",")
	}
	return nil
}

// Validate checks that required keys are present.
func (c *Config) Validate(required ...string) error {
	var missing []string
	for _, key := range required {
		if _, ok := c.Get(key); !ok {
			missing = append(missing, key)
		}
	}
	if len(missing) > 0 {
		return &ValidationError{MissingKeys: missing}
	}
	return nil
}

// ValidationError represents configuration validation errors.
type ValidationError struct {
	MissingKeys []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("missing required config keys: %s", strings.Join(e.MissingKeys, ", "))
}

// All returns all configuration values.
func (c *Config) All() map[string]any {
	result := make(map[string]any)
	for k, v := range c.defaults {
		result[k] = v
	}
	for k, v := range c.values {
		result[k] = v
	}
	return result
}
