package routix

import (
	"fmt"
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

	fmt.Println()
	fmt.Println("\033[32m" + `
  ██████╗  ██████╗ ██╗   ██╗████████╗██╗██╗  ██╗
  ██╔══██╗██╔═══██╗██║   ██║╚══██╔══╝██║╚██╗██╔╝
  ██████╔╝██║   ██║██║   ██║   ██║   ██║ ╚███╔╝ 
  ██╔══██╗██║   ██║██║   ██║   ██║   ██║ ██╔██╗ 
  ██║  ██║╚██████╔╝╚██████╔╝   ██║   ██║██╔╝ ██╗
  ╚═╝  ╚═╝ ╚═════╝  ╚═════╝    ╚═╝   ╚═╝╚═╝  ╚═╝` + "\033[0m")
	fmt.Println()
	fmt.Println("  \033[1mRoutix Framework v0.3.8\033[0m")
	fmt.Println("  \033[90mPowered by Ramusa Software Corporation\033[0m")
	fmt.Println()
	fmt.Println("  \033[36m➜\033[0m  \033[1mLocal:\033[0m   \033[36mhttp://localhost" + addr + "/\033[0m")

	if DevMode {
		fmt.Println()
		fmt.Println("  \033[33m⚡ Development Mode\033[0m")
		fmt.Printf("     Metrics: http://localhost%s/_dev/metrics\n", addr)
		fmt.Printf("     Routes:  http://localhost%s/_dev/routes\n", addr)
		fmt.Printf("     Health:  http://localhost%s/_dev/health\n", addr)
	}

	fmt.Println()
	fmt.Println("  \033[90mpress \033[1mh\033[0m\033[90m to show help\033[0m")
	fmt.Println()

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
