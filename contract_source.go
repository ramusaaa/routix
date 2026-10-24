package routix

import (
	"encoding/json"
	"fmt"
	"go/types"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"golang.org/x/tools/go/packages"
)

func sourceStruct(filename, name string) (types.Type, error) {
	absolute, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	loaded, err := packages.Load(&packages.Config{Dir: filepath.Dir(absolute), Mode: packages.NeedName | packages.NeedFiles | packages.NeedTypes | packages.NeedImports | packages.NeedDeps}, "file="+absolute)
	if err != nil {
		return nil, err
	}
	if len(loaded) != 1 {
		return nil, fmt.Errorf("expected one package for %s", filename)
	}
	if len(loaded[0].Errors) > 0 {
		return nil, fmt.Errorf("load struct: %s", loaded[0].Errors[0])
	}
	object := loaded[0].Types.Scope().Lookup(name)
	if object == nil {
		return nil, fmt.Errorf("struct %s not found in %s", name, filename)
	}
	if _, ok := object.(*types.TypeName); !ok {
		return nil, fmt.Errorf("%s is not a type", name)
	}
	typ := object.Type()
	if _, ok := typ.Underlying().(*types.Struct); !ok {
		return nil, fmt.Errorf("%s is not a struct", name)
	}
	return typ, nil
}

func sourceReflect(typ types.Type, active map[types.Type]bool) (reflect.Type, error) {
	typ = types.Unalias(typ)
	if active[typ] {
		return nil, fmt.Errorf("recursive type %s cannot be represented by reflect.StructOf; use ParseSchemaFromFile", typ)
	}
	active = cloneMap(active)
	active[typ] = true
	switch value := typ.(type) {
	case *types.Named:
		if value.Obj().Pkg() != nil {
			switch value.Obj().Pkg().Path() + "." + value.Obj().Name() {
			case "time.Time":
				return reflect.TypeOf(time.Time{}), nil
			case "encoding/json.Number":
				return reflect.TypeOf(json.Number("")), nil
			case "encoding/json.RawMessage":
				return reflect.TypeOf(json.RawMessage{}), nil
			}
		}
		return sourceReflect(value.Underlying(), active)
	case *types.Basic:
		table := map[types.BasicKind]reflect.Type{types.Bool: reflect.TypeOf(false), types.Int: reflect.TypeOf(int(0)), types.Int8: reflect.TypeOf(int8(0)), types.Int16: reflect.TypeOf(int16(0)), types.Int32: reflect.TypeOf(int32(0)), types.Int64: reflect.TypeOf(int64(0)), types.Uint: reflect.TypeOf(uint(0)), types.Uint8: reflect.TypeOf(uint8(0)), types.Uint16: reflect.TypeOf(uint16(0)), types.Uint32: reflect.TypeOf(uint32(0)), types.Uint64: reflect.TypeOf(uint64(0)), types.Float32: reflect.TypeOf(float32(0)), types.Float64: reflect.TypeOf(float64(0)), types.String: reflect.TypeOf("")}
		if result := table[value.Kind()]; result != nil {
			return result, nil
		}
	case *types.Pointer:
		element, err := sourceReflect(value.Elem(), active)
		if err != nil {
			return nil, err
		}
		return reflect.PointerTo(element), nil
	case *types.Slice:
		element, err := sourceReflect(value.Elem(), active)
		if err != nil {
			return nil, err
		}
		return reflect.SliceOf(element), nil
	case *types.Array:
		element, err := sourceReflect(value.Elem(), active)
		if err != nil {
			return nil, err
		}
		return reflect.ArrayOf(int(value.Len()), element), nil
	case *types.Map:
		key, err := sourceReflect(value.Key(), active)
		if err != nil {
			return nil, err
		}
		element, err := sourceReflect(value.Elem(), active)
		if err != nil {
			return nil, err
		}
		return reflect.MapOf(key, element), nil
	case *types.Interface:
		if value.NumMethods() == 0 {
			return reflect.TypeOf((*any)(nil)).Elem(), nil
		}
	case *types.Struct:
		fields := []reflect.StructField{}
		for i := 0; i < value.NumFields(); i++ {
			field := value.Field(i)
			if !field.Exported() {
				continue
			}
			element, err := sourceReflect(field.Type(), active)
			if err != nil {
				return nil, fmt.Errorf("field %s: %w", field.Name(), err)
			}
			if field.Embedded() && element.Name() == "" {
				return nil, fmt.Errorf("embedded type %s cannot be represented by reflect.StructOf; use ParseSchemaFromFile", field.Type())
			}
			anonymous := field.Embedded() && element.Name() != ""
			fields = append(fields, reflect.StructField{Name: field.Name(), Type: element, Tag: reflect.StructTag(value.Tag(i)), Anonymous: anonymous})
		}
		return reflect.StructOf(fields), nil
	}
	return nil, fmt.Errorf("unsupported source type: %s", typ)
}

func (cg *CodeGenerator) ParseStructFromFile(filename, name string) (result reflect.Type, err error) {
	defer func() {
		if value := recover(); value != nil {
			result = nil
			err = fmt.Errorf("represent struct %s: %v", name, value)
		}
	}()
	typ, err := sourceStruct(filename, name)
	if err != nil {
		return nil, err
	}
	result, err = sourceReflect(typ, map[types.Type]bool{})
	if err == nil {
		cg.typeNames[result] = name
	}
	return result, err
}

func (cg *CodeGenerator) ParseSchemaFromFile(filename, name string) (Schema, error) {
	typ, err := sourceStruct(filename, name)
	if err != nil {
		return nil, err
	}
	definitions := map[string]any{}
	document, err := sourceContract(typ, definitions)
	if err != nil {
		return nil, err
	}
	return &contractSchema{document: document, definitions: definitions, required: true}, nil
}

func sourceContract(typ types.Type, definitions map[string]any) (map[string]any, error) {
	typ = types.Unalias(typ)
	switch value := typ.(type) {
	case *types.Named:
		if value.Obj().Pkg() != nil {
			switch value.Obj().Pkg().Path() + "." + value.Obj().Name() {
			case "time.Time":
				return map[string]any{"type": "string", "format": "date-time"}, nil
			case "encoding/json.Number":
				return map[string]any{"type": "number"}, nil
			case "encoding/json.RawMessage":
				return map[string]any{}, nil
			}
		}
		for i := 0; i < value.NumMethods(); i++ {
			method := value.Method(i).Name()
			if method == "MarshalJSON" || method == "MarshalText" {
				return nil, fmt.Errorf("custom JSON marshaler %s requires an explicit schema", value)
			}
		}
		name := value.Obj().Name()
		if value.Obj().Pkg() != nil {
			name = value.Obj().Pkg().Path() + "_" + name
		}
		name = contractName(name)
		if _, exists := definitions[name]; !exists {
			definitions[name] = nil
			document, err := sourceContract(value.Underlying(), definitions)
			if err != nil {
				return nil, err
			}
			definitions[name] = document
		}
		return map[string]any{"$ref": "#/components/schemas/" + name}, nil
	case *types.Pointer:
		document, err := sourceContract(value.Elem(), definitions)
		if err != nil {
			return nil, err
		}
		return nullableContract(document, definitions), nil
	case *types.Struct:
		properties := map[string]any{}
		required := []string{}
		for _, entry := range sourceContractFields(value) {
			field, tag := entry.field, entry.tag
			parts := strings.Split(tag.Get("json"), ",")
			document, err := sourceContract(field.Type(), definitions)
			if err != nil {
				return nil, err
			}
			name := parts[0]
			if name == "" {
				name = field.Name()
			}
			if containsString(parts[1:], "string") {
				base := types.Unalias(field.Type())
				if pointer, ok := base.(*types.Pointer); ok {
					base = types.Unalias(pointer.Elem())
				}
				if basic, ok := base.Underlying().(*types.Basic); ok && basic.Info()&(types.IsBoolean|types.IsNumeric|types.IsString) != 0 {
					nullable := document["nullable"] == true
					document = map[string]any{"type": "string"}
					if nullable {
						document["nullable"] = true
					}
				}
			}
			if err := applyContractRules(document, tag.Get("validate")); err != nil {
				return nil, err
			}
			properties[name] = document
			if !entry.optional && (!containsString(parts[1:], "omitempty") && !containsString(parts[1:], "omitzero") || containsString(strings.Split(tag.Get("validate"), ","), "required")) {
				required = append(required, name)
			}
		}
		sort.Strings(required)
		result := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			result["required"] = required
		}
		return result, nil
	case *types.Slice:
		if types.Identical(value.Elem(), types.Typ[types.Byte]) {
			return map[string]any{"type": "string", "format": "byte", "nullable": true}, nil
		}
		item, err := sourceContract(value.Elem(), definitions)
		return map[string]any{"type": "array", "items": item, "nullable": true}, err
	case *types.Array:
		item, err := sourceContract(value.Elem(), definitions)
		return map[string]any{"type": "array", "items": item, "minItems": value.Len(), "maxItems": value.Len()}, err
	case *types.Map:
		key, ok := value.Key().Underlying().(*types.Basic)
		if !ok || key.Info()&(types.IsString|types.IsInteger) == 0 {
			return nil, fmt.Errorf("unsupported map key %s", value.Key())
		}
		item, err := sourceContract(value.Elem(), definitions)
		return map[string]any{"type": "object", "additionalProperties": item, "nullable": true}, err
	default:
		reflected, err := sourceReflect(typ, map[types.Type]bool{})
		if err != nil {
			return nil, err
		}
		return reflectContract(reflected, definitions)
	}
}

type sourceContractField struct {
	field    *types.Var
	tag      reflect.StructTag
	optional bool
}

func sourceContractFields(typ *types.Struct) []sourceContractField {
	type candidate struct {
		entry  sourceContractField
		depth  int
		tagged bool
	}
	candidates := map[string][]candidate{}
	var visit func(*types.Struct, int, bool, map[types.Type]bool)
	visit = func(current *types.Struct, depth int, optional bool, seen map[types.Type]bool) {
		if seen[current] {
			return
		}
		seen = cloneMap(seen)
		seen[current] = true
		for i := 0; i < current.NumFields(); i++ {
			field := current.Field(i)
			tag := reflect.StructTag(current.Tag(i))
			name := strings.Split(tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			base := types.Unalias(field.Type())
			pointer, nullable := base.(*types.Pointer)
			if nullable {
				base = types.Unalias(pointer.Elem())
			}
			if nested, ok := base.Underlying().(*types.Struct); ok && field.Embedded() && name == "" {
				visit(nested, depth+1, optional || nullable, seen)
				continue
			}
			if !field.Exported() {
				continue
			}
			key := name
			if key == "" {
				key = field.Name()
			}
			candidates[key] = append(candidates[key], candidate{sourceContractField{field, tag, optional}, depth, name != ""})
		}
	}
	visit(typ, 0, false, map[types.Type]bool{})
	keys := make([]string, 0, len(candidates))
	for name := range candidates {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	result := []sourceContractField{}
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
		result = append(result, values[0].entry)
	}
	return result
}
