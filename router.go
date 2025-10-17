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

type Context struct {
	Request  *http.Request
	Response http.ResponseWriter
	Params   map[string]string
	Query    map[string]string
	Body     map[string]interface{}
	index    int8
	handlers []Handler
}



func getContextFromPool(req *http.Request, w http.ResponseWriter, params, query map[string]string, body map[string]interface{}) *Context {
	ctx := getContext()
	ctx.Request = req
	ctx.Response = w
	ctx.Params = params
	ctx.Query = query
	ctx.Body = body
	ctx.index = -1
	return ctx
}

func putContextToPool(ctx *Context) {
	ctx.Request = nil
	ctx.Response = nil
	ctx.Params = nil
	ctx.Query = nil
	ctx.Body = nil
	ctx.handlers = nil
	putContext(ctx)
}

type Handler func(*Context) error

type Router struct {
	trees      map[string]*node
	params     *sync.Pool
	notFound   Handler
	notMethod  Handler
	middleware []Middleware
	cache      sync.Map
	devMode    bool
}

type node struct {
	path     string
	handlers map[string]Handler
	children map[string]*node
	params   []string
	wildcard bool
}

type Middleware func(Handler) Handler

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

func (r *Router) Use(middleware ...Middleware) *Router {
	r.middleware = append(r.middleware, middleware...)
	return r
}

// EnableDevMode enables development mode for the router
func (r *Router) EnableDevMode() *Router {
	r.devMode = true
	return r
}

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

func (r *Router) GET(path string, handler Handler) {
	r.Handle(http.MethodGet, path, handler)
}

func (r *Router) POST(path string, handler Handler) {
	r.Handle(http.MethodPost, path, handler)
}

func (r *Router) PUT(path string, handler Handler) {
	r.Handle(http.MethodPut, path, handler)
}

func (r *Router) DELETE(path string, handler Handler) {
	r.Handle(http.MethodDelete, path, handler)
}

func (r *Router) PATCH(path string, handler Handler) {
	r.Handle(http.MethodPatch, path, handler)
}

func (r *Router) NotFound(handler Handler) {
	r.notFound = handler
}

func (r *Router) MethodNotAllowed(handler Handler) {
	r.notMethod = handler
}

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
	root, ok := r.trees[method]
	if !ok {
		r.notMethod(getContextFromPool(req, w, nil, nil, nil))
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
	var body map[string]interface{}
	contentType := req.Header.Get("Content-Type")
	if contentType == "application/json" && req.ContentLength > 0 {
		json.NewDecoder(req.Body).Decode(&body)
	}

	ctx := getContextFromPool(req, w, params, query, body)
	defer putContextToPool(ctx)

	handler, found := r.findHandlerOptimized(root, path, params)
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

func (r *Router) findHandlerOptimized(root *node, path string, params map[string]string) (Handler, bool) {
	if path == "/" {
		if handler, ok := root.handlers["GET"]; ok {
			return handler, true
		}
	}
	pathLen := len(path)
	if pathLen == 0 {
		return nil, false
	}

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
			current = child
			if len(child.params) > 0 {
				params[child.params[0]] = part
			}
			start = end + 1
			continue
		}

		if child, ok := current.children["*"]; ok {
			current = child
			if start < pathLen {
				params["*"] = path[start:]
			}
			break
		}

		return nil, false
	}

	for method, handler := range current.handlers {
		if method != "" {
			return handler, true
		}
	}
	
	return nil, false
}

func (r *Router) findHandler(root *node, path string, params map[string]string) (Handler, bool) {
	return r.findHandlerOptimized(root, path, params)
}

func (r *Router) Group(prefix string) *Group {
	return &Group{
		router: r,
		prefix: prefix,
	}
}

type Group struct {
	router     *Router
	prefix     string
	middleware []Middleware
}

func (g *Group) Use(middleware ...Middleware) {
	g.middleware = append(g.middleware, middleware...)
}

func (g *Group) GET(path string, handler Handler) {
	g.router.GET(g.prefix+path, handler)
}

func (g *Group) POST(path string, handler Handler) {
	g.router.POST(g.prefix+path, handler)
}

func (g *Group) PUT(path string, handler Handler) {
	g.router.PUT(g.prefix+path, handler)
}

func (g *Group) DELETE(path string, handler Handler) {
	g.router.DELETE(g.prefix+path, handler)
}

func (g *Group) PATCH(path string, handler Handler) {
	g.router.PATCH(g.prefix+path, handler)
}

func (c *Context) String(status int, format string, values ...interface{}) error {
	c.Response.Header().Set("Content-Type", "text/plain")
	c.Response.WriteHeader(status)
	_, err := fmt.Fprintf(c.Response, format, values...)
	return err
}

func (c *Context) HTML(status int, html string) error {
	c.Response.Header().Set("Content-Type", "text/html")
	c.Response.WriteHeader(status)
	_, err := c.Response.Write([]byte(html))
	return err
}

func (c *Context) Redirect(status int, url string) error {
	c.Response.Header().Set("Location", url)
	c.Response.WriteHeader(status)
	return nil
}

func (c *Context) SetHeader(key, value string) {
	c.Response.Header().Set(key, value)
}

func (c *Context) GetHeader(key string) string {
	return c.Request.Header.Get(key)
}

func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.Request.Cookie(name)
}

func (c *Context) SetCookie(cookie *http.Cookie) {
	http.SetCookie(c.Response, cookie)
}

func (r *Router) Start(addr string) error {
	fmt.Println("Routix Framework")
	fmt.Printf(" Routix server starting on %s\n", addr)
	return http.ListenAndServe(addr, r)
}

func (c *Context) Cache(duration time.Duration) {
	recorder := httptest.NewRecorder()
	newCtx := &Context{
		Request:  c.Request,
		Response: recorder,
		Params:   c.Params,
		Query:    c.Query,
		Body:     c.Body,
	}

	if err := c.Request.Context().Value("handler").(Handler)(newCtx); err != nil {
		return
	}

	router := c.Request.Context().Value("router").(*Router)
	router.CacheResponse(
		c.Request.URL.Path,
		recorder.Body.Bytes(),
		recorder.Header(),
		recorder.Code,
		duration,
	)
}
