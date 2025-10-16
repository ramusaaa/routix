package routix

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
)

type CodeGenerator struct {
	schemas map[string]Schema
	routes  []RouteDefinition
}

type RouteDefinition struct {
	Method      string
	Path        string
	Handler     string
	Middleware  []string
	RequestBody Schema
	Response    Schema
}

func NewCodeGenerator() *CodeGenerator {
	return &CodeGenerator{
		schemas: make(map[string]Schema),
		routes:  make([]RouteDefinition, 0),
	}
}

func (cg *CodeGenerator) AddSchema(name string, schema Schema) {
	cg.schemas[name] = schema
}

func (cg *CodeGenerator) AddRoute(route RouteDefinition) {
	cg.routes = append(cg.routes, route)
}

func (cg *CodeGenerator) GenerateValidationCode(structType reflect.Type) string {
	var code strings.Builder
	
	code.WriteString(fmt.Sprintf("func Validate%s(obj %s) error {\n", structType.Name(), structType.Name()))
	
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("validate")
		
		if tag != "" {
			code.WriteString(fmt.Sprintf("    // Validate %s\n", field.Name))
			rules := strings.Split(tag, ",")
			
			for _, rule := range rules {
				switch {
				case rule == "required":
					code.WriteString(fmt.Sprintf("    if obj.%s == \"\" {\n", field.Name))
					code.WriteString(fmt.Sprintf("        return fmt.Errorf(\"%s is required\")\n", field.Name))
					code.WriteString("    }\n")
				case strings.HasPrefix(rule, "min="):
					min := rule[4:]
					code.WriteString(fmt.Sprintf("    if len(obj.%s) < %s {\n", field.Name, min))
					code.WriteString(fmt.Sprintf("        return fmt.Errorf(\"%s must be at least %s characters\")\n", field.Name, min))
					code.WriteString("    }\n")
				case strings.HasPrefix(rule, "max="):
					max := rule[4:]
					code.WriteString(fmt.Sprintf("    if len(obj.%s) > %s {\n", field.Name, max))
					code.WriteString(fmt.Sprintf("        return fmt.Errorf(\"%s must be at most %s characters\")\n", field.Name, max))
					code.WriteString("    }\n")
				}
			}
		}
	}
	
	code.WriteString("    return nil\n")
	code.WriteString("}\n")
	
	return code.String()
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

func (cg *CodeGenerator) ParseStructFromFile(filename, structName string) (reflect.Type, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	for _, decl := range node.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok {
			for _, spec := range genDecl.Specs {
				if typeSpec, ok := spec.(*ast.TypeSpec); ok {
					if typeSpec.Name.Name == structName {
						// This is a simplified version
						// In a real implementation, you'd need to properly parse the AST
						return nil, fmt.Errorf("struct parsing not fully implemented")
					}
				}
			}
		}
	}
	
	return nil, fmt.Errorf("struct %s not found in file %s", structName, filename)
}

func (cg *CodeGenerator) GenerateOpenAPISpec() map[string]interface{} {
	spec := map[string]interface{}{
		"openapi": "3.0.0",
		"info": map[string]interface{}{
			"title":   "Routix API",
			"version": "1.0.0",
		},
		"paths": make(map[string]interface{}),
	}
	
	paths := spec["paths"].(map[string]interface{})
	
	for _, route := range cg.routes {
		if paths[route.Path] == nil {
			paths[route.Path] = make(map[string]interface{})
		}
		
		pathItem := paths[route.Path].(map[string]interface{})
		pathItem[strings.ToLower(route.Method)] = map[string]interface{}{
			"summary": fmt.Sprintf("%s %s", route.Method, route.Path),
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Success",
				},
			},
		}
	}
	
	return spec
}

func (cg *CodeGenerator) GenerateTypeScriptTypes() string {
	var code strings.Builder
	
	code.WriteString("// Generated TypeScript types\n\n")
	
	for name, schema := range cg.schemas {
		code.WriteString(fmt.Sprintf("export interface %s {\n", name))
		
		// This is a simplified version
		// In a real implementation, you'd need to convert Go schemas to TypeScript
		switch s := schema.(type) {
		case *ObjectSchema:
			for fieldName := range s.fields {
				code.WriteString(fmt.Sprintf("  %s: any;\n", fieldName))
			}
		}
		
		code.WriteString("}\n\n")
	}
	
	return code.String()
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