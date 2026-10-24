package routix

import (
	"context"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

type contractSchema struct {
	document    map[string]any
	definitions map[string]any
	required    bool
}

func (s *contractSchema) IsRequired() bool { return s.required }
func (s *contractSchema) Validate(value any) error {
	if value == nil && !s.required {
		return nil
	}
	definitions := cloneMap(s.definitions)
	definitions["Root"] = s.document
	document := map[string]any{"openapi": "3.0.3", "info": map[string]any{"title": "Schema", "version": "1.0.0"}, "paths": map[string]any{}, "components": map[string]any{"schemas": definitions}}
	data, err := json.Marshal(document)
	if err != nil {
		return err
	}
	parsed, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return err
	}
	if err := parsed.Validate(context.Background()); err != nil {
		return err
	}
	data, err = json.Marshal(value)
	if err != nil {
		return err
	}
	var normalized any
	if err := json.Unmarshal(data, &normalized); err != nil {
		return err
	}
	return parsed.Components.Schemas["Root"].Value.VisitJSON(normalized)
}

func SchemaFromType(value any) (Schema, error) {
	typ, ok := value.(reflect.Type)
	if !ok {
		typ = reflect.TypeOf(value)
	}
	if typ == nil {
		return nil, fmt.Errorf("schema type must not be nil")
	}
	definitions := map[string]any{}
	document, err := reflectContract(typ, definitions)
	if err != nil {
		return nil, err
	}
	return &contractSchema{document: document, definitions: definitions, required: true}, nil
}

func contractName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' {
			result.WriteRune(r)
		} else {
			fmt.Fprintf(&result, "_%x_", r)
		}
	}
	return result.String()
}

func nullableContract(document map[string]any, definitions map[string]any) map[string]any {
	if ref, ok := document["$ref"].(string); ok {
		kind := any("object")
		if definition, ok := definitions[strings.TrimPrefix(ref, "#/components/schemas/")].(map[string]any); ok {
			result := cloneMap(definition)
			result["nullable"] = true
			return result
		}
		result := map[string]any{"nullable": true, "allOf": []any{document}}
		if kind != nil {
			result["type"] = kind
		}
		return result
	}
	result := cloneMap(document)
	result["nullable"] = true
	return result
}

func reflectContract(typ reflect.Type, definitions map[string]any) (map[string]any, error) {
	if typ == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}, nil
	}
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return map[string]any{}, nil
	}
	if typ == reflect.TypeOf(json.Number("")) {
		return map[string]any{"type": "number"}, nil
	}
	if typ.Kind() == reflect.Pointer {
		document, err := reflectContract(typ.Elem(), definitions)
		if err != nil {
			return nil, err
		}
		return nullableContract(document, definitions), nil
	}
	marshaler := reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshaler := reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	if typ.Implements(marshaler) || typ.Kind() != reflect.Pointer && reflect.PointerTo(typ).Implements(marshaler) || typ.Implements(textMarshaler) || typ.Kind() != reflect.Pointer && reflect.PointerTo(typ).Implements(textMarshaler) {
		return nil, fmt.Errorf("custom JSON marshaler %s requires an explicit schema", typ)
	}
	switch typ.Kind() {
	case reflect.String:
		return map[string]any{"type": "string"}, nil
	case reflect.Bool:
		return map[string]any{"type": "boolean"}, nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result := map[string]any{"type": "integer"}
		if typ.Kind() >= reflect.Uint && typ.Kind() <= reflect.Uint64 {
			result["minimum"] = 0
		}
		return result, nil
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number"}, nil
	case reflect.Interface:
		if typ.NumMethod() > 0 {
			return nil, fmt.Errorf("cannot infer JSON representation of %s", typ)
		}
		return map[string]any{}, nil
	case reflect.Slice, reflect.Array:
		if typ.Kind() == reflect.Slice && typ.Elem().Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "byte", "nullable": true}, nil
		}
		item, err := reflectContract(typ.Elem(), definitions)
		if err != nil {
			return nil, err
		}
		result := map[string]any{"type": "array", "items": item}
		if typ.Kind() == reflect.Slice {
			result["nullable"] = true
		} else {
			result["minItems"] = typ.Len()
			result["maxItems"] = typ.Len()
		}
		return result, nil
	case reflect.Map:
		if typ.Key().Kind() != reflect.String && !(typ.Key().Kind() >= reflect.Int && typ.Key().Kind() <= reflect.Uint64) {
			return nil, fmt.Errorf("unsupported JSON map key: %s", typ.Key())
		}
		item, err := reflectContract(typ.Elem(), definitions)
		return map[string]any{"type": "object", "additionalProperties": item, "nullable": true}, err
	case reflect.Struct:
		name := ""
		if typ.Name() != "" {
			name = contractName(typ.PkgPath() + "_" + typ.Name())
			if _, ok := definitions[name]; ok {
				return map[string]any{"$ref": "#/components/schemas/" + name}, nil
			}
			definitions[name] = nil
		}
		properties := map[string]any{}
		required := []string{}
		for _, field := range jsonContractFields(typ) {
			document, err := reflectContract(field.Type, definitions)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name, err)
			}
			parts := strings.Split(field.Tag.Get("json"), ",")
			key := parts[0]
			if key == "" {
				key = field.Name
			}
			if containsString(parts[1:], "string") {
				kind := field.Type
				for kind.Kind() == reflect.Pointer {
					kind = kind.Elem()
				}
				if kind.Kind() == reflect.String || kind.Kind() == reflect.Bool || kind.Kind() >= reflect.Int && kind.Kind() <= reflect.Float64 {
					nullable := document["nullable"] == true
					document = map[string]any{"type": "string"}
					if nullable {
						document["nullable"] = true
					}
				}
			}
			if err := applyContractRules(document, field.Tag.Get("validate")); err != nil {
				return nil, fmt.Errorf("field %s: %w", key, err)
			}
			properties[key] = document
			if !field.optional && (!containsString(parts[1:], "omitempty") && !containsString(parts[1:], "omitzero") || containsString(strings.Split(field.Tag.Get("validate"), ","), "required")) {
				required = append(required, key)
			}
		}
		sort.Strings(required)
		result := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			result["required"] = required
		}
		if name != "" {
			definitions[name] = result
			return map[string]any{"$ref": "#/components/schemas/" + name}, nil
		}
		return result, nil
	}
	return nil, fmt.Errorf("unsupported JSON type: %s", typ)
}

func containsString(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

type contractField struct {
	reflect.StructField
	optional bool
}

func jsonContractFields(typ reflect.Type) []contractField {
	type candidate struct {
		field  contractField
		depth  int
		tagged bool
	}
	candidates := map[string][]candidate{}
	var visit func(reflect.Type, int, bool, map[reflect.Type]bool)
	visit = func(current reflect.Type, depth int, optional bool, seen map[reflect.Type]bool) {
		if seen[current] {
			return
		}
		seen = cloneMap(seen)
		seen[current] = true
		for i := 0; i < current.NumField(); i++ {
			field := current.Field(i)
			tag := strings.Split(field.Tag.Get("json"), ",")[0]
			if tag == "-" {
				continue
			}
			base := field.Type
			for base.Kind() == reflect.Pointer {
				base = base.Elem()
			}
			if field.Anonymous && tag == "" && base.Kind() == reflect.Struct {
				visit(base, depth+1, optional || field.Type.Kind() == reflect.Pointer, seen)
				continue
			}
			if field.PkgPath != "" {
				continue
			}
			key := tag
			if key == "" {
				key = field.Name
			}
			candidates[key] = append(candidates[key], candidate{contractField{field, optional}, depth, tag != ""})
		}
	}
	visit(typ, 0, false, map[reflect.Type]bool{})
	keys := make([]string, 0, len(candidates))
	for key := range candidates {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	fields := []contractField{}
	for _, key := range keys {
		values := candidates[key]
		sort.SliceStable(values, func(i, j int) bool {
			if values[i].depth != values[j].depth {
				return values[i].depth < values[j].depth
			}
			return values[i].tagged && !values[j].tagged
		})
		if len(values) > 1 && values[0].depth == values[1].depth && values[0].tagged == values[1].tagged {
			continue
		}
		fields = append(fields, values[0].field)
	}
	return fields
}

func applyContractRules(document map[string]any, tag string) error {
	for _, rule := range strings.Split(tag, ",") {
		name, value, hasValue := strings.Cut(rule, "=")
		switch name {
		case "", "required", "omitempty":
		case "min", "max", "len":
			number, err := strconv.ParseFloat(value, 64)
			if !hasValue || err != nil {
				return fmt.Errorf("invalid validation rule %q", rule)
			}
			switch document["type"] {
			case "string":
				if name == "min" || name == "len" {
					document["minLength"] = number
				}
				if name == "max" || name == "len" {
					document["maxLength"] = number
				}
			case "array":
				if name == "min" || name == "len" {
					document["minItems"] = number
				}
				if name == "max" || name == "len" {
					document["maxItems"] = number
				}
			case "integer", "number":
				if name == "min" {
					document["minimum"] = number
				}
				if name == "max" {
					document["maximum"] = number
				}
			}
		case "email":
			document["format"] = "email"
		case "url":
			document["format"] = "uri"
		case "regex":
			document["pattern"] = value
		case "enum":
			values := []any{}
			for _, entry := range strings.Split(value, "|") {
				values = append(values, entry)
			}
			document["enum"] = values
		case "alpha":
			document["pattern"] = "^[a-zA-Z]+$"
		case "alphanum":
			document["pattern"] = "^[a-zA-Z0-9]+$"
		case "numeric":
			document["pattern"] = "^[+-]?[0-9]+(?:\\.[0-9]+)?$"
		case "date":
			if value == time.RFC3339 {
				document["format"] = "date-time"
			} else if value == "2006-01-02" {
				document["format"] = "date"
			}
		default:
			return fmt.Errorf("unsupported contract validation rule %q", rule)
		}
	}
	return nil
}

func schemaContract(schema Schema, definitions map[string]any) (map[string]any, error) {
	if schema == nil || reflect.ValueOf(schema).Kind() == reflect.Pointer && reflect.ValueOf(schema).IsNil() {
		return nil, fmt.Errorf("nil schema")
	}
	switch s := schema.(type) {
	case *contractSchema:
		for key, value := range s.definitions {
			if old, ok := definitions[key]; ok && !reflect.DeepEqual(old, value) {
				return nil, fmt.Errorf("conflicting schema %s", key)
			}
			definitions[key] = value
		}
		return s.document, nil
	case *StringSchema:
		result := map[string]any{"type": "string"}
		if s.min > 0 {
			result["minLength"] = s.min
		}
		if s.max > 0 {
			result["maxLength"] = s.max
		}
		return result, nil
	case *NumberSchema:
		kind := "number"
		if s.integer {
			kind = "integer"
		}
		result := map[string]any{"type": kind}
		if s.min != nil {
			result["minimum"] = *s.min
		}
		if s.max != nil {
			result["maximum"] = *s.max
		}
		return result, nil
	case *BooleanSchema:
		return map[string]any{"type": "boolean"}, nil
	case *ArraySchema:
		item, err := schemaContract(s.itemSchema, definitions)
		if err != nil {
			return nil, err
		}
		result := map[string]any{"type": "array", "items": item}
		if s.minItems != nil {
			result["minItems"] = *s.minItems
		}
		if s.maxItems != nil {
			result["maxItems"] = *s.maxItems
		}
		if s.unique {
			result["uniqueItems"] = true
		}
		return result, nil
	case *ObjectSchema:
		properties := map[string]any{}
		required := []string{}
		for key, field := range s.fields {
			value, err := schemaContract(field, definitions)
			if err != nil {
				return nil, err
			}
			properties[key] = value
			if field.IsRequired() {
				required = append(required, key)
			}
		}
		sort.Strings(required)
		result := map[string]any{"type": "object", "properties": properties}
		if s.strict {
			result["additionalProperties"] = false
		}
		if s.allowNull {
			result["nullable"] = true
		}
		if len(required) > 0 {
			result["required"] = required
		}
		return result, nil
	case *EnumSchema:
		return map[string]any{"enum": s.values}, nil
	case *BaseSchema:
		return map[string]any{}, nil
	}
	return nil, fmt.Errorf("schema %T does not support contract generation", schema)
}
