package validation

import "fmt"

// FieldValidator validates a specific field.
type FieldValidator struct {
	field      string
	validators []func() *ValidationError
}

// NewFieldValidator creates a new field validator.
func NewFieldValidator(name string) *FieldValidator {
	return &FieldValidator{field: name}
}

// String adds a string validator.
func (f *FieldValidator) String(value string, validators ...Validator[string]) *FieldValidator {
	for _, v := range validators {
		validator := v
		val := value
		f.validators = append(f.validators, func() *ValidationError {
			if err := validator(val); err != nil {
				err.Field = f.field
				return err
			}
			return nil
		})
	}
	return f
}

// Int adds an int validator.
func (f *FieldValidator) Int(value int, validators ...Validator[int]) *FieldValidator {
	for _, v := range validators {
		validator := v
		val := value
		f.validators = append(f.validators, func() *ValidationError {
			if err := validator(val); err != nil {
				err.Field = f.field
				return err
			}
			return nil
		})
	}
	return f
}

// Slice adds a slice validator.
func (f *FieldValidator) Slice(value []string, validators ...Validator[[]string]) *FieldValidator {
	for _, v := range validators {
		validator := v
		val := value
		f.validators = append(f.validators, func() *ValidationError {
			if err := validator(val); err != nil {
				err.Field = f.field
				return err
			}
			return nil
		})
	}
	return f
}

// Validate runs all validators.
func (f *FieldValidator) Validate() *Result {
	result := NewResult()
	for _, v := range f.validators {
		if err := v(); err != nil {
			result.AddError(*err)
		}
	}
	return result
}

// StructValidator validates a struct with multiple fields.
type StructValidator struct {
	fields []*FieldValidator
	nested map[string]*StructValidator
	path   string
}

// NewStructValidator creates a new struct validator.
func NewStructValidator() *StructValidator {
	return &StructValidator{
		nested: make(map[string]*StructValidator),
	}
}

// AddField adds a field validator.
func (s *StructValidator) AddField(field *FieldValidator) *StructValidator {
	s.fields = append(s.fields, field)
	return s
}

// AddNested adds a nested struct validator.
func (s *StructValidator) AddNested(name string, validator *StructValidator) *StructValidator {
	s.nested[name] = validator
	return s
}

// Validate runs all field and nested validators.
func (s *StructValidator) Validate() *Result {
	return s.validateWithPrefix("")
}

func (s *StructValidator) validateWithPrefix(prefix string) *Result {
	result := NewResult()

	// Validate fields
	for _, field := range s.fields {
		fieldResult := field.Validate()
		for _, err := range fieldResult.Errors() {
			if prefix != "" {
				err.Path = fmt.Sprintf("%s.%s", prefix, err.Field)
			} else {
				err.Path = err.Field
			}
			result.AddError(err)
		}
	}

	// Validate nested
	for name, nested := range s.nested {
		nestedPrefix := name
		if prefix != "" {
			nestedPrefix = fmt.Sprintf("%s.%s", prefix, name)
		}
		nestedResult := nested.validateWithPrefix(nestedPrefix)
		for _, err := range nestedResult.Errors() {
			result.AddError(err)
		}
	}

	return result
}

// ValidateAllFields validates multiple field validators at once.
func ValidateAllFields(fields ...*FieldValidator) *Result {
	result := NewResult()
	for _, field := range fields {
		result.Merge(field.Validate())
	}
	return result
}
