package logging

import "time"

// Field represents a log field.
type Field struct {
	Key   string
	Value any
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field.
func Error(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	return Field{Key: "error", Value: err.Error()}
}

// Duration creates a duration field.
func Duration(key string, value time.Duration) Field {
	return Field{Key: key, Value: value.String()}
}

// Time creates a time field.
func Time(key string, value time.Time) Field {
	return Field{Key: key, Value: value.Format(time.RFC3339Nano)}
}

// Any creates a field with any value.
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// fieldsToMap converts fields to a map.
func fieldsToMap(fields []Field) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	m := make(map[string]any, len(fields))
	for _, f := range fields {
		m[f.Key] = f.Value
	}
	return m
}
