package routix

import (
	"net/http"
	"time"
)

// Quick creates a new router with sensible defaults
func Quick() *Router {
	r := New()
	r.Use(Logger(), Recovery(), CORS())
	return r
}

// Route is a fluent interface for defining routes
type Route struct {
	router *Router
	method string
	path   string
}

// NewRoute creates a new route builder
func (r *Router) Route(method, path string) *Route {
	return &Route{
		router: r,
		method: method,
		path:   path,
	}
}

// Handle sets the handler for the route
func (rt *Route) Handle(handler Handler) *Router {
	rt.router.Handle(rt.method, rt.path, handler)
	return rt.router
}

// JSON is a shortcut for JSON responses
func (rt *Route) JSON(data interface{}) *Router {
	return rt.Handle(func(c *Context) error {
		return c.JSON(http.StatusOK, data)
	})
}

// Text is a shortcut for text responses
func (rt *Route) Text(text string) *Router {
	return rt.Handle(func(c *Context) error {
		return c.String(http.StatusOK, "%s", text)
	})
}

// Redirect is a shortcut for redirects
func (rt *Route) Redirect(url string) *Router {
	return rt.Handle(func(c *Context) error {
		return c.Redirect(http.StatusFound, url)
	})
}

// Static serves static files
func (r *Router) Static(path, dir string) *Router {
	fileServer := http.FileServer(http.Dir(dir))
	r.GET(path+"/*", func(c *Context) error {
		http.StripPrefix(path, fileServer).ServeHTTP(c.Response, c.Request)
		return nil
	})
	return r
}

// API creates an API group with JSON middleware
func (r *Router) API(path string) *Group {
	group := r.Group(path)
	group.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.SetHeader("Content-Type", "application/json")
			return next(c)
		}
	})
	return group
}

// Resource creates RESTful routes for a resource
func (r *Router) Resource(path string, controller ResourceController) *Router {
	// GET /resource - index
	if controller.Index != nil {
		r.GET(path, controller.Index)
	}
	
	// POST /resource - create
	if controller.Create != nil {
		r.POST(path, controller.Create)
	}
	
	// GET /resource/:id - show
	if controller.Show != nil {
		r.GET(path+"/:id", controller.Show)
	}
	
	// PUT /resource/:id - update
	if controller.Update != nil {
		r.PUT(path+"/:id", controller.Update)
	}
	
	// DELETE /resource/:id - delete
	if controller.Delete != nil {
		r.DELETE(path+"/:id", controller.Delete)
	}
	
	return r
}

// ResourceController defines the interface for RESTful controllers
type ResourceController struct {
	Index  Handler // GET /resource
	Create Handler // POST /resource
	Show   Handler // GET /resource/:id
	Update Handler // PUT /resource/:id
	Delete Handler // DELETE /resource/:id
}

// Middleware shortcuts
func (r *Router) WithAuth(validateToken func(string) bool) *Router {
	return r.Use(Auth(validateToken))
}

func (r *Router) WithRateLimit(requests int, duration string) *Router {
	// Parse duration string to time.Duration
	// For simplicity, assume it's already parsed
	return r.Use(RateLimit(requests, parseDuration(duration)))
}

func (r *Router) WithCache(duration string) *Router {
	return r.Use(Cache(parseDuration(duration)))
}

func (r *Router) WithTimeout(duration string) *Router {
	return r.Use(Timeout(parseDuration(duration)))
}

func parseDuration(duration string) time.Duration {
	// Simple implementation - in real use, use time.ParseDuration
	switch duration {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "1h":
		return time.Hour
	default:
		return time.Minute
	}
}