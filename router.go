// Package routix provides a high-performance HTTP router with an Express.js-like API.
// It features fast routing, middleware support, and a clean, intuitive interface.
package routix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Context represents the request context and provides helper methods for handling HTTP requests.
// It encapsulates the request, response, URL parameters, query parameters, and request body.
type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Params   map[string]string
	Query    map[string]string
	Body     map[string]interface{}
}

// Handler represents a request handler function that processes HTTP requests.
// It receives a Context pointer and returns an error if the request processing fails.
type Handler func(*Context) error

// Router is the main router instance that manages routes, middleware, and request handling.
// It uses a radix tree for efficient route matching and supports parameterized routes.
type Router struct {
	trees      map[string]*node
	params     *sync.Pool
	notFound   Handler
	notMethod  Handler
	middleware []Middleware
	cache      sync.Map
}

// node represents a node in the routing tree.
// Each node can have static children, parameter children, or a wildcard child.
type node struct {
	path     string
	handlers map[string]Handler
	children map[string]*node
	params   []string
	wildcard bool
}

// Middleware represents a middleware function that can be used to process requests
// before they reach their final handler.
type Middleware func(Handler) Handler

// New creates and returns a new router instance with default error handlers.
// The router is initialized with a sync.Pool for parameter maps to reduce allocations.
func New() *Router {
	return &Router{
		trees: make(map[string]*node),
		params: &sync.Pool{
			New: func() interface{} {
				return make(map[string]string)
			},
		},
		notFound: func(c *Context) error {
			http.Error(c.Response, "404 Not Found", http.StatusNotFound)
			return nil
		},
		notMethod: func(c *Context) error {
			http.Error(c.Response, "405 Method Not Allowed", http.StatusMethodNotAllowed)
			return nil
		},
	}
}

// Use adds one or more middleware functions to the router.
// Middleware functions are executed in the order they are added.
func (r *Router) Use(middleware ...Middleware) {
	r.middleware = append(r.middleware, middleware...)
}

// Handle registers a new route with the specified HTTP method and path.
// The path can contain parameters (e.g., /users/:id) and wildcards (e.g., /files/*).
func (r *Router) Handle(method, path string, handler Handler) {
	if path[0] != '/' {
		path = "/" + path
	}

	if _, ok := r.trees[method]; !ok {
		r.trees[method] = &node{
			handlers: make(map[string]Handler),
			children: make(map[string]*node),
		}
	}

	root := r.trees[method]
	parts := strings.Split(path, "/")[1:]

	for i, part := range parts {
		if part == "" {
			continue
		}

		if part[0] == ':' {
			// Parameter node
			paramName := part[1:]
			if root.children[":"] == nil {
				root.children[":"] = &node{
					path:     part,
					handlers: make(map[string]Handler),
					children: make(map[string]*node),
					params:   []string{paramName},
					wildcard: true,
				}
			}
			root = root.children[":"]
		} else if part == "*" {
			// Wildcard node
			if root.children["*"] == nil {
				root.children["*"] = &node{
					path:     part,
					handlers: make(map[string]Handler),
					children: make(map[string]*node),
					wildcard: true,
				}
			}
			root = root.children["*"]
		} else {
			// Static node
			if root.children[part] == nil {
				root.children[part] = &node{
					path:     part,
					handlers: make(map[string]Handler),
					children: make(map[string]*node),
				}
			}
			root = root.children[part]
		}

		if i == len(parts)-1 {
			root.handlers[method] = handler
		}
	}
}

// GET registers a new GET route with the specified path and handler.
func (r *Router) GET(path string, handler Handler) {
	r.Handle(http.MethodGet, path, handler)
}

// POST registers a new POST route with the specified path and handler.
func (r *Router) POST(path string, handler Handler) {
	r.Handle(http.MethodPost, path, handler)
}

// PUT registers a new PUT route with the specified path and handler.
func (r *Router) PUT(path string, handler Handler) {
	r.Handle(http.MethodPut, path, handler)
}

// DELETE registers a new DELETE route with the specified path and handler.
func (r *Router) DELETE(path string, handler Handler) {
	r.Handle(http.MethodDelete, path, handler)
}

// PATCH registers a new PATCH route with the specified path and handler.
func (r *Router) PATCH(path string, handler Handler) {
	r.Handle(http.MethodPatch, path, handler)
}

// NotFound sets a custom handler for 404 Not Found responses.
func (r *Router) NotFound(handler Handler) {
	r.notFound = handler
}

// MethodNotAllowed sets a custom handler for 405 Method Not Allowed responses.
func (r *Router) MethodNotAllowed(handler Handler) {
	r.notMethod = handler
}

// CacheResponse caches a response for the given duration
func (r *Router) CacheResponse(key string, response []byte, headers http.Header, code int, duration time.Duration) {
	r.cache.Store(key, struct {
		response []byte
		headers  http.Header
		code     int
		expires  time.Time
	}{
		response: response,
		headers:  headers,
		code:     code,
		expires:  time.Now().Add(duration),
	})
}

// GetCachedResponse gets a cached response if it exists and hasn't expired
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
		// Remove expired cache
		r.cache.Delete(key)
	}
	return nil, nil, 0, false
}

// ServeHTTP implements the http.Handler interface.
// It processes incoming HTTP requests by finding the appropriate handler
// and executing any middleware functions.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	path := req.URL.Path
	method := req.Method

	// Check cache for GET requests
	if method == http.MethodGet {
		if response, headers, code, ok := r.GetCachedResponse(path); ok {
			// Write cached response
			for k, v := range headers {
				w.Header()[k] = v
			}
			w.WriteHeader(code)
			w.Write(response)
			return
		}
	}

	// Get the root node for the method
	root, ok := r.trees[method]
	if !ok {
		r.notMethod(&Context{Request: req, Response: w})
		return
	}

	// Get params from pool
	params := r.params.Get().(map[string]string)
	defer r.params.Put(params)

	// Parse query parameters
	query := make(map[string]string)
	for k, v := range req.URL.Query() {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}

	// Parse body if it's JSON
	var body map[string]interface{}
	if req.Header.Get("Content-Type") == "application/json" {
		json.NewDecoder(req.Body).Decode(&body)
	}

	// Create context
	ctx := &Context{
		Request:  req,
		Response: w,
		Params:   params,
		Query:    query,
		Body:     body,
	}

	// Find the handler
	handler, found := r.findHandler(root, path, params)
	if !found {
		r.notFound(ctx)
		return
	}

	// Apply middleware
	h := handler
	for i := len(r.middleware) - 1; i >= 0; i-- {
		h = r.middleware[i](h)
	}

	// Call the handler
	if err := h(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// findHandler finds the handler for the given path and extracts URL parameters.
// It returns the handler and a boolean indicating whether a handler was found.
func (r *Router) findHandler(root *node, path string, params map[string]string) (Handler, bool) {
	parts := strings.Split(path, "/")[1:]
	current := root

	for i, part := range parts {
		if part == "" {
			continue
		}

		// Try static match first
		if child, ok := current.children[part]; ok {
			current = child
			continue
		}

		// Try parameter match
		if child, ok := current.children[":"]; ok {
			current = child
			params[child.params[0]] = part
			continue
		}

		// Try wildcard match
		if child, ok := current.children["*"]; ok {
			current = child
			params["*"] = strings.Join(parts[i:], "/")
			break
		}

		return nil, false
	}

	handler, ok := current.handlers[current.path]
	return handler, ok
}

// Group creates a new route group with the specified prefix.
// Route groups allow you to apply common middleware and prefixes to multiple routes.
func (r *Router) Group(prefix string) *Group {
	return &Group{
		router: r,
		prefix: prefix,
	}
}

// Group represents a group of routes
type Group struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

// Use adds middleware to the group
func (g *Group) Use(middleware ...Middleware) {
	g.middleware = append(g.middleware, middleware...)
}

// GET registers a new GET route in the group.
func (g *Group) GET(path string, handler Handler) {
	g.router.GET(g.prefix+path, handler)
}

// POST registers a new POST route in the group.
func (g *Group) POST(path string, handler Handler) {
	g.router.POST(g.prefix+path, handler)
}

// PUT registers a new PUT route in the group.
func (g *Group) PUT(path string, handler Handler) {
	g.router.PUT(g.prefix+path, handler)
}

// DELETE registers a new DELETE route in the group.
func (g *Group) DELETE(path string, handler Handler) {
	g.router.DELETE(g.prefix+path, handler)
}

// PATCH registers a new PATCH route in the group.
func (g *Group) PATCH(path string, handler Handler) {
	g.router.PATCH(g.prefix+path, handler)
}

// String sends a plain text response with the specified status code and format.
func (c *Context) String(status int, format string, values ...interface{}) error {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(status)
	_, err := fmt.Fprintf(c.Response, format, values...)
	return err
}

// HTML sends an HTML response with the specified status code.
func (c *Context) HTML(status int, html string) error {
	c.Response.Header().Set("Content-Type", "text/html")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(html))
	return err
}

// Redirect sends a redirect response with the specified status code and URL.
func (c *Context) Redirect(status int, url string) error {
	c.Response.Header().Set("Location", url)
	c.Response.WriteHeader(status)
	return nil
}

// SetHeader sets a response header with the specified key and value.
func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

// GetHeader gets a request header value for the specified key.
func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

// Cookie gets a cookie from the request by name.
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

// SetCookie sets a cookie in the response.
func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response, cookie)
}

// Cache caches the response for the given duration
func (c *Context) Cache(duration time.Duration) {
	// Create a response recorder
	recorder := httptest.NewRecorder()

	// Create a new context with the recorder
	newCtx := &Context{
		Request:  c.Request,
		Response: recorder,
		Params:   c.Params,
		Query:    c.Query,
		Body:     c.Body,
	}

	// Process the request with the new context
	if err := c.Request.Context().Value("handler").(Handler)(newCtx); err != nil {
		return
	}

	// Get the router
	router := c.Request.Context().Value("router").(*Router)

	// Cache the response
	router.CacheResponse(
		c.Request.URL.Path,
		recorder.Body.Bytes(),
		recorder.Header(),
		recorder.Code,
		duration,
	)
}
