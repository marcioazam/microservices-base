package loggingclient

import (
	"fmt"
	"time"
)

// Field represents a structured log field.
type Field struct {
	Key   string
	Value string
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an integer field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: fmt.Sprintf("%d", value)}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: fmt.Sprintf("%d", value)}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: fmt.Sprintf("%f", value)}
}

// Bool creates a boolean field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: fmt.Sprintf("%t", value)}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Error creates an error field.
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: ""}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Any creates a field from any value.
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: fmt.Sprintf("%v", value)}
}

// Strings creates a string slice field.
func Strings(key string, values []string) Field {
	return Field{Key: key, Value: fmt.Sprintf("%v", values)}
}

// Time creates a time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value.Format(time.RFC3339)}
}

// fieldsToMap converts fields to a map.
func fieldsToMap(fields []Field) map[string]string {
	m := make(map[string]string, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}
