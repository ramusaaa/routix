package routix

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"
)

// responseWriter wraps http.ResponseWriter to capture the status code.
type responseWriter struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		rw.ResponseWriter.WriteHeader(code)
		return
	}
	if !rw.written {
		rw.status = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.status = http.StatusOK
		rw.written = true
	}
	return rw.ResponseWriter.Write(b)
}

func (rw *responseWriter) Unwrap() http.ResponseWriter { return rw.ResponseWriter }

func (rw *responseWriter) Flush() {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(rw.ResponseWriter).Flush()
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(rw.ResponseWriter).Hijack()
}

func (rw *responseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Context holds request/response state for a single HTTP request.
type Context struct {
	Request   *http.Request
	Writer    *responseWriter
	Response  http.ResponseWriter // kept for backward compat; points to Writer
	Params    map[string]string
	Query     map[string]string
	Body      map[string]any
	values    map[string]any
	rawBody   []byte
	bodyRead  bool
	bodyErr   error
	bodyLimit int64
	handler   Handler
	clientIP  string
	routePath string
	secure    bool
}

// Set stores a value in the context, scoped to this request.
func (c *Context) Set(key string, value any) {
	if c.values == nil {
		c.values = make(map[string]any)
	}
	c.values[key] = value
}

// Get retrieves a value previously stored with Set.
func (c *Context) Get(key string) (any, bool) {
	if c.values == nil {
		return nil, false
	}
	v, ok := c.values[key]
	return v, ok
}

// MustGet retrieves a value or panics if the key doesn't exist.
func (c *Context) MustGet(key string) any {
	v, ok := c.Get(key)
	if !ok {
		panic("routix: key not found in context: " + key)
	}
	return v
}

// Status returns the HTTP status code written for this request.
func (c *Context) Status() int {
	return c.Writer.Status()
}

func getContextFromPool(req *http.Request, w *responseWriter, params, query map[string]string, body map[string]any) *Context {
	ctx := getContext()
	ctx.Request = req
	ctx.Writer = w
	ctx.Response = w
	ctx.Params = params
	ctx.Query = query
	ctx.Body = body
	ctx.values = nil
	ctx.rawBody = nil
	ctx.bodyRead = false
	ctx.bodyErr = nil
	ctx.bodyLimit = 1 << 20
	ctx.handler = nil
	ctx.clientIP = ""
	ctx.routePath = ""
	ctx.secure = false
	return ctx
}

func putContextToPool(ctx *Context) {
	ctx.Request = nil
	ctx.Writer = nil
	ctx.Response = nil
	ctx.Params = nil
	ctx.Query = nil
	ctx.Body = nil
	ctx.values = nil
	ctx.rawBody = nil
	ctx.bodyRead = false
	ctx.bodyErr = nil
	ctx.bodyLimit = 1 << 20
	ctx.handler = nil
	ctx.clientIP = ""
	ctx.routePath = ""
	ctx.secure = false
	putContext(ctx)
}

// Handler is a function that handles an HTTP request.
type Handler func(*Context) error

// RouteInfo holds information about a registered route.
type RouteInfo struct {
	Method string
	Path   string
}

// Router is the core HTTP router.
type Router struct {
	trees          map[string]*node
	routes         []RouteInfo
	params         *sync.Pool
	notFound       Handler
	notMethod      Handler
	middleware     []Middleware
	cache          sync.Map
	devMode        bool
	MaxBodyBytes   int64
	trustedProxies []*net.IPNet
	mu             sync.RWMutex
}

type node struct {
	path       string
	handler    Handler
	children   map[string]*node
	params     []string
	wildcard   bool
	paramNames []string
	routeIndex int
	middleware []Middleware
}

// Middleware wraps a Handler with additional logic.
type Middleware func(Handler) Handler

// New creates a new Router with sensible defaults.
func New() *Router {
	return &Router{
		trees: make(map[string]*node),
		params: &sync.Pool{
			New: func() any {
				return make(map[string]string)
			},
		},
		notFound: func(c *Context) error {
			c.Response.Header().Set("Content-Type", "application/json")
			c.Response.WriteHeader(http.StatusNotFound)
			json.NewEncoder(c.Response).Encode(map[string]any{
				"status":  "error",
				"message": "route not found",
			})
			return nil
		},
		notMethod: func(c *Context) error {
			c.Response.Header().Set("Content-Type", "application/json")
			c.Response.WriteHeader(http.StatusMethodNotAllowed)
			json.NewEncoder(c.Response).Encode(map[string]any{
				"status":  "error",
				"message": "method not allowed",
			})
			return nil
		},
		devMode:      false,
		MaxBodyBytes: 1 << 20,
	}
}

// Use appends global middleware to the router.
func (r *Router) Use(middleware ...Middleware) *Router {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.middleware = append(r.middleware, middleware...)
	return r
}

// EnableDevMode turns on verbose request logging.
func (r *Router) EnableDevMode() *Router {
	r.devMode = true
	return r
}

// Routes returns a snapshot of all registered routes.
func (r *Router) Routes() []RouteInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]RouteInfo, len(r.routes))
	copy(out, r.routes)
	return out
}

// Handle registers a handler for the given method and path.
func (r *Router) Handle(method, path string, handler Handler) {
	r.handle(method, path, handler, nil)
}

func (r *Router) handle(method, path string, handler Handler, middleware []Middleware) {
	if handler == nil {
		panic("routix: handler must not be nil")
	}
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	names := make([]string, 0)
	seen := make(map[string]bool)
	for i, part := range parts {
		if part == "*" && i != len(parts)-1 {
			panic("routix: wildcard must be the final path segment")
		}
		name := ""
		if strings.HasPrefix(part, ":") {
			name = part[1:]
			if name == "" {
				panic("routix: parameter name must not be empty")
			}
		}
		if part == "*" {
			name = "*"
		}
		if name != "" {
			if seen[name] {
				panic("routix: duplicate parameter name")
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &node{children: make(map[string]*node)}
	}

	root := r.trees[method]

	for _, part := range parts {
		if part == "" {
			continue
		}
		switch {
		case part[0] == ':':
			paramName := part[1:]
			if root.children[":"] == nil {
				root.children[":"] = &node{
					path:     part,
					children: make(map[string]*node),
					params:   []string{paramName},
					wildcard: true,
				}
			}
			root = root.children[":"]
		case part == "*":
			if root.children["*"] == nil {
				root.children["*"] = &node{
					path:     part,
					children: make(map[string]*node),
					wildcard: true,
				}
			}
			root = root.children["*"]
		default:
			if root.children[part] == nil {
				root.children[part] = &node{
					path:     part,
					children: make(map[string]*node),
				}
			}
			root = root.children[part]
		}

	}
	if root.handler == nil {
		root.routeIndex = len(r.routes)
		r.routes = append(r.routes, RouteInfo{Method: method, Path: path})
	} else {
		r.routes[root.routeIndex] = RouteInfo{Method: method, Path: path}
	}
	root.handler = handler
	root.middleware = append([]Middleware(nil), middleware...)
	root.paramNames = names
}

func (r *Router) GET(path string, handler Handler)     { r.Handle(http.MethodGet, path, handler) }
func (r *Router) POST(path string, handler Handler)    { r.Handle(http.MethodPost, path, handler) }
func (r *Router) PUT(path string, handler Handler)     { r.Handle(http.MethodPut, path, handler) }
func (r *Router) DELETE(path string, handler Handler)  { r.Handle(http.MethodDelete, path, handler) }
func (r *Router) PATCH(path string, handler Handler)   { r.Handle(http.MethodPatch, path, handler) }
func (r *Router) HEAD(path string, handler Handler)    { r.Handle(http.MethodHead, path, handler) }
func (r *Router) OPTIONS(path string, handler Handler) { r.Handle(http.MethodOptions, path, handler) }

func (r *Router) NotFound(handler Handler) { r.mu.Lock(); defer r.mu.Unlock(); r.notFound = handler }
func (r *Router) MethodNotAllowed(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notMethod = handler
}

type routerCachedResponse struct {
	response []byte
	headers  http.Header
	code     int
	expires  time.Time
}

func (r *Router) CacheResponse(key string, response []byte, headers http.Header, code int, duration time.Duration) {
	if duration <= 0 {
		r.cache.Delete(key)
		return
	}
	if len(response) > 1<<20 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	var oldest any
	now := time.Now()
	r.cache.Range(func(k, value any) bool {
		if !now.Before(value.(*routerCachedResponse).expires) {
			r.cache.Delete(k)
		} else {
			count++
			oldest = k
		}
		return true
	})
	if count >= 1024 {
		r.cache.Delete(oldest)
	}
	r.cache.Store(key, &routerCachedResponse{response: append([]byte(nil), response...), headers: headers.Clone(), code: code, expires: now.Add(duration)})
}

func (r *Router) GetCachedResponse(key string) ([]byte, http.Header, int, bool) {
	if value, ok := r.cache.Load(key); ok {
		cached := value.(*routerCachedResponse)
		if time.Now().Before(cached.expires) {
			return append([]byte(nil), cached.response...), cached.headers.Clone(), cached.code, true
		}
		r.cache.CompareAndDelete(key, value)
	}
	return nil, nil, 0, false
}

type headResponseWriter struct{ http.ResponseWriter }

func (w headResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (w headResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if req.Method == http.MethodHead {
		w = headResponseWriter{w}
	}
	rw := &responseWriter{ResponseWriter: w}
	params := r.params.Get().(map[string]string)
	defer func() { clear(params); r.params.Put(params) }()
	var query map[string]string
	if req.URL.RawQuery != "" {
		query = make(map[string]string)
		for k, v := range req.URL.Query() {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}
	}
	ctx := getContextFromPool(req, rw, params, query, nil)
	defer putContextToPool(ctx)

	r.mu.RLock()
	matched, found := r.findRoute(r.trees[req.Method], req.URL.Path, params)
	if !found && req.Method == http.MethodHead {
		matched, found = r.findRoute(r.trees[http.MethodGet], req.URL.Path, params)
	}
	var handler Handler
	if found {
		handler = matched.handler
		ctx.routePath = r.routes[matched.routeIndex].Path
	}
	ctx.clientIP = r.requestClientIP(req)
	ctx.secure = r.requestIsSecure(req)
	if !found {
		allowed := make(map[string]bool)
		for method, root := range r.trees {
			if _, ok := r.findRoute(root, req.URL.Path, make(map[string]string)); ok {
				allowed[method] = true
			}
		}
		if allowed[http.MethodGet] {
			allowed[http.MethodHead] = true
		}
		handler = r.notFound
		if len(allowed) > 0 {
			methods := make([]string, 0, len(allowed))
			for method := range allowed {
				methods = append(methods, method)
			}
			sort.Strings(methods)
			ctx.SetHeader("Allow", strings.Join(methods, ", "))
			handler = r.notMethod
		}
	}
	middleware := append([]Middleware(nil), r.middleware...)
	if found {
		middleware = append(middleware, matched.middleware...)
	}
	maxBody := r.MaxBodyBytes
	ctx.bodyLimit = maxBody
	r.mu.RUnlock()

	ctx.handler = handler
	handler = executeHandler
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	if err := handler(ctx); err != nil {
		writeHandlerError(ctx, err)
	}
}

func executeHandler(c *Context) error {
	if c.routePath != "" && c.IsJSON() && c.Request.Body != nil && c.Request.Body != http.NoBody {
		_ = c.readJSONBody(c.bodyLimit)
	}
	if c.bodyErr != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(c.bodyErr, &tooLarge) {
			return c.JSON(http.StatusRequestEntityTooLarge, map[string]any{"message": "request body too large"})
		}
		return c.BadRequest("invalid JSON body")
	}
	err := c.handler(c)
	writeHandlerError(c, err)
	return err
}

func writeHandlerError(c *Context, err error) {
	if err == nil || c.Writer.written {
		return
	}
	var re *Error
	if errors.As(err, &re) {
		response := ErrorResponse{Code: re.Code, Message: re.Message}
		_ = c.JSON(re.Code, response)
		return
	}
	_ = c.JSON(http.StatusInternalServerError, ErrorResponse{Code: http.StatusInternalServerError, Message: "Internal Server Error"})
}

func (r *Router) findRoute(root *node, path string, params map[string]string) (*node, bool) {
	if root == nil {
		return nil, false
	}
	var match func(*node, int, int) *node
	match = func(current *node, start, paramIndex int) *node {
		for start < len(path) && path[start] == '/' {
			start++
		}
		if start == len(path) {
			if current.handler != nil {
				return current
			}
			if child := current.children["*"]; child != nil && child.handler != nil {
				params[child.paramNames[paramIndex]] = ""
				return child
			}
			return nil
		}
		end := start
		for end < len(path) && path[end] != '/' {
			end++
		}
		part := path[start:end]
		if child := current.children[part]; child != nil && part != ":" && part != "*" {
			if leaf := match(child, end, paramIndex); leaf != nil {
				return leaf
			}
		}
		if child := current.children[":"]; child != nil {
			if leaf := match(child, end, paramIndex+1); leaf != nil {
				params[leaf.paramNames[paramIndex]] = part
				return leaf
			}
		}
		if child := current.children["*"]; child != nil && child.handler != nil {
			params[child.paramNames[paramIndex]] = path[start:]
			return child
		}
		return nil
	}
	if leaf := match(root, 0, 0); leaf != nil {
		return leaf, true
	}
	return nil, false
}

// Group returns a new route group with the given prefix.
func (r *Router) Group(prefix string) *Group {
	return &Group{router: r, prefix: prefix}
}

// Group is a set of routes sharing a common prefix and middleware.
type Group struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

// Use appends middleware to this group and returns the group for chaining.
func (g *Group) Use(middleware ...Middleware) *Group {
	g.middleware = append(g.middleware, middleware...)
	return g
}

func (g *Group) applyMiddleware(handler Handler) Handler {
	for i := len(g.middleware) - 1; i >= 0; i-- {
		handler = g.middleware[i](handler)
	}
	return handler
}

// Group creates a sub-group nested under this group's prefix.
func (g *Group) Group(prefix string) *Group {
	return &Group{
		router:     g.router,
		prefix:     g.prefix + prefix,
		middleware: append([]Middleware{}, g.middleware...),
	}
}

func (g *Group) Handle(method, path string, handler Handler) {
	g.router.handle(method, g.prefix+path, handler, g.middleware)
}

func (g *Group) GET(path string, handler Handler)     { g.Handle(http.MethodGet, path, handler) }
func (g *Group) POST(path string, handler Handler)    { g.Handle(http.MethodPost, path, handler) }
func (g *Group) PUT(path string, handler Handler)     { g.Handle(http.MethodPut, path, handler) }
func (g *Group) DELETE(path string, handler Handler)  { g.Handle(http.MethodDelete, path, handler) }
func (g *Group) PATCH(path string, handler Handler)   { g.Handle(http.MethodPatch, path, handler) }
func (g *Group) HEAD(path string, handler Handler)    { g.Handle(http.MethodHead, path, handler) }
func (g *Group) OPTIONS(path string, handler Handler) { g.Handle(http.MethodOptions, path, handler) }

// Context response helpers

func (c *Context) SetHeader(key, value string) { c.Response.Header().Set(key, value) }
func (c *Context) GetHeader(key string) string { return c.Request.Header.Get(key) }

func (c *Context) Cookie(name string) (*http.Cookie, error) { return c.Request.Cookie(name) }
func (c *Context) SetCookie(cookie *http.Cookie)            { http.SetCookie(c.Response, cookie) }

func (c *Context) String(status int, format string, values ...any) error {
	c.Response.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err := fmt.Fprintf(c.Response, format, values...)
	return err
}

func (c *Context) HTML(status int, html string) error {
	c.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(html))
	return err
}

func (c *Context) Redirect(status int, url string) error {
	c.Response.Header().Set("Location", url)
	c.Response.WriteHeader(status)
	return nil
}

// Start listens on addr and handles graceful shutdown on SIGINT/SIGTERM.
func (r *Router) Start(addr string) error {
	if len(r.trees) == 0 {
		r.GET("/", WelcomeHandler("Routix"))
	}

	if listed, err := r.writeRouteList(); listed {
		return err
	}

	printBanner(addr, false)

	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return listenAndServe(srv)
}

func listenAndServe(srv *http.Server) error {
	return serveWithShutdown(srv, srv.ListenAndServe)
}

func serveWithShutdown(srv *http.Server, serve func() error) error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(quit)

	errCh := make(chan error, 1)
	go func() { errCh <- serve() }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-quit:
		fmt.Println("\nshutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			srv.Close()
			return err
		}
		return nil
	}
}

func printBanner(addr string, devMode bool) {
	fmt.Println()
	fmt.Print("\033[32m")
	fmt.Println(`  ██████╗  ██████╗ ██╗   ██╗████████╗██╗██╗  ██╗`)
	fmt.Println(`  ██╔══██╗██╔═══██╗██║   ██║╚══██╔══╝██║╚██╗██╔╝`)
	fmt.Println(`  ██████╔╝██║   ██║██║   ██║   ██║   ██║ ╚███╔╝ `)
	fmt.Println(`  ██╔══██╗██║   ██║██║   ██║   ██║   ██║ ██╔██╗ `)
	fmt.Println(`  ██║  ██║╚██████╔╝╚██████╔╝   ██║   ██║██╔╝ ██╗`)
	fmt.Println(`  ╚═╝  ╚═╝ ╚═════╝  ╚═════╝    ╚═╝   ╚═╝╚═╝  ╚═╝`)
	fmt.Print("\033[0m")
	fmt.Println()
	fmt.Println("  \033[1mRoutix v0.4.0\033[0m  \033[90mby Ramusa Software Corporation\033[0m")
	fmt.Println()
	fmt.Printf("  \033[36m->\033[0m  http://localhost%s\n", addr)
	if devMode {
		fmt.Printf("  \033[33m->\033[0m  dev mode  metrics: http://localhost%s/_dev/metrics\n", addr)
	}
	fmt.Println()
}
