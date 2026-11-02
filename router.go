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
	"sync/atomic"
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
	Request       *http.Request
	Writer        *responseWriter
	Response      http.ResponseWriter // kept for backward compat; points to Writer
	Params        map[string]string
	Query         map[string]string
	Body          map[string]any
	writer        responseWriter
	jsonWriter    jsonResponseWriter
	requestParams map[string]string
	routePath     string
	requestPeer   string
	bodyLimit     int64
	secure        bool
	dirty         bool
	bodyRead      bool
	handler       Handler
	values        map[string]any
	rawBody       []byte
	bodyErr       error
	clientIP      string
}

// Set stores a value in the context, scoped to this request.
func (c *Context) Set(key string, value any) {
	c.dirty = true
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

func (c *Context) ResponseWritten() bool { return c.Writer.written }

func putContextToPool(ctx *Context) {
	if len(ctx.requestParams) != 0 {
		clear(ctx.requestParams)
	}
	ctx.Request = nil
	ctx.Writer = nil
	ctx.Response = nil
	ctx.Params = nil
	if ctx.Query != nil {
		ctx.Query = nil
	}
	if ctx.Body != nil {
		ctx.Body = nil
	}
	if ctx.dirty {
		if ctx.values != nil {
			ctx.values = nil
		}
		if ctx.rawBody != nil {
			ctx.rawBody = nil
		}
		if ctx.bodyErr != nil {
			ctx.bodyErr = nil
		}
		ctx.bodyRead = false
		if ctx.handler != nil {
			ctx.handler = nil
		}
		if ctx.clientIP != "" {
			ctx.clientIP = ""
		}
		if ctx.jsonWriter.response != nil || ctx.jsonWriter.status != 0 {
			ctx.jsonWriter = jsonResponseWriter{}
		}
		ctx.dirty = false
	}
	ctx.bodyLimit = 0
	ctx.routePath = ""
	ctx.requestPeer = ""
	ctx.secure = false
	ctx.writer = responseWriter{}
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
	notFound       Handler
	notMethod      Handler
	middleware     []Middleware
	cache          sync.Map
	devMode        bool
	MaxBodyBytes   int64
	trustedProxies []*net.IPNet
	mu             sync.RWMutex
	dispatch       atomic.Pointer[routingState]
	disablePooling atomic.Bool
}

type node struct {
	handler       Handler
	routePath     string
	paramName     string
	middleware    []Middleware
	paramNames    []string
	children      map[string]*node
	paramChild    *node
	wildcardChild *node
	routeIndex    int
}

// Middleware wraps a Handler with additional logic.
type Middleware func(Handler) Handler

// New creates a new Router with sensible defaults.
func New() *Router {
	return &Router{
		trees: make(map[string]*node),
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
	r.dispatch.Store(nil)
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
			if root.children[":"] == nil {
				root.children[":"] = &node{
					children: make(map[string]*node),
				}
			}
			root.paramChild = root.children[":"]
			root = root.paramChild
		case part == "*":
			if root.children["*"] == nil {
				root.children["*"] = &node{
					children: make(map[string]*node),
				}
			}
			root.wildcardChild = root.children["*"]
			root = root.wildcardChild
		default:
			if root.children[part] == nil {
				root.children[part] = &node{
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
	root.routePath = path
	r.dispatch.Store(nil)
}

func (r *Router) GET(path string, handler Handler)     { r.Handle(http.MethodGet, path, handler) }
func (r *Router) POST(path string, handler Handler)    { r.Handle(http.MethodPost, path, handler) }
func (r *Router) PUT(path string, handler Handler)     { r.Handle(http.MethodPut, path, handler) }
func (r *Router) DELETE(path string, handler Handler)  { r.Handle(http.MethodDelete, path, handler) }
func (r *Router) PATCH(path string, handler Handler)   { r.Handle(http.MethodPatch, path, handler) }
func (r *Router) HEAD(path string, handler Handler)    { r.Handle(http.MethodHead, path, handler) }
func (r *Router) OPTIONS(path string, handler Handler) { r.Handle(http.MethodOptions, path, handler) }

func (r *Router) NotFound(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notFound = handler
	r.dispatch.Store(nil)
}
func (r *Router) MethodNotAllowed(handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.notMethod = handler
	r.dispatch.Store(nil)
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
	pooled := !r.disablePooling.Load()
	var query map[string]string
	if req.URL.RawQuery != "" {
		query = make(map[string]string)
		for k, v := range req.URL.Query() {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}
	}
	var ctx *Context
	if pooled {
		ctx = getContext()
	} else {
		ctx = &Context{requestParams: make(map[string]string)}
	}
	ctx.Request = req
	ctx.writer.ResponseWriter = w
	ctx.Writer = &ctx.writer
	ctx.Response = &ctx.writer
	ctx.Params = ctx.requestParams
	ctx.Query = query
	params := ctx.Params
	if pooled {
		defer putContextToPool(ctx)
	}

	state := r.dispatch.Load()
	if state == nil {
		state = r.routingState()
	}
	path := req.URL.Path
	routes := state.routesFor(req.Method)
	retried := false
	var matched *node
	for {
		if routes != nil {
			if n := routes.prefixes; n != nil && strings.HasPrefix(path, n.prefix) {
				rest := path[len(n.prefix):]
				for n.route == nil && len(rest) != 0 {
					var child *routePrefix
					skip := 1
					if flat := n.flat; flat != nil {
						skip = int(flat.depth)
						if len(rest) < skip {
							break
						}
						var index int
						if flat.lookup != nil {
							index = int(flat.lookup[0][rest[0]]) + int(flat.lookup[1][rest[1]])
							if skip == 3 {
								index += int(flat.lookup[2][rest[2]])
							}
							if index >= len(n.children) {
								break
							}
						} else {
							first := int(rest[0]) - int(flat.lows[0])
							second := int(rest[1]) - int(flat.lows[1])
							if uint(first) >= uint(flat.widths[0]) || uint(second) >= uint(flat.widths[1]) {
								break
							}
							index = first*int(flat.widths[1]) + second
							if skip == 3 {
								third := int(rest[2]) - int(flat.lows[2])
								if uint(third) >= uint(flat.widths[2]) {
									break
								}
								index = index*int(flat.widths[2]) + third
							}
						}
						child = n.children[index]
					} else {
						child = n.child(rest[0])
					}
					if child == nil {
						break
					}
					if len(child.prefix) > 1 && !strings.HasPrefix(rest[skip:], child.prefix[1:]) {
						break
					}
					rest = rest[skip+len(child.prefix)-1:]
					n = child
				}
				if route := n.route; route != nil {
					if n.wildcard {
						for len(rest) > 0 && rest[0] == '/' {
							rest = rest[1:]
						}
						params["*"] = rest
						matched = route
						break
					}
					if rest != "" && strings.IndexByte(rest, '/') < 0 {
						params[route.paramName] = rest
						matched = route
						break
					}
				}
			}
			if len(routes.static) != 0 {
				matched = routes.static[path]
				if matched != nil {
					break
				}
			}
			matched, _ = matchRoute(routes.root, path, params)
			if matched != nil {
				break
			}
		}
		if req.Method != http.MethodHead || retried {
			break
		}
		retried = true
		routes = state.get
	}
	found := matched != nil
	var handler Handler
	if found {
		handler = matched.handler
		ctx.routePath = matched.routePath
	}
	ctx.requestPeer = req.RemoteAddr
	ctx.secure = req.TLS != nil
	if len(state.proxies) != 0 {
		ctx.dirty = true
		ctx.clientIP = requestClientIP(req, state.proxies)
		ctx.secure = requestIsSecure(req, state.proxies)
	}
	if !found {
		allowed := make(map[string]bool)
		for method, routes := range state.methods {
			if _, ok := matchRoute(routes.root, path, make(map[string]string)); ok {
				allowed[method] = true
			}
		}
		if allowed[http.MethodGet] {
			allowed[http.MethodHead] = true
		}
		handler = state.notFound
		if len(allowed) > 0 {
			methods := make([]string, 0, len(allowed))
			for method := range allowed {
				methods = append(methods, method)
			}
			sort.Strings(methods)
			ctx.SetHeader("Allow", strings.Join(methods, ", "))
			handler = state.notMethod
		}
	}
	ctx.bodyLimit = r.MaxBodyBytes
	if len(state.middleware) != 0 || found && len(matched.middleware) != 0 || req.Body != nil && req.Body != http.NoBody {
		ctx.dirty = true
		ctx.handler = handler
		handler = executeHandler
	}
	if found {
		for i := len(matched.middleware) - 1; i >= 0; i-- {
			handler = matched.middleware[i](handler)
		}
	}
	for i := len(state.middleware) - 1; i >= 0; i-- {
		handler = state.middleware[i](handler)
	}
	if err := handler(ctx); err != nil {
		writeHandlerError(ctx, err)
	}
}

func executeHandler(c *Context) error {
	if c.routePath != "" && c.Request.Body != nil && c.Request.Body != http.NoBody && c.IsJSON() {
		_ = c.readJSONBody(c.bodyLimit)
	}
	if c.bodyErr != nil {
		var timedOut net.Error
		if errors.As(c.bodyErr, &timedOut) && timedOut.Timeout() {
			return NewError(http.StatusServiceUnavailable, "Request timed out", c.bodyErr)
		}
		var tooLarge *http.MaxBytesError
		if errors.As(c.bodyErr, &tooLarge) {
			return NewError(http.StatusRequestEntityTooLarge, "request body too large", c.bodyErr)
		}
		return BadRequest("invalid JSON body", c.bodyErr)
	}
	return c.handler(c)
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
	status := GetHTTPStatusCode(err)
	_ = c.JSON(status, ErrorResponse{Code: status, Message: http.StatusText(status)})
}

func (r *Router) findRoute(root *node, path string, params map[string]string) (*node, bool) {
	return matchRoute(root, path, params)
}

func matchRoute(root *node, path string, params map[string]string) (*node, bool) {
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
			if child := current.wildcardChild; child != nil && child.handler != nil {
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
		if child := current.paramChild; child != nil {
			if leaf := match(child, end, paramIndex+1); leaf != nil {
				params[leaf.paramNames[paramIndex]] = part
				return leaf
			}
		}
		if child := current.wildcardChild; child != nil && child.handler != nil {
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
	if router, ok := srv.Handler.(*Router); ok {
		router.routingState()
	}
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
