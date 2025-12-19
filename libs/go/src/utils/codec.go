package utils

import (
	"encoding/base64"
	"encoding/json"

	"github.com/authcorp/libs/go/src/functional"
)

// Codec provides encoding/decoding operations.
type Codec[T any] interface {
	Encode(T) ([]byte, error)
	Decode([]byte) (T, error)
}

// JSONCodec encodes/decodes using JSON.
type JSONCodec[T any] struct{}

// NewJSONCodec creates a new JSON codec.
func NewJSONCodec[T any]() *JSONCodec[T] {
	return &JSONCodec[T]{}
}

// Encode encodes value to JSON.
func (c *JSONCodec[T]) Encode(v T) ([]byte, error) {
	return json.Marshal(v)
}

// Decode decodes JSON to value.
func (c *JSONCodec[T]) Decode(data []byte) (T, error) {
	var v T
	err := json.Unmarshal(data, &v)
	return v, err
}

// EncodeResult encodes and returns Result.
func EncodeResult[T any](codec Codec[T], v T) functional.Result[[]byte] {
	data, err := codec.Encode(v)
	if err != nil {
		return functional.Err[[]byte](err)
	}
	return functional.Ok(data)
}

// DecodeResult decodes and returns Result.
func DecodeResult[T any](codec Codec[T], data []byte) functional.Result[T] {
	v, err := codec.Decode(data)
	if err != nil {
		return functional.Err[T](err)
	}
	return functional.Ok(v)
}

// Base64Encode encodes bytes to base64 string.
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// Base64Decode decodes base64 string to bytes.
func Base64Decode(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// Base64URLEncode encodes bytes to URL-safe base64.
func Base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

// Base64URLDecode decodes URL-safe base64 to bytes.
func Base64URLDecode(s string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(s)
}
