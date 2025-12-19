package codec

import (
	"reflect"
	"testing"
)

type testStruct struct {
	Name  string `json:"name" yaml:"name"`
	Value int    `json:"value" yaml:"value"`
}

func TestMarshalJSON(t *testing.T) {
	input := testStruct{Name: "test", Value: 42}
	data, err := MarshalJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"name":"test","value":42}`
	if string(data) != expected {
		t.Errorf("expected %s, got %s", expected, string(data))
	}
}

func TestUnmarshalJSON(t *testing.T) {
	data := []byte(`{"name":"test","value":42}`)
	result, err := UnmarshalJSON[testStruct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := testStruct{Name: "test", Value: 42}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMarshalJSONIndent(t *testing.T) {
	input := testStruct{Name: "test", Value: 42}
	data, err := MarshalJSONIndent(input, "", "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMarshalYAML(t *testing.T) {
	input := testStruct{Name: "test", Value: 42}
	data, err := MarshalYAML(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestUnmarshalYAML(t *testing.T) {
	data := []byte("name: test\nvalue: 42\n")
	result, err := UnmarshalYAML[testStruct](data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := testStruct{Name: "test", Value: 42}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestRoundTripJSON(t *testing.T) {
	input := testStruct{Name: "roundtrip", Value: 123}
	result, err := RoundTripJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected %v, got %v", input, result)
	}
}

func TestRoundTripYAML(t *testing.T) {
	input := testStruct{Name: "roundtrip", Value: 456}
	result, err := RoundTripYAML(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected %v, got %v", input, result)
	}
}

func TestClone(t *testing.T) {
	input := testStruct{Name: "clone", Value: 789}
	result, err := Clone(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != input {
		t.Errorf("expected %v, got %v", input, result)
	}
}

func TestMustMarshalJSON(t *testing.T) {
	input := testStruct{Name: "must", Value: 1}
	data := MustMarshalJSON(input)
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMustUnmarshalJSON(t *testing.T) {
	data := []byte(`{"name":"must","value":2}`)
	result := MustUnmarshalJSON[testStruct](data)
	expected := testStruct{Name: "must", Value: 2}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestMustMarshalYAML(t *testing.T) {
	input := testStruct{Name: "must", Value: 3}
	data := MustMarshalYAML(input)
	if len(data) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestMustUnmarshalYAML(t *testing.T) {
	data := []byte("name: must\nvalue: 4\n")
	result := MustUnmarshalYAML[testStruct](data)
	expected := testStruct{Name: "must", Value: 4}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestToJSONString(t *testing.T) {
	input := testStruct{Name: "string", Value: 5}
	result, err := ToJSONString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"name":"string","value":5}`
	if result != expected {
		t.Errorf("expected %s, got %s", expected, result)
	}
}

func TestFromJSONString(t *testing.T) {
	input := `{"name":"string","value":6}`
	result, err := FromJSONString[testStruct](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := testStruct{Name: "string", Value: 6}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestToYAMLString(t *testing.T) {
	input := testStruct{Name: "yaml", Value: 7}
	result, err := ToYAMLString(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty output")
	}
}

func TestFromYAMLString(t *testing.T) {
	input := "name: yaml\nvalue: 8\n"
	result, err := FromYAMLString[testStruct](input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := testStruct{Name: "yaml", Value: 8}
	if result != expected {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestRoundTripJSON_Complex(t *testing.T) {
	type nested struct {
		Items []string          `json:"items"`
		Map   map[string]int    `json:"map"`
	}
	input := nested{
		Items: []string{"a", "b", "c"},
		Map:   map[string]int{"x": 1, "y": 2},
	}
	result, err := RoundTripJSON(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, input) {
		t.Errorf("expected %v, got %v", input, result)
	}
}

func TestRoundTripYAML_Complex(t *testing.T) {
	type nested struct {
		Items []string          `yaml:"items"`
		Map   map[string]int    `yaml:"map"`
	}
	input := nested{
		Items: []string{"a", "b", "c"},
		Map:   map[string]int{"x": 1, "y": 2},
	}
	result, err := RoundTripYAML(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(result, input) {
		t.Errorf("expected %v, got %v", input, result)
	}
}

func TestUnmarshalJSON_Error(t *testing.T) {
	data := []byte(`invalid json`)
	_, err := UnmarshalJSON[testStruct](data)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUnmarshalYAML_Error(t *testing.T) {
	data := []byte(`invalid: yaml: content:`)
	_, err := UnmarshalYAML[testStruct](data)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}
