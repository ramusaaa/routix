package routix

import (
	"net/http"
	"time"
)

var DevMode = false

type APIBuilder struct {
	router *Router
}

func NewAPI() *APIBuilder {
	router := New()

	router.Use(PerformanceMonitor())

	return &APIBuilder{
		router: router,
	}
}

func (api *APIBuilder) Dev() *APIBuilder {
	DevMode = true
	return api
}

func (api *APIBuilder) Prod() *APIBuilder {
	api.router.Use(
		Recovery(),
		Logger(),
		Compress(),
	)
	return api
}

func (api *APIBuilder) CORS() *APIBuilder {
	api.router.Use(CORS())
	return api
}

func (api *APIBuilder) Auth(validator func(string) bool) *APIBuilder {
	api.router.Use(Auth(validator))
	return api
}

func (api *APIBuilder) RateLimit(requests int, window string) *APIBuilder {
	duration := parseDuration(window)
	api.router.Use(RateLimit(requests, duration))
	return api
}

func (api *APIBuilder) Cache(duration string) *APIBuilder {
	d := parseDuration(duration)
	api.router.Use(Cache(d))
	return api
}

func (api *APIBuilder) Timeout(duration string) *APIBuilder {
	d := parseDuration(duration)
	api.router.Use(Timeout(d))
	return api
}

func (api *APIBuilder) JSON() *APIBuilder {
	api.router.Use(func(next Handler) Handler {
		return func(c *Context) error {
			c.SetHeader("Content-Type", "application/json; charset=utf-8")
			return next(c)
		}
	})
	return api
}

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

func (api *APIBuilder) Metrics(path string) *APIBuilder {
	if path == "" {
		path = "/metrics"
	}

	api.router.GET(path, func(c *Context) error {
		return c.JSON(http.StatusOK, globalMetrics.GetMetrics())
	})
	return api
}

func (api *APIBuilder) Static(path, dir string) *APIBuilder {
	api.router.Static(path, dir)
	return api
}

func (api *APIBuilder) Group(path string, fn func(*Group)) *APIBuilder {
	group := api.router.Group(path)
	fn(group)
	return api
}

func (api *APIBuilder) GroupRouter(path string) *Group {
	return api.router.Group(path)
}
func (api *APIBuilder) V1(fn func(*Group)) *APIBuilder {
	return api.Group("/api/v1", fn)
}

func (api *APIBuilder) V2(fn func(*Group)) *APIBuilder {
	return api.Group("/api/v2", fn)
}

func (api *APIBuilder) V3(fn func(*Group)) *APIBuilder {
	return api.Group("/api/v3", fn)
}

func (api *APIBuilder) CRUD(path string, controller ResourceController) *APIBuilder {
	api.router.Resource(path, controller)
	return api
}

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

func (api *APIBuilder) Build() *Router {
	return api.router
}

func (api *APIBuilder) Start(addr string) error {
	if len(api.router.trees) == 0 {
		api.router.GET("/", WelcomeHandler("Routix"))
	}

	printBanner(addr, DevMode)

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return listenAndServe(srv)
}

// StartTLS starts the server with TLS support.
func (api *APIBuilder) StartTLS(addr, certFile, keyFile string) error {
	if len(api.router.trees) == 0 {
		api.router.GET("/", WelcomeHandler("Routix"))
	}

	printBanner(addr, DevMode)

	srv := &http.Server{
		Addr:         addr,
		Handler:      api.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return srv.ListenAndServeTLS(certFile, keyFile)
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
