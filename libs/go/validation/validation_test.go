package validation

import (
	"testing"
	"time"
)

func TestPositive(t *testing.T) {
	validator := Positive[int]()

	tests := []struct {
		value   int
		wantErr bool
	}{
		{1, false},
		{100, false},
		{0, true},
		{-1, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Positive(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestNonNegative(t *testing.T) {
	validator := NonNegative[int]()

	tests := []struct {
		value   int
		wantErr bool
	}{
		{0, false},
		{1, false},
		{100, false},
		{-1, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("NonNegative(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestInRange(t *testing.T) {
	validator := InRange[int](1, 10)

	tests := []struct {
		value   int
		wantErr bool
	}{
		{1, false},
		{5, false},
		{10, false},
		{0, true},
		{11, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("InRange(1,10)(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestNonEmpty(t *testing.T) {
	validator := NonEmpty()

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"hello", false},
		{"a", false},
		{"", true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("NonEmpty(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestMinLength(t *testing.T) {
	validator := MinLength(3)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"abc", false},
		{"abcd", false},
		{"ab", true},
		{"", true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("MinLength(3)(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestMaxLength(t *testing.T) {
	validator := MaxLength(5)

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"", false},
		{"abc", false},
		{"abcde", false},
		{"abcdef", true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("MaxLength(5)(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestPositiveDuration(t *testing.T) {
	validator := PositiveDuration()

	tests := []struct {
		value   time.Duration
		wantErr bool
	}{
		{time.Second, false},
		{time.Millisecond, false},
		{0, true},
		{-time.Second, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("PositiveDuration(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestDurationInRange(t *testing.T) {
	validator := DurationInRange(time.Second, time.Minute)

	tests := []struct {
		value   time.Duration
		wantErr bool
	}{
		{time.Second, false},
		{30 * time.Second, false},
		{time.Minute, false},
		{500 * time.Millisecond, true},
		{2 * time.Minute, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("DurationInRange(1s,1m)(%v) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestOneOf(t *testing.T) {
	validator := OneOf("a", "b", "c")

	tests := []struct {
		value   string
		wantErr bool
	}{
		{"a", false},
		{"b", false},
		{"c", false},
		{"d", true},
		{"", true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("OneOf(a,b,c)(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestNotNil(t *testing.T) {
	validator := NotNil[int]()

	value := 42
	if err := validator("field", &value); err != nil {
		t.Errorf("NotNil(&42) unexpected error: %v", err)
	}

	if err := validator("field", nil); err == nil {
		t.Error("NotNil(nil) expected error")
	}
}

func TestCompose(t *testing.T) {
	validator := Compose(
		Positive[int](),
		InRange[int](1, 100),
	)

	tests := []struct {
		value   int
		wantErr bool
	}{
		{1, false},
		{50, false},
		{100, false},
		{0, true},
		{101, true},
	}

	for _, tt := range tests {
		err := validator("field", tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Compose(Positive, InRange)(%d) error = %v, wantErr %v", tt.value, err, tt.wantErr)
		}
	}
}

func TestBuilder(t *testing.T) {
	// No errors
	b := NewBuilder()
	b.Validate(nil)
	b.Validate(nil)
	if err := b.Build(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// Single error
	b = NewBuilder()
	b.Validate(&ValidationError{Field: "f1", Message: "error1"})
	err := b.Build()
	if err == nil {
		t.Error("expected error")
	}

	// Multiple errors
	b = NewBuilder()
	b.Validate(&ValidationError{Field: "f1", Message: "error1"})
	b.Validate(&ValidationError{Field: "f2", Message: "error2"})
	err = b.Build()
	if err == nil {
		t.Error("expected error")
	}
	if errs, ok := err.(ValidationErrors); ok {
		if len(errs) != 2 {
			t.Errorf("expected 2 errors, got %d", len(errs))
		}
	}
}

func TestValidationErrorsHasErrors(t *testing.T) {
	var errs ValidationErrors
	if errs.HasErrors() {
		t.Error("expected no errors")
	}

	errs = append(errs, &ValidationError{Field: "f", Message: "m"})
	if !errs.HasErrors() {
		t.Error("expected errors")
	}
}
