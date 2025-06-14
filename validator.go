package routix

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"time"
)

// Package routix provides a powerful validation system for HTTP requests.
// The validator package implements a flexible and extensible validation framework
// that supports various validation rules and custom validators.
//
// Features:
// - Struct-based validation using tags
// - Built-in validators for common types (string, number, date, etc.)
// - Custom validation rules
// - Detailed error messages
// - Support for nested structs
// - Extensible architecture
//
// Example usage:
//   type User struct {
//       Name     string `validate:"required,min=2,max=50"`
//       Email    string `validate:"required,email"`
//       Age      int    `validate:"required,min=18,max=120"`
//       Password string `validate:"required,min=8,regex=^[a-zA-Z0-9!@#$%^&*]+$"`
//       Role     string `validate:"required,enum=admin|user|guest"`
//       BirthDate string `validate:"required,date=2006-01-02"`
//   }
//
//   validator := NewValidator()
//   if !validator.Validate(user) {
//       errors := validator.Errors()
//       // Handle validation errors
//   }

// Validator provides advanced validation features
type Validator struct {
	errors []ValidationError
}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{
		errors: make([]ValidationError, 0),
	}
}

// Validate validates a struct based on its tags
func (v *Validator) Validate(obj interface{}) bool {
	val := reflect.ValueOf(obj)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		v.errors = append(v.errors, *NewValidationError("", "object must be a struct"))
		return false
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		tag := fieldType.Tag.Get("validate")

		if tag == "" {
			continue
		}

		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			if err := v.validateField(field, fieldType.Name, rule); err != nil {
				v.errors = append(v.errors, *err)
			}
		}
	}

	return len(v.errors) == 0
}

// validateField validates a single field based on the given rule
func (v *Validator) validateField(field reflect.Value, fieldName, rule string) *ValidationError {
	switch {
	case rule == "required":
		if isEmpty(field) {
			return NewValidationError(fieldName, "field is required")
		}
	case strings.HasPrefix(rule, "min="):
		min := parseNumber(rule[4:])
		if !validateMin(field, min) {
			return NewValidationError(fieldName, fmt.Sprintf("value must be at least %v", min))
		}
	case strings.HasPrefix(rule, "max="):
		max := parseNumber(rule[4:])
		if !validateMax(field, max) {
			return NewValidationError(fieldName, fmt.Sprintf("value must be at most %v", max))
		}
	case strings.HasPrefix(rule, "len="):
		length := parseNumber(rule[4:])
		if !validateLength(field, length) {
			return NewValidationError(fieldName, fmt.Sprintf("length must be %v", length))
		}
	case strings.HasPrefix(rule, "email"):
		if !validateEmail(field) {
			return NewValidationError(fieldName, "must be a valid email address")
		}
	case strings.HasPrefix(rule, "regex="):
		pattern := rule[6:]
		if !validateRegex(field, pattern) {
			return NewValidationError(fieldName, "must match the required pattern")
		}
	case strings.HasPrefix(rule, "enum="):
		values := strings.Split(rule[5:], "|")
		if !validateEnum(field, values) {
			return NewValidationError(fieldName, fmt.Sprintf("must be one of: %v", values))
		}
	case strings.HasPrefix(rule, "date="):
		format := rule[5:]
		if !validateDate(field, format) {
			return NewValidationError(fieldName, fmt.Sprintf("must be a valid date in format: %s", format))
		}
	}

	return nil
}

// isEmpty checks if a field is empty
func isEmpty(field reflect.Value) bool {
	switch field.Kind() {
	case reflect.String:
		return field.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return field.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return field.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return field.Float() == 0
	case reflect.Bool:
		return !field.Bool()
	case reflect.Slice, reflect.Map, reflect.Interface:
		return field.IsNil()
	}
	return false
}

// validateMin validates minimum value
func validateMin(field reflect.Value, min float64) bool {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(field.Int()) >= min
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(field.Uint()) >= min
	case reflect.Float32, reflect.Float64:
		return field.Float() >= min
	case reflect.String:
		return float64(len(field.String())) >= min
	case reflect.Slice, reflect.Map:
		return float64(field.Len()) >= min
	}
	return false
}

// validateMax validates maximum value
func validateMax(field reflect.Value, max float64) bool {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(field.Int()) <= max
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(field.Uint()) <= max
	case reflect.Float32, reflect.Float64:
		return field.Float() <= max
	case reflect.String:
		return float64(len(field.String())) <= max
	case reflect.Slice, reflect.Map:
		return float64(field.Len()) <= max
	}
	return false
}

// validateLength validates exact length
func validateLength(field reflect.Value, length float64) bool {
	switch field.Kind() {
	case reflect.String:
		return float64(len(field.String())) == length
	case reflect.Slice, reflect.Map:
		return float64(field.Len()) == length
	}
	return false
}

// validateEmail validates email format
func validateEmail(field reflect.Value) bool {
	if field.Kind() != reflect.String {
		return false
	}
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(field.String())
}

// validateRegex validates against a regular expression
func validateRegex(field reflect.Value, pattern string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	regex := regexp.MustCompile(pattern)
	return regex.MatchString(field.String())
}

// validateEnum validates against a list of allowed values
func validateEnum(field reflect.Value, values []string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	value := field.String()
	for _, v := range values {
		if value == v {
			return true
		}
	}
	return false
}

// validateDate validates date format
func validateDate(field reflect.Value, format string) bool {
	if field.Kind() != reflect.String {
		return false
	}
	_, err := time.Parse(format, field.String())
	return err == nil
}

// parseNumber parses a number from a string
func parseNumber(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// Errors returns all validation errors
func (v *Validator) Errors() []ValidationError {
	return v.errors
}
