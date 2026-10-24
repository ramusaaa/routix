package routix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/getkin/kin-openapi/openapi3"
	"reflect"
	"strings"
)

type CodeGenerator struct {
	schemas   map[string]Schema
	routes    []RouteDefinition
	router    *Router
	typeNames map[reflect.Type]string
}

type RouteDefinition struct {
	Method       string
	Path         string
	Handler      string
	Middleware   []string
	RequestBody  Schema
	Response     Schema
	RequestType  any
	ResponseType any
	StatusCode   int
	Summary      string
	OperationID  string
}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		schemas:   make(map[string]Schema),
		typeNames: make(map[reflect.Type]string),
		routes:    make([]RouteDefinition, 0),
	}
}

func (cg *CodeGenerator) AddSchema(name string, schema Schema) {
	cg.schemas[name] = schema
}

func (cg *CodeGenerator) AddRoute(route RouteDefinition) {
	cg.routes = append(cg.routes, route)
}

func (cg *CodeGenerator) GenerateValidationCode(structType reflect.Type) string {
	name := structType.Name()
	if name == "" {
		name = cg.typeNames[structType]
	}
	if name == "" {
		panic("routix: validation generation requires a named struct")
	}
	return fmt.Sprintf("func Validate%s(obj %s) error {\n return routix.ValidateStruct(obj)\n}\n", name, name)
}

func (cg *CodeGenerator) GenerateRouterCode() string {
	var code strings.Builder

	code.WriteString("func SetupRoutes(r *routix.Router) {\n")

	for _, route := range cg.routes {
		code.WriteString(fmt.Sprintf("    r.%s(\"%s\", %s)\n",
			strings.ToUpper(route.Method),
			route.Path,
			route.Handler))
	}

	code.WriteString("}\n")

	return code.String()
}

func (cg *CodeGenerator) GenerateMiddlewareCode(middlewares []string) string {
	var code strings.Builder

	code.WriteString("func SetupMiddleware(r *routix.Router) {\n")

	for _, middleware := range middlewares {
		code.WriteString(fmt.Sprintf("    r.Use(routix.%s())\n", middleware))
	}

	code.WriteString("}\n")

	return code.String()
}

func (cg *CodeGenerator) BindRouter(router *Router) *CodeGenerator { cg.router = router; return cg }

func (cg *CodeGenerator) AddType(name string, value any) error {
	schema, err := SchemaFromType(value)
	if err != nil {
		return err
	}
	cg.AddSchema(name, schema)
	return nil
}

func (cg *CodeGenerator) contractRoutes() ([]RouteDefinition, error) {
	if cg.router == nil {
		return append([]RouteDefinition(nil), cg.routes...), nil
	}
	configured := map[RouteInfo]RouteDefinition{}
	for _, route := range cg.routes {
		key := RouteInfo{Method: strings.ToUpper(route.Method), Path: route.Path}
		if _, exists := configured[key]; exists {
			return nil, fmt.Errorf("duplicate route contract %s %s", key.Method, key.Path)
		}
		configured[key] = route
	}
	routes := []RouteDefinition{}
	for _, info := range cg.router.Routes() {
		route, exists := configured[info]
		if !exists {
			route = RouteDefinition{Method: info.Method, Path: info.Path}
		}
		routes = append(routes, route)
		delete(configured, info)
	}
	if len(configured) > 0 {
		return nil, fmt.Errorf("route contract does not match a registered route")
	}
	return routes, nil
}

func (cg *CodeGenerator) OpenAPI() (map[string]any, error) {
	definitions := map[string]any{}
	names := make([]string, 0, len(cg.schemas))
	for name := range cg.schemas {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		document, err := schemaContract(cg.schemas[name], definitions)
		if err != nil {
			return nil, err
		}
		if old, exists := definitions[name]; exists && !reflect.DeepEqual(old, document) {
			return nil, fmt.Errorf("conflicting schema name %s", name)
		}
		definitions[name] = document
	}
	paths := map[string]any{}
	routes, err := cg.contractRoutes()
	if err != nil {
		return nil, err
	}
	for _, route := range routes {
		method := strings.ToLower(route.Method)
		switch method {
		case "get", "put", "post", "delete", "options", "head", "patch", "trace":
		default:
			return nil, fmt.Errorf("unsupported OpenAPI method %s", route.Method)
		}
		if route.Path == "" || route.Path[0] != '/' {
			return nil, fmt.Errorf("invalid route path %q", route.Path)
		}
		parts := strings.Split(route.Path, "/")
		parameters := []any{}
		for i, part := range parts {
			parameter := ""
			if strings.HasPrefix(part, ":") {
				parameter = strings.TrimPrefix(part, ":")
			}
			if part == "*" {
				parameter = "wildcard"
			}
			if parameter != "" {
				parts[i] = "{" + parameter + "}"
				parameters = append(parameters, map[string]any{"name": parameter, "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
			}
		}
		path := strings.Join(parts, "/")
		item, ok := paths[path].(map[string]any)
		if !ok {
			item = map[string]any{}
			paths[path] = item
		}
		if _, exists := item[method]; exists {
			return nil, fmt.Errorf("duplicate OpenAPI operation %s %s", method, path)
		}
		status := route.StatusCode
		if status == 0 {
			status = http.StatusOK
		}
		if status < 100 || status > 599 {
			return nil, fmt.Errorf("invalid response status %d", status)
		}
		response := map[string]any{"description": http.StatusText(status)}
		if response["description"] == "" {
			response["description"] = "Response"
		}
		responseSchema := route.Response
		if route.ResponseType != nil {
			if responseSchema != nil {
				return nil, fmt.Errorf("choose response schema or response type")
			}
			responseSchema, err = SchemaFromType(route.ResponseType)
			if err != nil {
				return nil, err
			}
		}
		if responseSchema != nil {
			if status == http.StatusNoContent || status == http.StatusNotModified {
				return nil, fmt.Errorf("response status %d cannot contain a body", status)
			}
			document, err := schemaContract(responseSchema, definitions)
			if err != nil {
				return nil, err
			}
			response["content"] = map[string]any{"application/json": map[string]any{"schema": document}}
		}
		operation := map[string]any{"responses": map[string]any{strconv.Itoa(status): response}}
		if route.Summary != "" {
			operation["summary"] = route.Summary
		}
		if route.OperationID != "" {
			operation["operationId"] = route.OperationID
		}
		if len(parameters) > 0 {
			operation["parameters"] = parameters
		}
		requestSchema := route.RequestBody
		if route.RequestType != nil {
			if requestSchema != nil {
				return nil, fmt.Errorf("choose request schema or request type")
			}
			requestSchema, err = SchemaFromType(route.RequestType)
			if err != nil {
				return nil, err
			}
		}
		if requestSchema != nil {
			document, err := schemaContract(requestSchema, definitions)
			if err != nil {
				return nil, err
			}
			operation["requestBody"] = map[string]any{"required": requestSchema.IsRequired(), "content": map[string]any{"application/json": map[string]any{"schema": document}}}
		}
		item[method] = operation
	}
	document := map[string]any{"openapi": "3.0.3", "info": map[string]any{"title": "Routix API", "version": "1.0.0"}, "paths": paths, "components": map[string]any{"schemas": definitions}}
	data, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	parsed, err := openapi3.NewLoader().LoadFromData(data)
	if err != nil {
		return nil, err
	}
	if err := parsed.Validate(context.Background()); err != nil {
		return nil, err
	}
	return document, nil
}

func (cg *CodeGenerator) GenerateOpenAPISpec() map[string]interface{} {
	document, err := cg.OpenAPI()
	if err != nil {
		panic(err)
	}
	return document
}

func (cg *CodeGenerator) OpenAPIHandler() Handler {
	return func(c *Context) error {
		document, err := cg.OpenAPI()
		if err != nil {
			return err
		}
		return c.JSON(http.StatusOK, document)
	}
}

func GenerateHandlerTemplate(method, path, handlerName string) string {
	template := `func %s(c *routix.Context) error {
	// TODO: Implement %s %s
	return c.Success(map[string]string{
		"message": "Handler not implemented",
	})
}`

	return fmt.Sprintf(template, handlerName, method, path)
}

func GenerateModelTemplate(modelName string, fields map[string]string) string {
	var code strings.Builder

	code.WriteString(fmt.Sprintf("type %s struct {\n", modelName))

	for fieldName, fieldType := range fields {
		code.WriteString(fmt.Sprintf("    %s %s `json:\"%s\"`\n",
			fieldName,
			fieldType,
			strings.ToLower(fieldName)))
	}

	code.WriteString("}\n")

	return code.String()
}

func GenerateTestTemplate(handlerName, method, path string) string {
	template := `func Test%s(t *testing.T) {
	r := routix.New()
	r.%s("%s", %s)
	
	req := httptest.NewRequest("%s", "%s", nil)
	w := httptest.NewRecorder()
	
	r.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %%d", w.Code)
	}
}`

	return fmt.Sprintf(template,
		handlerName,
		strings.ToUpper(method),
		path,
		handlerName,
		strings.ToUpper(method),
		path)
}
