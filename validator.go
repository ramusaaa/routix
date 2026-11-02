package routix

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Validator struct {
	mu     sync.Mutex
	errors []ValidationError
}

type validatorField struct {
	index int
	name  string
	rules []validatorRule
}

type validatorRule struct {
	text   string
	number float64
	err    error
}

type validatorRegex struct {
	once    sync.Once
	pattern *regexp.Regexp
	err     error
}

var validatorFields sync.Map
var validatorPatterns sync.Map
var validatorEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

func fieldsForValidation(typ reflect.Type) []validatorField {
	if cached, ok := validatorFields.Load(typ); ok {
		return cached.([]validatorField)
	}
	fields := make([]validatorField, 0, typ.NumField())
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("validate"); tag != "" {
			rules := make([]validatorRule, 0)
			for _, text := range strings.Split(tag, ",") {
				rule := validatorRule{text: text}
				name, value, _ := strings.Cut(text, "=")
				if name == "min" || name == "max" || name == "len" {
					rule.number, rule.err = parseNumber(value)
					kind := field.Type.Kind()
					if rule.err == nil && (name == "len" || kind == reflect.String || kind == reflect.Slice || kind == reflect.Map || kind == reflect.Array) && (rule.number < 0 || math.Trunc(rule.number) != rule.number) {
						rule.err = fmt.Errorf("length must be a nonnegative integer")
					}
				}
				rules = append(rules, rule)
			}
			fields = append(fields, validatorField{index: i, name: field.Name, rules: rules})
		}
	}
	cached, _ := validatorFields.LoadOrStore(typ, fields)
	return cached.([]validatorField)
}

func NewValidator() *Validator {
	return &Validator{
		errors: make([]ValidationError, 0),
	}
}

func (v *Validator) Validate(obj interface{}) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.errors = nil
	val := reflect.ValueOf(obj)
	for val.IsValid() && val.Kind() == reflect.Ptr {
		if val.IsNil() {
			v.errors = append(v.errors, *NewValidationError("", "object must be a non-nil struct"))
			return false
		}
		val = val.Elem()
	}
	if !val.IsValid() || val.Kind() != reflect.Struct {
		v.errors = append(v.errors, *NewValidationError("", "object must be a struct"))
		return false
	}
	for _, field := range fieldsForValidation(val.Type()) {
		for _, rule := range field.rules {
			if err := v.validateField(val.Field(field.index), field.name, rule); err != nil {
				v.errors = append(v.errors, *err)
			}
		}
	}
	return len(v.errors) == 0
}

// validateField validates a single field based on the given rule
func (v *Validator) validateField(field reflect.Value, fieldName string, compiled validatorRule) *ValidationError {
	if compiled.err != nil {
		return NewValidationError(fieldName, fmt.Sprintf("invalid validation rule %q: %v", compiled.text, compiled.err))
	}
	rule := compiled.text
	switch {
	case rule == "required":
		if isEmpty(field) {
			return NewValidationError(fieldName, "field is required")
		}
	case strings.HasPrefix(rule, "min="):
		min := compiled.number
		if !validateMin(field, min) {
			return NewValidationError(fieldName, fmt.Sprintf("value must be at least %v", min))
		}
	case strings.HasPrefix(rule, "max="):
		max := compiled.number
		if !validateMax(field, max) {
			return NewValidationError(fieldName, fmt.Sprintf("value must be at most %v", max))
		}
	case strings.HasPrefix(rule, "len="):
		length := compiled.number
		if !validateLength(field, length) {
			return NewValidationError(fieldName, fmt.Sprintf("length must be %v", length))
		}
	case strings.HasPrefix(rule, "email"):
		if !validateEmail(field) {
			return NewValidationError(fieldName, "must be a valid email address")
		}
	case strings.HasPrefix(rule, "regex="):
		pattern := rule[6:]
		matched, err := validateRegex(field, pattern)
		if err != nil {
			return NewValidationError(fieldName, "invalid regex validation rule")
		}
		if !matched {
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
	if !field.IsValid() {
		return true
	}
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
	case reflect.Slice, reflect.Map, reflect.Interface, reflect.Ptr:
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
	return validatorEmail.MatchString(field.String())
}

// validateRegex validates against a regular expression
func validateRegex(field reflect.Value, pattern string) (bool, error) {
	cached, ok := validatorPatterns.Load(pattern)
	if !ok {
		cached, _ = validatorPatterns.LoadOrStore(pattern, &validatorRegex{})
	}
	compiled := cached.(*validatorRegex)
	compiled.once.Do(func() { compiled.pattern, compiled.err = regexp.Compile(pattern) })
	if compiled.err != nil {
		return false, compiled.err
	}
	if field.Kind() != reflect.String {
		return false, nil
	}
	return compiled.pattern.MatchString(field.String()), nil
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
func parseNumber(s string) (float64, error) {
	number, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}
	if math.IsNaN(number) || math.IsInf(number, 0) {
		return 0, fmt.Errorf("value must be finite")
	}
	return number, nil
}

// Errors returns all validation errors
func (v *Validator) Errors() []ValidationError {
	v.mu.Lock()
	defer v.mu.Unlock()
	return append([]ValidationError{}, v.errors...)
}

// ValidateJSON validates JSON data against a struct
func ValidateJSON(data []byte, v interface{}) error {
	if err := json.Unmarshal(data, v); err != nil {
		return BadRequest("Invalid JSON format", err)
	}

	validator := NewValidator()
	if !validator.Validate(v) {
		errors := validator.Errors()
		if len(errors) > 0 {
			return BadRequest("Validation failed", ValidationErrors(convertToValidationErrors(errors)))
		}
	}

	return nil
}

// ValidateStruct validates a struct
func ValidateStruct(v interface{}) error {
	validator := NewValidator()
	if !validator.Validate(v) {
		errors := validator.Errors()
		if len(errors) > 0 {
			return BadRequest("Validation failed", ValidationErrors(convertToValidationErrors(errors)))
		}
	}
	return nil
}

// Helper function to convert validation errors
func convertToValidationErrors(errors []ValidationError) []*ValidationError {
	result := make([]*ValidationError, len(errors))
	for i, err := range errors {
		result[i] = &err
	}
	return result
}
