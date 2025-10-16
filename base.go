package routix

import (
	"fmt"
)

type Schema interface {
	Validate(value any) error
	IsRequired() bool
}

type BaseSchema struct {
	required bool
}

func (b *BaseSchema) IsRequired() bool {
	return b.required
}

func (b *BaseSchema) Required() *BaseSchema {
	b.required = true
	return b
}

func (b *BaseSchema) Validate(value any) error {
	if value == nil && b.required {
		return fmt.Errorf("value is required")
	}
	return nil
}

type StringSchema struct {
	*BaseSchema
	min int
	max int
}

func (s *StringSchema) Min(min int) *StringSchema {
	s.min = min
	return s
}

func (s *StringSchema) Max(max int) *StringSchema {
	s.max = max
	return s
}

func (s *StringSchema) Validate(value any) error {
	if value == nil {
		if s.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}

	if s.IsRequired() && str == "" {
		return fmt.Errorf("value is required")
	}

	if s.min > 0 && len(str) < s.min {
		return fmt.Errorf("string length must be at least %d", s.min)
	}

	if s.max > 0 && len(str) > s.max {
		return fmt.Errorf("string length must be at most %d", s.max)
	}

	return nil
}

type NumberSchema struct {
	*BaseSchema
	min     *float64
	max     *float64
	integer bool
}

func (n *NumberSchema) Min(min float64) *NumberSchema {
	n.min = &min
	return n
}

func (n *NumberSchema) Max(max float64) *NumberSchema {
	n.max = &max
	return n
}

func (n *NumberSchema) Integer() *NumberSchema {
	n.integer = true
	return n
}

func (n *NumberSchema) Validate(value any) error {
	if value == nil {
		if n.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	var num float64
	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int32:
		num = float64(v)
	case int64:
		num = float64(v)
	default:
		return fmt.Errorf("value must be a number")
	}

	if n.integer && num != float64(int(num)) {
		return fmt.Errorf("value must be an integer")
	}

	if n.min != nil && num < *n.min {
		return fmt.Errorf("value must be greater than or equal to %v", *n.min)
	}

	if n.max != nil && num > *n.max {
		return fmt.Errorf("value must be less than or equal to %v", *n.max)
	}

	return nil
}

type BooleanSchema struct {
	*BaseSchema
}

func (b *BooleanSchema) Validate(value any) error {
	if value == nil {
		if b.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	_, ok := value.(bool)
	if !ok {
		return fmt.Errorf("value must be a boolean")
	}

	return nil
}

type ArraySchema struct {
	*BaseSchema
	itemSchema Schema
	minItems   *int
	maxItems   *int
	unique     bool
}

func (a *ArraySchema) MinItems(min int) *ArraySchema {
	a.minItems = &min
	return a
}

func (a *ArraySchema) MaxItems(max int) *ArraySchema {
	a.maxItems = &max
	return a
}

func (a *ArraySchema) Unique() *ArraySchema {
	a.unique = true
	return a
}

func (a *ArraySchema) Validate(value any) error {
	if value == nil {
		if a.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	arr, ok := value.([]any)
	if !ok {
		return fmt.Errorf("value must be an array")
	}

	if a.minItems != nil && len(arr) < *a.minItems {
		return fmt.Errorf("array must have at least %d items", *a.minItems)
	}

	if a.maxItems != nil && len(arr) > *a.maxItems {
		return fmt.Errorf("array must have at most %d items", *a.maxItems)
	}

	if a.unique {
		seen := make(map[any]bool)
		for _, item := range arr {
			if seen[item] {
				return fmt.Errorf("array must contain unique items")
			}
			seen[item] = true
		}
	}

	for i, item := range arr {
		if err := a.itemSchema.Validate(item); err != nil {
			return fmt.Errorf("item at index %d: %w", i, err)
		}
	}

	return nil
}

type ObjectSchema struct {
	*BaseSchema
	fields    map[string]Schema
	strict    bool
	allowNull bool
}

func (o *ObjectSchema) Strict() *ObjectSchema {
	o.strict = true
	return o
}

func (o *ObjectSchema) AllowNull() *ObjectSchema {
	o.allowNull = true
	return o
}

func (o *ObjectSchema) Validate(value any) error {
	if value == nil {
		if o.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	obj, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("value must be an object")
	}

	if o.strict {
		for key := range obj {
			if _, exists := o.fields[key]; !exists {
				return fmt.Errorf("unknown field: %s", key)
			}
		}
	}

	for fieldName, schema := range o.fields {
		fieldValue, exists := obj[fieldName]
		if !exists {
			if schema.IsRequired() {
				return fmt.Errorf("required field missing: %s", fieldName)
			}
			continue
		}

		if err := schema.Validate(fieldValue); err != nil {
			return fmt.Errorf("field %s: %w", fieldName, err)
		}
	}

	return nil
}

type EnumSchema struct {
	*BaseSchema
	values []any
}

func (e *EnumSchema) Validate(value any) error {
	if value == nil {
		if e.IsRequired() {
			return fmt.Errorf("value is required")
		}
		return nil
	}

	for _, allowedValue := range e.values {
		if value == allowedValue {
			return nil
		}
	}

	return fmt.Errorf("value must be one of: %v", e.values)
}

func NewStringSchema() *StringSchema {
	return &StringSchema{BaseSchema: &BaseSchema{}}
}

func NewNumberSchema() *NumberSchema {
	return &NumberSchema{BaseSchema: &BaseSchema{}}
}

func NewBooleanSchema() *BooleanSchema {
	return &BooleanSchema{BaseSchema: &BaseSchema{}}
}

func NewArraySchema(itemSchema Schema) *ArraySchema {
	return &ArraySchema{
		BaseSchema: &BaseSchema{},
		itemSchema: itemSchema,
	}
}

func NewObjectSchema(fields map[string]Schema) *ObjectSchema {
	return &ObjectSchema{
		BaseSchema: &BaseSchema{},
		fields:     fields,
	}
}

func NewEnumSchema(values ...any) *EnumSchema {
	return &EnumSchema{
		BaseSchema: &BaseSchema{},
		values:     values,
	}
}