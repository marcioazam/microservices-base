// Package codec provides generic serialization functions for JSON and YAML.
package codec

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

// MarshalJSON marshals a value to JSON bytes.
func MarshalJSON[T any](value T) ([]byte, error) {
	return json.Marshal(value)
}

// UnmarshalJSON unmarshals JSON bytes to a value.
func UnmarshalJSON[T any](data []byte) (T, error) {
	var result T
	err := json.Unmarshal(data, &result)
	return result, err
}

// MarshalJSONIndent marshals a value to indented JSON bytes.
func MarshalJSONIndent[T any](value T, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(value, prefix, indent)
}

// MarshalYAML marshals a value to YAML bytes.
func MarshalYAML[T any](value T) ([]byte, error) {
	return yaml.Marshal(value)
}

// UnmarshalYAML unmarshals YAML bytes to a value.
func UnmarshalYAML[T any](data []byte) (T, error) {
	var result T
	err := yaml.Unmarshal(data, &result)
	return result, err
}

// RoundTripJSON marshals and unmarshals a value through JSON.
// Useful for testing serialization consistency.
func RoundTripJSON[T any](value T) (T, error) {
	data, err := MarshalJSON(value)
	if err != nil {
		var zero T
		return zero, err
	}
	return UnmarshalJSON[T](data)
}

// RoundTripYAML marshals and unmarshals a value through YAML.
// Useful for testing serialization consistency.
func RoundTripYAML[T any](value T) (T, error) {
	data, err := MarshalYAML(value)
	if err != nil {
		var zero T
		return zero, err
	}
	return UnmarshalYAML[T](data)
}

// Clone creates a deep copy of a value using JSON serialization.
func Clone[T any](value T) (T, error) {
	return RoundTripJSON(value)
}

// MustMarshalJSON marshals a value to JSON or panics.
func MustMarshalJSON[T any](value T) []byte {
	data, err := MarshalJSON(value)
	if err != nil {
		panic(err)
	}
	return data
}

// MustUnmarshalJSON unmarshals JSON or panics.
func MustUnmarshalJSON[T any](data []byte) T {
	result, err := UnmarshalJSON[T](data)
	if err != nil {
		panic(err)
	}
	return result
}

// MustMarshalYAML marshals a value to YAML or panics.
func MustMarshalYAML[T any](value T) []byte {
	data, err := MarshalYAML(value)
	if err != nil {
		panic(err)
	}
	return data
}

// MustUnmarshalYAML unmarshals YAML or panics.
func MustUnmarshalYAML[T any](data []byte) T {
	result, err := UnmarshalYAML[T](data)
	if err != nil {
		panic(err)
	}
	return result
}

// ToJSONString converts a value to a JSON string.
func ToJSONString[T any](value T) (string, error) {
	data, err := MarshalJSON(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromJSONString parses a JSON string to a value.
func FromJSONString[T any](s string) (T, error) {
	return UnmarshalJSON[T]([]byte(s))
}

// ToYAMLString converts a value to a YAML string.
func ToYAMLString[T any](value T) (string, error) {
	data, err := MarshalYAML(value)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FromYAMLString parses a YAML string to a value.
func FromYAMLString[T any](s string) (T, error) {
	return UnmarshalYAML[T]([]byte(s))
}
