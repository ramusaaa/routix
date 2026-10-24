package routix

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

func contractKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func typescriptSchema(schema map[string]any) string {
	result := "unknown"
	if ref, ok := schema["$ref"].(string); ok {
		result = strings.TrimPrefix(ref, "#/components/schemas/")
	} else if values, ok := schema["enum"].([]any); ok {
		parts := []string{}
		for _, value := range values {
			encoded, err := json.Marshal(value)
			if err == nil {
				parts = append(parts, string(encoded))
			}
		}
		if len(parts) > 0 {
			result = strings.Join(parts, " | ")
		}
	} else if items, ok := schema["allOf"].([]any); ok {
		parts := []string{}
		for _, item := range items {
			if value, ok := item.(map[string]any); ok {
				parts = append(parts, typescriptSchema(value))
			}
		}
		if len(parts) > 0 {
			result = strings.Join(parts, " & ")
		}
	} else {
		switch schema["type"] {
		case "string":
			result = "string"
		case "integer", "number":
			result = "number"
		case "boolean":
			result = "boolean"
		case "array":
			item, _ := schema["items"].(map[string]any)
			result = "Array<" + typescriptSchema(item) + ">"
		case "object":
			properties, _ := schema["properties"].(map[string]any)
			required, _ := schema["required"].([]string)
			var body strings.Builder
			body.WriteString("{ ")
			for _, name := range contractKeys(properties) {
				field, _ := properties[name].(map[string]any)
				suffix := "?"
				if containsString(required, name) {
					suffix = ""
				}
				fmt.Fprintf(&body, "%s%s: %s; ", strconv.Quote(name), suffix, typescriptSchema(field))
			}
			switch additional := schema["additionalProperties"].(type) {
			case map[string]any:
				fmt.Fprintf(&body, "[key: string]: %s; ", typescriptSchema(additional))
			case bool:
				if additional {
					body.WriteString("[key: string]: unknown; ")
				}
			}
			body.WriteString("}")
			result = body.String()
		}
	}
	if schema["nullable"] == true && result != "unknown" {
		result = "(" + result + ") | null"
	}
	return result
}

func (cg *CodeGenerator) TypeScriptTypes() (string, error) {
	document, err := cg.OpenAPI()
	if err != nil {
		return "", err
	}
	definitions := document["components"].(map[string]any)["schemas"].(map[string]any)
	identifier := regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
	reserved := strings.Fields("any boolean break case catch class const continue debugger declare default delete do else enum export extends false finally for function if import in instanceof interface let module namespace never new null number object package private protected public return static string super switch symbol this throw true try type typeof undefined unknown var void while with yield APIPaths")
	var code strings.Builder
	for _, name := range contractKeys(definitions) {
		if !identifier.MatchString(name) || containsString(reserved, name) {
			return "", fmt.Errorf("schema name %q is not a valid TypeScript type name", name)
		}
		schema, _ := definitions[name].(map[string]any)
		fmt.Fprintf(&code, "export type %s = %s;\n\n", name, typescriptSchema(schema))
	}
	code.WriteString("export interface APIPaths {\n")
	paths := document["paths"].(map[string]any)
	for _, path := range contractKeys(paths) {
		fmt.Fprintf(&code, "  %s: {\n", strconv.Quote(path))
		methods := paths[path].(map[string]any)
		for _, method := range contractKeys(methods) {
			operation := methods[method].(map[string]any)
			request := "undefined"
			if body, ok := operation["requestBody"].(map[string]any); ok {
				content := body["content"].(map[string]any)["application/json"].(map[string]any)
				request = typescriptSchema(content["schema"].(map[string]any))
				if body["required"] != true {
					request = "(" + request + ") | undefined"
				}
			}
			response := "unknown"
			responses := operation["responses"].(map[string]any)
			for _, status := range contractKeys(responses) {
				entry := responses[status].(map[string]any)
				if content, ok := entry["content"].(map[string]any); ok {
					media := content["application/json"].(map[string]any)
					response = typescriptSchema(media["schema"].(map[string]any))
				} else if status == "204" || status == "304" {
					response = "undefined"
				}
			}
			params := []string{}
			if parameters, ok := operation["parameters"].([]any); ok {
				for _, parameter := range parameters {
					name := parameter.(map[string]any)["name"].(string)
					params = append(params, strconv.Quote(name)+": string;")
				}
			}
			fmt.Fprintf(&code, "    %s: { params: { %s }; request: %s; response: %s };\n", strconv.Quote(method), strings.Join(params, " "), request, response)
		}
		code.WriteString("  };\n")
	}
	code.WriteString("}\n")
	return code.String(), nil
}

func (cg *CodeGenerator) GenerateTypeScriptTypes() string {
	code, err := cg.TypeScriptTypes()
	if err != nil {
		panic(err)
	}
	return code
}
