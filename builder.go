package routix

import (
	"fmt"
	"net/http"
	"time"
)

// DevMode indicates if the application is running in development mode
var DevMode bool

// APIBuilder provides a fluent interface for building APIs
type APIBuilder struct {
	router *Router
}

// NewAPI creates a new API builder with optimized defaults
func NewAPI() *APIBuilder {
	router := New()
	
	// Add performance optimizations by default
	router.Use(PerformanceMonitor())
	
	return &APIBuilder{
		router: router,
	}
}

// Dev enables development mode
func (api *APIBuilder) Dev() *APIBuilder {
	api.router.EnableDevMode()
	return api
}

// Prod enables production optimizations
func (api *APIBuilder) Prod() *APIBuilder {
	// Production optimizations
	api.router.Use(
		Recovery(),
		Logger(),
		Compress(),
	)
	return api
}

// CORS enables CORS with default settings
func (api *APIBuilder) CORS() *APIBuilder {
	api.router.Use(CORS())
	return api
}

// Auth adds authentication
func (api *APIBuilder) Auth(validator func(string) bool) *APIBuilder {
	api.router.Use(Auth(validator))
	return api
}

// RateLimit adds rate limiting
func (api *APIBuilder) RateLimit(requests int, window string) *APIBuilder {
	duration := parseDuration(window)
	api.router.Use(RateLimit(requests, duration))
	return api
}

// Cache adds response caching
func (api *APIBuilder) Cache(duration string) *APIBuilder {
	d := parseDuration(duration)
	api.router.Use(Cache(d))
	return api
}

// Timeout adds request timeout
func (api *APIBuilder) Timeout(duration string) *APIBuilder {
	d := parseDuration(duration)
	api.router.Use(Timeout(d))
	return api
}

// JSON sets up JSON API defaults
func (api *APIBuilder) JSON() *APIBuilder {
	api.router.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.SetHeader("Content-Type", "application/json; charset=utf-8")
			return next(c)
		}
	})
	return api
}

// Health adds health check endpoint
func (api *APIBuilder) Health(path string) *APIBuilder {
	if path == "" {
		path = "/health"
	}
	
	api.router.GET(path, func(c *Context) error {
		return c.Success(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "routix-api",
		})
	})
	return api
}

// Metrics adds metrics endpoint
func (api *APIBuilder) Metrics(path string) *APIBuilder {
	if path == "" {
		path = "/metrics"
	}
	
	api.router.GET(path, func(c *Context) error {
		return c.JSON(http.StatusOK, globalMetrics.GetMetrics())
	})
	return api
}

// Static serves static files
func (api *APIBuilder) Static(path, dir string) *APIBuilder {
	api.router.Static(path, dir)
	return api
}

// Group creates a route group
func (api *APIBuilder) Group(path string, fn func(*Group)) *APIBuilder {
	group := api.router.Group(path)
	fn(group)
	return api
}

// V1, V2, V3 - Version shortcuts
func (api *APIBuilder) V1(fn func(*Group)) *APIBuilder {
	return api.Group("/v1", fn)
}

func (api *APIBuilder) V2(fn func(*Group)) *APIBuilder {
	return api.Group("/v2", fn)
}

func (api *APIBuilder) V3(fn func(*Group)) *APIBuilder {
	return api.Group("/v3", fn)
}

// CRUD creates full CRUD endpoints for a resource
func (api *APIBuilder) CRUD(path string, controller ResourceController) *APIBuilder {
	api.router.Resource(path, controller)
	return api
}

// Route shortcuts
func (api *APIBuilder) GET(path string, handler Handler) *APIBuilder {
	api.router.GET(path, handler)
	return api
}

func (api *APIBuilder) POST(path string, handler Handler) *APIBuilder {
	api.router.POST(path, handler)
	return api
}

func (api *APIBuilder) PUT(path string, handler Handler) *APIBuilder {
	api.router.PUT(path, handler)
	return api
}

func (api *APIBuilder) DELETE(path string, handler Handler) *APIBuilder {
	api.router.DELETE(path, handler)
	return api
}

func (api *APIBuilder) PATCH(path string, handler Handler) *APIBuilder {
	api.router.PATCH(path, handler)
	return api
}

// Build returns the configured router
func (api *APIBuilder) Build() *Router {
	return api.router
}

// Start starts the server
func (api *APIBuilder) Start(addr string) error {
	fmt.Println(`
______                _    _        ______                                                       _
| ___ \              | |  (_)       |  ___|                                                     | |
| |_/ /  ___   _   _ | |_  _ __  __ | |_    _ __   __ _  _ __ ___    ___ __      __  ___   _ __ | | __
|    /  / _ \ | | | || __|| |\ \/ / |  _|  | '__| / _` || _ ` _ \  / _ \\ \ /\ / / / _ \ | __|| |/ /
| |\ \ | (_) || |_| || |_ | | >  <  | |    | |   | (_| || | | | | ||  __/ \ V  V / | (_) || |   |   <
\_| \_| \___/  \__,_| \__||_|/_/\_\ \_|    |_|    \__,_||_| |_| |_| \___|  \_/\_/   \___/ |_|   |_|\_\

`)
	fmt.Printf(" Routix API server starting on %s\n", addr)
	
	if DevMode {
		fmt.Printf(" Development endpoints:\n")
		fmt.Printf("  Metrics: http://localhost%s/_dev/metrics\n", addr)
		fmt.Printf("  Routes: http://localhost%s/_dev/routes\n", addr)
		fmt.Printf("  Health: http://localhost%s/_dev/health\n", addr)
	}
	
	return http.ListenAndServe(addr, api.router)
}

// StartTLS starts the server with TLS
func (api *APIBuilder) StartTLS(addr, certFile, keyFile string) error {
	fmt.Printf("🔒 Routix API server starting with TLS on %s\n", addr)
	return http.ListenAndServeTLS(addr, certFile, keyFile, api.router)
}

// Shortcut functions for common patterns

// QuickAPI creates a production-ready API in one line
func QuickAPI() *APIBuilder {
	return NewAPI().
		Prod().
		CORS().
		JSON().
		Health("/health").
		Metrics("/metrics")
}

// DevAPI creates a development API with debugging
func DevAPI() *APIBuilder {
	return NewAPI().
		Dev().
		CORS().
		JSON().
		Health("/health")
}

// MicroAPI creates a minimal API for microservices
func MicroAPI() *APIBuilder {
	return NewAPI().
		JSON().
		Health("/health").
		RateLimit(1000, "1m")
}