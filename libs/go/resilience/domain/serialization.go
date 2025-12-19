package domain

import (
	"encoding/json"
	"time"

	"gopkg.in/yaml.v3"
)

// MarshalJSON implements json.Marshaler for CircuitState.
func (s CircuitState) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler for CircuitState.
func (s *CircuitState) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "closed":
		*s = CircuitClosed
	case "open":
		*s = CircuitOpen
	case "half-open":
		*s = CircuitHalfOpen
	default:
		*s = CircuitClosed
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler for CircuitState.
func (s CircuitState) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for CircuitState.
func (s *CircuitState) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return err
	}
	switch str {
	case "closed":
		*s = CircuitClosed
	case "open":
		*s = CircuitOpen
	case "half-open":
		*s = CircuitHalfOpen
	default:
		*s = CircuitClosed
	}
	return nil
}

// MarshalJSON implements json.Marshaler for HealthStatus.
func (s HealthStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalJSON implements json.Unmarshaler for HealthStatus.
func (s *HealthStatus) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	switch str {
	case "healthy":
		*s = Healthy
	case "degraded":
		*s = Degraded
	case "unhealthy":
		*s = Unhealthy
	default:
		*s = Healthy
	}
	return nil
}

// MarshalYAML implements yaml.Marshaler for HealthStatus.
func (s HealthStatus) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler for HealthStatus.
func (s *HealthStatus) UnmarshalYAML(node *yaml.Node) error {
	var str string
	if err := node.Decode(&str); err != nil {
		return err
	}
	switch str {
	case "healthy":
		*s = Healthy
	case "degraded":
		*s = Degraded
	case "unhealthy":
		*s = Unhealthy
	default:
		*s = Healthy
	}
	return nil
}

// durationJSON is a helper for JSON duration marshaling.
type durationJSON time.Duration

func (d durationJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

func (d *durationJSON) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	dur, err := time.ParseDuration(str)
	if err != nil {
		return err
	}
	*d = durationJSON(dur)
	return nil
}

// ToJSON serializes a ResiliencePolicy to JSON.
func (p *ResiliencePolicy) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON deserializes a ResiliencePolicy from JSON.
func FromJSON(data []byte) (*ResiliencePolicy, error) {
	var p ResiliencePolicy
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ToYAML serializes a ResiliencePolicy to YAML.
func (p *ResiliencePolicy) ToYAML() ([]byte, error) {
	return yaml.Marshal(p)
}

// FromYAML deserializes a ResiliencePolicy from YAML.
func FromYAML(data []byte) (*ResiliencePolicy, error) {
	var p ResiliencePolicy
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// CircuitBreakerConfigToJSON serializes a CircuitBreakerConfig to JSON.
func CircuitBreakerConfigToJSON(c *CircuitBreakerConfig) ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// CircuitBreakerConfigFromJSON deserializes a CircuitBreakerConfig from JSON.
func CircuitBreakerConfigFromJSON(data []byte) (*CircuitBreakerConfig, error) {
	var c CircuitBreakerConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// RetryConfigToJSON serializes a RetryConfig to JSON.
func RetryConfigToJSON(c *RetryConfig) ([]byte, error) {
	return json.MarshalIndent(c, "", "  ")
}

// RetryConfigFromJSON deserializes a RetryConfig from JSON.
func RetryConfigFromJSON(data []byte) (*RetryConfig, error) {
	var c RetryConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, err
	}
	return &c, nil
}
