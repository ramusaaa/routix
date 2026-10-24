package routix

import (
	"fmt"
	"path"
	"reflect"
	"sort"
)

type Module struct {
	Path        string
	Routes      []ModuleRoute
	Middleware  []Middleware
	Services    map[reflect.Type]interface{}
	SubModules  []*Module
	Controllers []Controller
	Imports     []*Module
}

type ModuleInterface interface {
	Configure() *ModuleConfig
}

type ModuleConfig struct {
	Path        string
	Controllers []Controller
	Services    []interface{}
	Imports     []*Module
	Middleware  []Middleware
}

type ModuleRoute struct {
	Method  string
	Path    string
	Handler Handler
}

type Controller interface {
	Register(r *Router)
}

func NewModule(path string) *Module {
	return &Module{
		Path:       path,
		Services:   make(map[reflect.Type]interface{}),
		SubModules: make([]*Module, 0),
		Imports:    make([]*Module, 0),
	}
}

func (m *Module) Use(middleware ...Middleware) {
	m.Middleware = append(m.Middleware, middleware...)
}

func (m *Module) AddRoute(method, path string, handler Handler) {
	m.Routes = append(m.Routes, ModuleRoute{
		Method:  method,
		Path:    path,
		Handler: handler,
	})
}

func (m *Module) AddController(controller Controller) {
	m.Controllers = append(m.Controllers, controller)
}

func (m *Module) AddService(service interface{}) {
	t := reflect.TypeOf(service)
	m.Services[t] = service
}

func (m *Module) AddSubModule(subModule *Module) {
	m.SubModules = append(m.SubModules, subModule)
}

func (m *Module) Import(module *Module) {
	m.Imports = append(m.Imports, module)
}

func (m *Module) GetService(t reflect.Type) (interface{}, bool) {
	service, ok := m.Services[t]
	if ok {
		return service, true
	}

	for _, imp := range m.Imports {
		service, ok = imp.GetService(t)
		if ok {
			return service, true
		}
	}

	return nil, false
}

func (m *Module) Register(r *Router) {
	m.register(r, "/", nil, make(map[*Module]bool))
}

func (m *Module) register(r *Router, prefix string, inherited []Middleware, active map[*Module]bool) {
	if m == nil || active[m] {
		panic("routix: cyclic or nil module")
	}
	active[m] = true
	defer delete(active, m)
	prefix = path.Join(prefix, m.Path)
	middleware := append(append([]Middleware(nil), inherited...), m.Middleware...)
	for _, route := range m.Routes {
		r.handle(route.Method, path.Join(prefix, route.Path), route.Handler, middleware)
	}
	for _, controller := range m.Controllers {
		local := New()
		controller.Register(local)
		for _, route := range local.Routes() {
			matched, _ := local.findRoute(local.trees[route.Method], route.Path, make(map[string]string))
			scoped := append(append(append([]Middleware(nil), middleware...), local.middleware...), matched.middleware...)
			r.handle(route.Method, path.Join(prefix, route.Path), matched.handler, scoped)
		}
	}
	for _, child := range m.SubModules {
		child.register(r, prefix, middleware, active)
	}
}

type ModuleBuilder struct {
	module *Module
}

func NewModuleBuilder(path string) *ModuleBuilder {
	return &ModuleBuilder{
		module: NewModule(path),
	}
}

func (b *ModuleBuilder) WithMiddleware(middleware ...Middleware) *ModuleBuilder {
	b.module.Use(middleware...)
	return b
}

func (b *ModuleBuilder) WithController(controller Controller) *ModuleBuilder {
	b.module.AddController(controller)
	return b
}

func (b *ModuleBuilder) WithService(service interface{}) *ModuleBuilder {
	b.module.AddService(service)
	return b
}

func (b *ModuleBuilder) WithSubModule(subModule *Module) *ModuleBuilder {
	b.module.AddSubModule(subModule)
	return b
}

func (b *ModuleBuilder) WithImport(module *Module) *ModuleBuilder {
	b.module.Import(module)
	return b
}

func (b *ModuleBuilder) Build() *Module {
	return b.module
}

type ServiceContainer struct {
	services   map[reflect.Type]interface{}
	singletons map[reflect.Type]interface{}
}

func NewServiceContainer() *ServiceContainer {
	return &ServiceContainer{
		services:   make(map[reflect.Type]interface{}),
		singletons: make(map[reflect.Type]interface{}),
	}
}

func (sc *ServiceContainer) Register(service interface{}) {
	t := reflect.TypeOf(service)
	sc.services[t] = service
}

func (sc *ServiceContainer) RegisterSingleton(service interface{}) {
	t := reflect.TypeOf(service)
	sc.singletons[t] = service
}

func (sc *ServiceContainer) Get(t reflect.Type) (interface{}, bool) {
	if singleton, ok := sc.singletons[t]; ok {
		return singleton, true
	}

	if service, ok := sc.services[t]; ok {
		return service, true
	}

	return nil, false
}

func (sc *ServiceContainer) Resolve(ptr interface{}) error {
	v := reflect.ValueOf(ptr)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("ptr must be a pointer")
	}

	elem := v.Elem()
	t := elem.Type()

	service, ok := sc.Get(t)
	if !ok {
		return fmt.Errorf("service of type %s not found", t.String())
	}

	elem.Set(reflect.ValueOf(service))
	return nil
}

type Dependency struct {
	Type     reflect.Type
	Instance interface{}
	Factory  func() interface{}
	Scope    DependencyScope
}

type DependencyScope int

const (
	Transient DependencyScope = iota
	Singleton
	Scoped
)

type DIContainer struct {
	dependencies map[reflect.Type]*Dependency
	instances    map[reflect.Type]interface{}
}

func NewDIContainer() *DIContainer {
	return &DIContainer{
		dependencies: make(map[reflect.Type]*Dependency),
		instances:    make(map[reflect.Type]interface{}),
	}
}

func (di *DIContainer) RegisterTransient(factory func() interface{}) {
	if factory == nil {
		panic("routix: transient factory must not be nil")
	}
	t := reflect.TypeOf(factory())
	if t == nil {
		panic("routix: transient factory must return a non-nil value")
	}
	di.dependencies[t] = &Dependency{
		Type:    t,
		Factory: factory,
		Scope:   Transient,
	}
}

func (di *DIContainer) RegisterSingleton(instance interface{}) {
	t := reflect.TypeOf(instance)
	di.dependencies[t] = &Dependency{
		Type:     t,
		Instance: instance,
		Scope:    Singleton,
	}
	di.instances[t] = instance
}

func (di *DIContainer) Resolve(t reflect.Type) (interface{}, error) {
	dep, ok := di.dependencies[t]
	if !ok {
		return nil, fmt.Errorf("dependency of type %s not registered", t.String())
	}

	switch dep.Scope {
	case Singleton:
		if instance, ok := di.instances[t]; ok {
			return instance, nil
		}
		if dep.Instance != nil {
			return dep.Instance, nil
		}
		if dep.Factory != nil {
			instance := dep.Factory()
			di.instances[t] = instance
			return instance, nil
		}
	case Transient:
		if dep.Factory != nil {
			return dep.Factory(), nil
		}
	}

	return nil, fmt.Errorf("cannot resolve dependency of type %s", t.String())
}

type ModuleRegistry struct {
	modules   map[string]*Module
	container *DIContainer
}

func NewModuleRegistry() *ModuleRegistry {
	return &ModuleRegistry{
		modules:   make(map[string]*Module),
		container: NewDIContainer(),
	}
}

func (mr *ModuleRegistry) RegisterModule(name string, module *Module) {
	mr.modules[name] = module

	for t, service := range module.Services {
		mr.container.RegisterSingleton(service)
		_ = t
	}
}

func (mr *ModuleRegistry) GetModule(name string) (*Module, bool) {
	module, ok := mr.modules[name]
	return module, ok
}

func (mr *ModuleRegistry) GetService(t reflect.Type) (interface{}, error) {
	return mr.container.Resolve(t)
}

func (mr *ModuleRegistry) BuildRouter() *Router {
	router := New()

	names := make([]string, 0, len(mr.modules))
	for name := range mr.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		mr.modules[name].Register(router)
	}

	return router
}

type AppModule struct {
	*Module
	registry *ModuleRegistry
}

func NewAppModule() *AppModule {
	return &AppModule{
		Module:   NewModule("/"),
		registry: NewModuleRegistry(),
	}
}

func (app *AppModule) ImportModule(name string, module *Module) *AppModule {
	app.registry.RegisterModule(name, module)
	app.Import(module)
	return app
}

func (app *AppModule) GetService(t reflect.Type) (interface{}, error) {
	return app.registry.GetService(t)
}

func (app *AppModule) CreateRouter() *Router {
	router := New()
	app.Module.Register(router)
	names := make([]string, 0, len(app.registry.modules))
	for name := range app.registry.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		app.registry.modules[name].register(router, app.Path, app.Middleware, make(map[*Module]bool))
	}
	return router
}
