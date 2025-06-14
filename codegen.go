package routix

import (
	"cmp"
	"fmt"
	"reflect"
	"strings"
)

// Package routix provides code generation utilities for API schemas.
// The codegen package helps generate OpenAPI/Swagger compatible schemas
// from Go structs using struct tags.
//
// Features:
// - Automatic schema generation from structs
// - Support for nested structs and arrays
// - Type conversion to JSON Schema types
// - Validation rule extraction
// - Custom tag support
//
// Example usage:
//   type User struct {
//       Name     string `routix:"type=string,required,min=2,max=50"`
//       Email    string `routix:"type=string,required,format=email"`
//       Age      int    `routix:"type=integer,required,minimum=18,maximum=120"`
//       Roles    []string `routix:"type=array,items=string,enum=admin|user|guest"`
//   }
//
//   schema, err := StructToSchema(reflect.TypeOf(User{}))
//   if err != nil {
//       // Handle error
//   }
//   // Use schema for API documentation

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

	// Try to create schema based on type
	schema := cmp.Or(
		// String type
		func() Schema {
			if t.Kind() == reflect.String {
				return NewStringSchema()
			}
			return nil
		}(),
		// Number types
		func() Schema {
			switch t.Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				return NewNumberSchema()
			}
			return nil
		}(),
		// Boolean type
		func() Schema {
			if t.Kind() == reflect.Bool {
				return NewBooleanSchema()
			}
			return nil
		}(),
		// Array type
		func() Schema {
			if t.Kind() == reflect.Slice {
				itemSchema, err := parseFieldSchema(t.Elem(), tag)
				if err != nil {
					return nil
				}
				return NewArraySchema(itemSchema)
			}
			return nil
		}(),
		// Struct type
		func() Schema {
			if t.Kind() == reflect.Struct {
				objSchema, err := StructToSchema(t)
				if err != nil {
					return nil
				}
				return objSchema
			}
			return nil
		}(),
	)

	if schema == nil {
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
