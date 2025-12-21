// Package codec provides unified encoding/decoding with multiple formats.
// This is the single authoritative codec implementation (consolidated from utils/).
package codec

import (
	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/authcorp/libs/go/src/functional"
	"gopkg.in/yaml.v3"
)

// Codec provides encoding/decoding operations.
type Codec interface {
	Encode(v any) ([]byte, error)
	Decode(data []byte, v any) error
}

// TypedCodec provides generic type-safe encoding/decoding operations.
type TypedCodec[T any] interface {
	Encode(T) ([]byte, error)
	Decode([]byte) (T, error)
}

// JSONCodec encodes/decodes using JSON.
type JSONCodec struct {
	Pretty bool
	Indent string
}

// NewJSONCodec creates a new JSON codec with default options.
func NewJSONCodec() *JSONCodec {
	return &JSONCodec{Indent: "  "}
}

// Encode encodes value to JSON.
func (c *JSONCodec) Encode(v any) ([]byte, error) {
	if c.Pretty {
		return json.MarshalIndent(v, "", c.Indent)
	}
	return json.Marshal(v)
}

// Decode decodes JSON to value.
func (c *JSONCodec) Decode(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// WithPretty enables pretty printing.
func (c *JSONCodec) WithPretty() *JSONCodec {
	c.Pretty = true
	return c
}

// WithIndent sets the indentation string.
func (c *JSONCodec) WithIndent(indent string) *JSONCodec {
	c.Indent = indent
	return c
}

// TypedJSONCodec provides type-safe JSON encoding/decoding.
type TypedJSONCodec[T any] struct {
	Pretty bool
	Indent string
}

// NewTypedJSONCodec creates a new type-safe JSON codec.
func NewTypedJSONCodec[T any]() *TypedJSONCodec[T] {
	return &TypedJSONCodec[T]{Indent: "  "}
}

// Encode encodes value to JSON.
func (c *TypedJSONCodec[T]) Encode(v T) ([]byte, error) {
	if c.Pretty {
		return json.MarshalIndent(v, "", c.Indent)
	}
	return json.Marshal(v)
}

// Decode decodes JSON to value.
func (c *TypedJSONCodec[T]) Decode(data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}

// YAMLCodec encodes/decodes using YAML.
type YAMLCodec struct {
	Indent int
}

// NewYAMLCodec creates a new YAML codec.
func NewYAMLCodec() *YAMLCodec {
	return &YAMLCodec{Indent: 2}
}

// Encode encodes value to YAML.
func (c *YAMLCodec) Encode(v any) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(c.Indent)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes YAML to value.
func (c *YAMLCodec) Decode(data []byte, v any) error {
	return yaml.Unmarshal(data, v)
}

// WithIndent sets the indentation level.
func (c *YAMLCodec) WithIndent(indent int) *YAMLCodec {
	c.Indent = indent
	return c
}

// TypedYAMLCodec provides type-safe YAML encoding/decoding.
type TypedYAMLCodec[T any] struct {
	Indent int
}

// NewTypedYAMLCodec creates a new type-safe YAML codec.
func NewTypedYAMLCodec[T any]() *TypedYAMLCodec[T] {
	return &TypedYAMLCodec[T]{Indent: 2}
}

// Encode encodes value to YAML.
func (c *TypedYAMLCodec[T]) Encode(v T) ([]byte, error) {
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(c.Indent)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decode decodes YAML to value.
func (c *TypedYAMLCodec[T]) Decode(data []byte) (T, error) {
	var v T
	err := yaml.Unmarshal(data, &v)
	return v, err
}

// Base64Codec encodes/decodes using Base64.
type Base64Codec struct {
	URLSafe bool
	Padding bool
}

// NewBase64Codec creates a new Base64 codec.
func NewBase64Codec() *Base64Codec {
	return &Base64Codec{Padding: true}
}

// Encode encodes bytes to base64 string.
func (c *Base64Codec) Encode(data []byte) string {
	return c.encoding().EncodeToString(data)
}

// Decode decodes base64 string to bytes.
func (c *Base64Codec) Decode(s string) ([]byte, error) {
	return c.encoding().DecodeString(s)
}

// WithURLSafe enables URL-safe encoding.
func (c *Base64Codec) WithURLSafe() *Base64Codec {
	c.URLSafe = true
	return c
}

// WithoutPadding disables padding.
func (c *Base64Codec) WithoutPadding() *Base64Codec {
	c.Padding = false
	return c
}

func (c *Base64Codec) encoding() *base64.Encoding {
	var enc *base64.Encoding
	if c.URLSafe {
		enc = base64.URLEncoding
	} else {
		enc = base64.StdEncoding
	}
	if !c.Padding {
		enc = enc.WithPadding(base64.NoPadding)
	}
	return enc
}

// Result-based functions for functional error handling

// EncodeResult encodes and returns Result for functional error handling.
func EncodeResult[T any](codec TypedCodec[T], v T) functional.Result[[]byte] {
	data, err := codec.Encode(v)
	if err != nil {
		return functional.Err[[]byte](err)
	}
	return functional.Ok(data)
}

// DecodeResult decodes and returns Result for functional error handling.
func DecodeResult[T any](codec TypedCodec[T], data []byte) functional.Result[T] {
	v, err := codec.Decode(data)
	if err != nil {
		return functional.Err[T](err)
	}
	return functional.Ok(v)
}

// Convenience functions

// EncodeJSON is a convenience function for JSON encoding.
func EncodeJSON(v any) ([]byte, error) {
	return NewJSONCodec().Encode(v)
}

// DecodeJSON is a convenience function for JSON decoding.
func DecodeJSON[T any](data []byte) (T, error) {
	var v T
	err := NewJSONCodec().Decode(data, &v)
	return v, err
}

// EncodeYAML is a convenience function for YAML encoding.
func EncodeYAML(v any) ([]byte, error) {
	return NewYAMLCodec().Encode(v)
}

// DecodeYAML is a convenience function for YAML decoding.
func DecodeYAML[T any](data []byte) (T, error) {
	var v T
	err := NewYAMLCodec().Decode(data, &v)
	return v, err
}

// Base64Encode encodes bytes to base64 string.
func Base64Encode(data []byte) string {
	return NewBase64Codec().Encode(data)
}

// Base64Decode decodes base64 string to bytes.
func Base64Decode(s string) ([]byte, error) {
	return NewBase64Codec().Decode(s)
}

// Base64URLEncode encodes bytes to URL-safe base64.
func Base64URLEncode(data []byte) string {
	return NewBase64Codec().WithURLSafe().Encode(data)
}

// Base64URLDecode decodes URL-safe base64 to bytes.
func Base64URLDecode(s string) ([]byte, error) {
	return NewBase64Codec().WithURLSafe().Decode(s)
}

// MustEncode encodes or panics.
func MustEncode(codec Codec, v any) []byte {
	data, err := codec.Encode(v)
	if err != nil {
		panic(err)
	}
	return data
}

// MustDecode decodes or panics.
func MustDecode[T any](codec Codec, data []byte) T {
	var v T
	if err := codec.Decode(data, &v); err != nil {
		panic(err)
	}
	return v
}
