// Package routix provides a module system for organizing routes, middleware, and services.
// It allows developers to create modular applications with dependency injection.
package routix

import (
	"path/filepath"
	"reflect"
)

// Module represents a module in the application.
// It contains routes, middleware, and services.
type Module struct {
	Path        string
	Routes      []Route
	Middleware  []Middleware
	Services    map[reflect.Type]interface{}
	SubModules  []*Module
	Controllers []Controller
	Imports     []*Module
}

// ModuleInterface defines the interface that all modules must implement
type ModuleInterface interface {
	// Configure configures the module
	Configure() *ModuleConfig
}

// ModuleConfig represents the configuration for a module
type ModuleConfig struct {
	Path        string
	Controllers []Controller
	Services    []interface{}
	Imports     []*Module
	Middleware  []Middleware
}

// Route represents a route in the module
type Route struct {
	Method  string
	Path    string
	Handler Handler
}

// Controller represents a controller in the module
type Controller interface {
	Register(r *Router)
}

// NewModule creates a new module with the given path
func NewModule(path string) *Module {
	return &Module{
		Path:       path,
		Services:   make(map[reflect.Type]interface{}),
		SubModules: make([]*Module, 0),
		Imports:    make([]*Module, 0),
	}
}

// Use adds middleware to the module
func (m *Module) Use(middleware ...Middleware) {
	m.Middleware = append(m.Middleware, middleware...)
}

// AddRoute adds a route to the module
func (m *Module) AddRoute(method, path string, handler Handler) {
	m.Routes = append(m.Routes, Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
}

// AddController adds a controller to the module
func (m *Module) AddController(controller Controller) {
	m.Controllers = append(m.Controllers, controller)
}

// AddService adds a service to the module
func (m *Module) AddService(service interface{}) {
	t := reflect.TypeOf(service)
	m.Services[t] = service
}

// AddSubModule adds a submodule to the module
func (m *Module) AddSubModule(subModule *Module) {
	m.SubModules = append(m.SubModules, subModule)
}

// Import adds an imported module
func (m *Module) Import(module *Module) {
	m.Imports = append(m.Imports, module)
}

// GetService gets a service from the module
func (m *Module) GetService(t reflect.Type) (interface{}, bool) {
	// First check in this module
	service, ok := m.Services[t]
	if ok {
		return service, true
	}

	// Then check in imported modules
	for _, imp := range m.Imports {
		service, ok = imp.GetService(t)
		if ok {
			return service, true
		}
	}

	return nil, false
}

// Register registers the module with the router
func (m *Module) Register(r *Router) {
	// Register middleware
	for _, middleware := range m.Middleware {
		r.Use(middleware)
	}

	// Register routes
	for _, route := range m.Routes {
		path := filepath.Join(m.Path, route.Path)
		r.Handle(route.Method, path, route.Handler)
	}

	// Register controllers
	for _, controller := range m.Controllers {
		controller.Register(r)
	}

	// Register submodules
	for _, subModule := range m.SubModules {
		subModule.Register(r)
	}
}

// ModuleBuilder helps build modules with a fluent interface
type ModuleBuilder struct {
	module *Module
}

// NewModuleBuilder creates a new module builder
func NewModuleBuilder(path string) *ModuleBuilder {
	return &ModuleBuilder{
		module: NewModule(path),
	}
}

// WithMiddleware adds middleware to the module
func (b *ModuleBuilder) WithMiddleware(middleware ...Middleware) *ModuleBuilder {
	b.module.Use(middleware...)
	return b
}

// WithController adds a controller to the module
func (b *ModuleBuilder) WithController(controller Controller) *ModuleBuilder {
	b.module.AddController(controller)
	return b
}

// WithService adds a service to the module
func (b *ModuleBuilder) WithService(service interface{}) *ModuleBuilder {
	b.module.AddService(service)
	return b
}

// WithSubModule adds a submodule to the module
func (b *ModuleBuilder) WithSubModule(subModule *Module) *ModuleBuilder {
	b.module.AddSubModule(subModule)
	return b
}

// WithImport adds an imported module
func (b *ModuleBuilder) WithImport(module *Module) *ModuleBuilder {
	b.module.Import(module)
	return b
}

// Build builds the module
func (b *ModuleBuilder) Build() *Module {
	return b.module
}

// Example usage:
// type StoreModule struct {
//     *Module
// }
//
// func NewStoreModule() *StoreModule {
//     return &StoreModule{
//         Module: NewModule("/store"),
//     }
// }
//
// func (m *StoreModule) Configure() *ModuleConfig {
//     return &ModuleConfig{
//         Path: "/store",
//         Controllers: []Controller{
//             &StoreController{},
//         },
//         Services: []interface{}{
//             &StoreService{},
//         },
//         Middleware: []Middleware{
//             Logger(),
//         },
//     }
// }
//
// func main() {
//     router := routix.New()
//
//     // Create modules
//     storeModule := NewStoreModule()
//     config := storeModule.Configure()
//
//     // Build module
//     module := routix.NewModuleBuilder(config.Path).
//         WithMiddleware(config.Middleware...).
//         WithController(config.Controllers[0]).
//         WithService(config.Services[0]).
//         Build()
//
//     // Register module
//     module.Register(router)
//
//     // Start server
//     http.ListenAndServe(":8080", router)
// }
