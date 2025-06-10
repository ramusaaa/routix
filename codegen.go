package routix

import (
	"fmt"
	"reflect"
	"strings"
)

// StructToSchema converts a struct type to a schema based on struct tags
func StructToSchema(t reflect.Type) (*ObjectSchema, error) {
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("type must be a struct")
	}

	fields := make(map[string]Schema)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tag := field.Tag.Get("routix")
		if tag == "" {
			continue
		}

		schema, err := parseFieldSchema(field.Type, tag)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", field.Name, err)
		}

		fields[field.Name] = schema
	}

	return NewObjectSchema(fields), nil
}

func parseFieldSchema(t reflect.Type, tag string) (Schema, error) {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid tag format")
	}

	var schema Schema
	switch t.Kind() {
	case reflect.String:
		schema = NewStringSchema()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		schema = NewNumberSchema()
	case reflect.Bool:
		schema = NewBooleanSchema()
	case reflect.Slice:
		itemSchema, err := parseFieldSchema(t.Elem(), tag)
		if err != nil {
			return nil, err
		}
		schema = NewArraySchema(itemSchema)
	case reflect.Struct:
		objSchema, err := StructToSchema(t)
		if err != nil {
			return nil, err
		}
		schema = objSchema
	default:
		return nil, fmt.Errorf("unsupported type: %v", t.Kind())
	}

	// Parse validation rules
	for _, part := range parts[1:] {
		if part == "required" {
			schema.(interface{ Required() Schema }).Required()
		} else if minStr := strings.TrimPrefix(part, "min="); minStr != part {
			// Handle min validation with minStr
		} else if maxStr := strings.TrimPrefix(part, "max="); maxStr != part {
			// Handle max validation with maxStr
		} else if enumStr := strings.TrimPrefix(part, "enum="); enumStr != part {
			// Handle enum validation with enumStr
		}
	}

	return schema, nil
}
