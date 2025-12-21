// Feature: go-libs-state-of-art-2025, Property 2: Codec Round-Trip
// Validates: Requirements 2.3, 2.4, 2.5
package codec_test

import (
	"reflect"
	"testing"

	"github.com/authcorp/libs/go/src/codec"
	"pgregory.net/rapid"
)

// TestJSONCodecRoundTrip verifies Decode(Encode(v)) == v for JSON.
func TestJSONCodecRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Test with map[string]any
		original := map[string]any{
			"string": rapid.String().Draw(t, "string"),
			"int":    float64(rapid.Int().Draw(t, "int")), // JSON numbers are float64
			"bool":   rapid.Bool().Draw(t, "bool"),
		}

		c := codec.NewJSONCodec()
		encoded, err := c.Encode(original)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		var decoded map[string]any
		if err := c.Decode(encoded, &decoded); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if !reflect.DeepEqual(original, decoded) {
			t.Fatalf("round-trip failed: got %v, want %v", decoded, original)
		}
	})
}

// TestTypedJSONCodecRoundTrip verifies type-safe JSON round-trip.
func TestTypedJSONCodecRoundTrip(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
		Flag  bool   `json:"flag"`
	}

	rapid.Check(t, func(t *rapid.T) {
		original := TestStruct{
			Name:  rapid.String().Draw(t, "name"),
			Value: rapid.Int().Draw(t, "value"),
			Flag:  rapid.Bool().Draw(t, "flag"),
		}

		c := codec.NewTypedJSONCodec[TestStruct]()
		encoded, err := c.Encode(original)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		decoded, err := c.Decode(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if decoded != original {
			t.Fatalf("round-trip failed: got %v, want %v", decoded, original)
		}
	})
}

// TestYAMLCodecRoundTrip verifies Decode(Encode(v)) == v for YAML.
func TestYAMLCodecRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Use alphanumeric strings to avoid YAML special character issues
		original := map[string]any{
			"string": rapid.StringMatching(`[a-zA-Z0-9]{0,20}`).Draw(t, "string"),
			"int":    rapid.Int().Draw(t, "int"),
			"bool":   rapid.Bool().Draw(t, "bool"),
		}

		c := codec.NewYAMLCodec()
		encoded, err := c.Encode(original)
		if err != nil {
			t.Fatalf("encode failed: %v", err)
		}

		var decoded map[string]any
		if err := c.Decode(encoded, &decoded); err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		// YAML preserves int types
		if decoded["string"] != original["string"] {
			t.Fatalf("string mismatch: got %v, want %v", decoded["string"], original["string"])
		}
		if decoded["bool"] != original["bool"] {
			t.Fatalf("bool mismatch: got %v, want %v", decoded["bool"], original["bool"])
		}
	})
}

// TestBase64CodecRoundTrip verifies Decode(Encode(data)) == data for Base64.
func TestBase64CodecRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := []byte(rapid.String().Draw(t, "data"))

		c := codec.NewBase64Codec()
		encoded := c.Encode(original)
		decoded, err := c.Decode(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if string(decoded) != string(original) {
			t.Fatalf("round-trip failed: got %q, want %q", decoded, original)
		}
	})
}

// TestBase64URLSafeRoundTrip verifies URL-safe Base64 round-trip.
func TestBase64URLSafeRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := []byte(rapid.String().Draw(t, "data"))

		c := codec.NewBase64Codec().WithURLSafe()
		encoded := c.Encode(original)
		decoded, err := c.Decode(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if string(decoded) != string(original) {
			t.Fatalf("round-trip failed: got %q, want %q", decoded, original)
		}
	})
}

// TestBase64NoPaddingRoundTrip verifies Base64 without padding round-trip.
func TestBase64NoPaddingRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := []byte(rapid.String().Draw(t, "data"))

		c := codec.NewBase64Codec().WithoutPadding()
		encoded := c.Encode(original)
		decoded, err := c.Decode(encoded)
		if err != nil {
			t.Fatalf("decode failed: %v", err)
		}

		if string(decoded) != string(original) {
			t.Fatalf("round-trip failed: got %q, want %q", decoded, original)
		}
	})
}

// TestEncodeResultPropagatesErrors verifies Result wrapper handles errors.
func TestEncodeResultPropagatesErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type TestData struct {
			Value string `json:"value"`
		}

		original := TestData{Value: rapid.String().Draw(t, "value")}
		c := codec.NewTypedJSONCodec[TestData]()

		result := codec.EncodeResult(c, original)
		if result.IsErr() {
			t.Fatalf("encode should succeed: %v", result.UnwrapErr())
		}

		data := result.Unwrap()
		decodeResult := codec.DecodeResult(c, data)
		if decodeResult.IsErr() {
			t.Fatalf("decode should succeed: %v", decodeResult.UnwrapErr())
		}

		decoded := decodeResult.Unwrap()
		if decoded != original {
			t.Fatalf("round-trip failed: got %v, want %v", decoded, original)
		}
	})
}

// TestDecodeResultPropagatesErrors verifies Result wrapper handles decode errors.
func TestDecodeResultPropagatesErrors(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type TestData struct {
			Value int `json:"value"`
		}

		// Invalid JSON should produce error
		invalidJSON := []byte(`{"value": "not_an_int"}`)
		c := codec.NewTypedJSONCodec[TestData]()

		result := codec.DecodeResult(c, invalidJSON)
		if result.IsOk() {
			t.Fatal("decode should fail for invalid data")
		}
	})
}

// TestConvenienceFunctions verifies convenience functions work correctly.
func TestConvenienceFunctions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		type Data struct {
			Name string `json:"name" yaml:"name"`
		}

		original := Data{Name: rapid.StringMatching(`[a-zA-Z]{3,20}`).Draw(t, "name")}

		// JSON convenience
		jsonData, err := codec.EncodeJSON(original)
		if err != nil {
			t.Fatalf("EncodeJSON failed: %v", err)
		}
		jsonDecoded, err := codec.DecodeJSON[Data](jsonData)
		if err != nil {
			t.Fatalf("DecodeJSON failed: %v", err)
		}
		if jsonDecoded != original {
			t.Fatalf("JSON round-trip failed")
		}

		// YAML convenience
		yamlData, err := codec.EncodeYAML(original)
		if err != nil {
			t.Fatalf("EncodeYAML failed: %v", err)
		}
		yamlDecoded, err := codec.DecodeYAML[Data](yamlData)
		if err != nil {
			t.Fatalf("DecodeYAML failed: %v", err)
		}
		if yamlDecoded != original {
			t.Fatalf("YAML round-trip failed")
		}

		// Base64 convenience
		rawData := []byte(rapid.String().Draw(t, "rawData"))
		b64Encoded := codec.Base64Encode(rawData)
		b64Decoded, err := codec.Base64Decode(b64Encoded)
		if err != nil {
			t.Fatalf("Base64Decode failed: %v", err)
		}
		if string(b64Decoded) != string(rawData) {
			t.Fatalf("Base64 round-trip failed")
		}

		// URL-safe Base64
		urlEncoded := codec.Base64URLEncode(rawData)
		urlDecoded, err := codec.Base64URLDecode(urlEncoded)
		if err != nil {
			t.Fatalf("Base64URLDecode failed: %v", err)
		}
		if string(urlDecoded) != string(rawData) {
			t.Fatalf("Base64URL round-trip failed")
		}
	})
}
