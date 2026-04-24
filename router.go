package routix

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

func (rw *responseWriter) Status() int {
	if rw.status == 0 {
		return http.StatusOK
	}
	return rw.status
}

// Context holds request/response state for a single HTTP request.
type Context struct {
	Request  *http.Request
	Writer   *responseWriter
	Response http.ResponseWriter // kept for backward compat; points to Writer
	Params   map[string]string
	Query    map[string]string
	Body     map[string]any
	values   map[string]any
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
	trees      map[string]*node
	routes     []RouteInfo
	params     *sync.Pool
	notFound   Handler
	notMethod  Handler
	middleware []Middleware
	cache      sync.Map
	devMode    bool
	mu         sync.RWMutex
}

type node struct {
	path     string
	handler  Handler
	children map[string]*node
	params   []string
	wildcard bool
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
		devMode: false,
	}
}

// Use appends global middleware to the router.
func (r *Router) Use(middleware ...Middleware) *Router {
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
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}

	r.mu.Lock()
	r.routes = append(r.routes, RouteInfo{Method: method, Path: path})
	r.mu.Unlock()

	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &node{children: make(map[string]*node)}
	}

	root := r.trees[method]

	if path == "/" {
		root.handler = handler
		return
	}

	parts := strings.Split(path[1:], "/")
	for i, part := range parts {
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

		if i == len(parts)-1 {
			root.handler = handler
		}
	}
}

func (r *Router) GET(path string, handler Handler)     { r.Handle(http.MethodGet, path, handler) }
func (r *Router) POST(path string, handler Handler)    { r.Handle(http.MethodPost, path, handler) }
func (r *Router) PUT(path string, handler Handler)     { r.Handle(http.MethodPut, path, handler) }
func (r *Router) DELETE(path string, handler Handler)  { r.Handle(http.MethodDelete, path, handler) }
func (r *Router) PATCH(path string, handler Handler)   { r.Handle(http.MethodPatch, path, handler) }
func (r *Router) HEAD(path string, handler Handler)    { r.Handle(http.MethodHead, path, handler) }
func (r *Router) OPTIONS(path string, handler Handler) { r.Handle(http.MethodOptions, path, handler) }

func (r *Router) NotFound(handler Handler)        { r.notFound = handler }
func (r *Router) MethodNotAllowed(handler Handler) { r.notMethod = handler }

func (r *Router) CacheResponse(key string, response []byte, headers http.Header, code int, duration time.Duration) {
	r.cache.Store(key, struct {
		response []byte
		headers  http.Header
		code     int
		expires  time.Time
	}{response, headers, code, time.Now().Add(duration)})
}

func (r *Router) GetCachedResponse(key string) ([]byte, http.Header, int, bool) {
	if value, ok := r.cache.Load(key); ok {
		cached := value.(struct {
			response []byte
			headers  http.Header
			code     int
			expires  time.Time
		})
		if time.Now().Before(cached.expires) {
			return cached.response, cached.headers, cached.code, true
		}
		r.cache.Delete(key)
	}
	return nil, nil, 0, false
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	// Serve cached GET responses without hitting the handler chain.
	if method == http.MethodGet {
		if response, headers, code, ok := r.GetCachedResponse(path); ok {
			for k, v := range headers {
				w.Header()[k] = v
			}
			w.WriteHeader(code)
			w.Write(response)
			return
		}
	}

	rw := &responseWriter{ResponseWriter: w}

	root, ok := r.trees[method]
	if !ok {
		r.notMethod(getContextFromPool(req, rw, nil, nil, nil))
		return
	}

	params := r.params.Get().(map[string]string)
	defer func() {
		for k := range params {
			delete(params, k)
		}
		r.params.Put(params)
	}()

	var query map[string]string
	if req.URL.RawQuery != "" {
		query = make(map[string]string)
		for k, v := range req.URL.Query() {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}
	}

	// Parse JSON body when content-type is application/json.
	// ContentLength == -1 means chunked; still attempt decode.
	var body map[string]any
	ct := req.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "application/json") && req.Body != nil {
		json.NewDecoder(req.Body).Decode(&body) //nolint:errcheck
	}

	ctx := getContextFromPool(req, rw, params, query, body)
	defer putContextToPool(ctx)

	handler, found := r.findHandler(root, path, params)
	if !found {
		r.notFound(ctx)
		return
	}

	h := handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	if err := h(ctx); err != nil {
		if routixErr, ok := err.(*Error); ok {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(routixErr.Code)
			json.NewEncoder(w).Encode(routixErr.ToResponse())
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func (r *Router) findHandler(root *node, path string, params map[string]string) (Handler, bool) {
	if path == "/" {
		if root.handler != nil {
			return root.handler, true
		}
		return nil, false
	}

	pathLen := len(path)
	current := root
	start := 1

	for start < pathLen {
		end := start
		for end < pathLen && path[end] != '/' {
			end++
		}
		if start == end {
			start++
			continue
		}

		part := path[start:end]

		if child, ok := current.children[part]; ok {
			current = child
			start = end + 1
			continue
		}

		if child, ok := current.children[":"]; ok {
			if len(child.params) > 0 {
				params[child.params[0]] = part
			}
			current = child
			start = end + 1
			continue
		}

		if child, ok := current.children["*"]; ok {
			params["*"] = path[start:]
			current = child
			break
		}

		return nil, false
	}

	if current.handler != nil {
		return current.handler, true
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
	g.router.Handle(method, g.prefix+path, g.applyMiddleware(handler))
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
func (c *Context) GetHeader(key string) string  { return c.Request.Header.Get(key) }

func (c *Context) Cookie(name string) (*http.Cookie, error) { return c.Request.Cookie(name) }
func (c *Context) SetCookie(cookie *http.Cookie)             { http.SetCookie(c.Response, cookie) }

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
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-quit:
		fmt.Println("\nshutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(ctx)
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
