package routix_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ramusaaa/routix"
)

func newRequest(method, path, body string) *http.Request {
	var b *strings.Reader
	if body != "" {
		b = strings.NewReader(body)
	} else {
		b = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, b)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

func TestStaticRoute(t *testing.T) {
	r := routix.New()
	r.GET("/hello", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{"msg": "hello"})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/hello", ""))

	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestParamRoute(t *testing.T) {
	r := routix.New()
	r.GET("/users/:id", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{"id": c.Params["id"]})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/users/42", ""))

	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
	var body map[string]any
	json.NewDecoder(w.Body).Decode(&body)
	if body["id"] != "42" {
		t.Fatalf("expected id=42 got %v", body["id"])
	}
}

func TestNotFound(t *testing.T) {
	r := routix.New()
	r.GET("/exists", func(c *routix.Context) error { return nil })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/missing", ""))

	if w.Code != 404 {
		t.Fatalf("expected 404 got %d", w.Code)
	}
}

func TestMethodNotAllowed(t *testing.T) {
	r := routix.New()
	r.GET("/only-get", func(c *routix.Context) error { return nil })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("POST", "/only-get", ""))

	if w.Code != 405 {
		t.Fatalf("expected 405 got %d", w.Code)
	}
}

func TestMiddlewareChain(t *testing.T) {
	r := routix.New()
	order := []string{}

	r.Use(func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			order = append(order, "mw1")
			return next(c)
		}
	})
	r.Use(func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			order = append(order, "mw2")
			return next(c)
		}
	})

	r.GET("/", func(c *routix.Context) error {
		order = append(order, "handler")
		return c.JSON(200, nil)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/", ""))

	if len(order) != 3 || order[0] != "mw1" || order[1] != "mw2" || order[2] != "handler" {
		t.Fatalf("unexpected middleware order: %v", order)
	}
}

func TestGroupMiddleware(t *testing.T) {
	r := routix.New()
	hit := false

	api := r.Group("/api")
	api.Use(func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			hit = true
			return next(c)
		}
	})
	api.GET("/ping", func(c *routix.Context) error {
		return c.JSON(200, nil)
	})

	// Group middleware should run for group routes.
	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/api/ping", ""))
	if !hit {
		t.Fatal("group middleware did not run")
	}

	// Group middleware should NOT run for top-level routes.
	hit = false
	r.GET("/top", func(c *routix.Context) error { return c.JSON(200, nil) })
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, newRequest("GET", "/top", ""))
	if hit {
		t.Fatal("group middleware ran for non-group route")
	}
}

func TestSubGroup(t *testing.T) {
	r := routix.New()
	api := r.Group("/api")
	v1 := api.Group("/v1")
	v1.GET("/users", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{"version": "v1"})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/api/v1/users", ""))
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestContextSetGet(t *testing.T) {
	r := routix.New()
	r.Use(func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			c.Set("user_id", 99)
			return next(c)
		}
	})
	r.GET("/me", func(c *routix.Context) error {
		v, ok := c.Get("user_id")
		if !ok {
			t.Fatal("key not found")
		}
		if v.(int) != 99 {
			t.Fatalf("expected 99 got %v", v)
		}
		return c.JSON(200, nil)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/me", ""))
}

func TestResponseHelpers(t *testing.T) {
	r := routix.New()

	r.POST("/created", func(c *routix.Context) error {
		return c.Created(map[string]any{"id": 1})
	})
	r.DELETE("/nothing", func(c *routix.Context) error {
		return c.NoContent()
	})
	r.GET("/bad", func(c *routix.Context) error {
		return c.BadRequest("invalid input")
	})
	r.GET("/auth", func(c *routix.Context) error {
		return c.Unauthorized("")
	})
	r.GET("/forbidden", func(c *routix.Context) error {
		return c.Forbidden("")
	})
	r.GET("/missing", func(c *routix.Context) error {
		return c.NotFound("")
	})

	cases := []struct {
		method, path string
		want         int
	}{
		{"POST", "/created", 201},
		{"DELETE", "/nothing", 204},
		{"GET", "/bad", 400},
		{"GET", "/auth", 401},
		{"GET", "/forbidden", 403},
		{"GET", "/missing", 404},
	}

	for _, tc := range cases {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRequest(tc.method, tc.path, ""))
		if w.Code != tc.want {
			t.Errorf("%s %s: expected %d got %d", tc.method, tc.path, tc.want, w.Code)
		}
	}
}

func TestCacheHeader(t *testing.T) {
	r := routix.New()
	r.GET("/static", func(c *routix.Context) error {
		c.Cache(24 * time.Hour)
		return c.JSON(200, map[string]any{"ok": true})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/static", ""))

	cc := w.Header().Get("Cache-Control")
	if cc != "public, max-age=86400" {
		t.Fatalf("unexpected Cache-Control: %q", cc)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	r := routix.New()
	r.Use(routix.RateLimit(2, time.Second))
	r.GET("/ping", func(c *routix.Context) error {
		return c.JSON(200, nil)
	})

	for i := 0; i < 2; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, newRequest("GET", "/ping", ""))
		if w.Code != 200 {
			t.Fatalf("request %d: expected 200 got %d", i+1, w.Code)
		}
	}

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/ping", ""))
	if w.Code != 400 {
		t.Fatalf("3rd request past rate limit: expected 400 got %d", w.Code)
	}
}

func TestCORSMiddleware(t *testing.T) {
	r := routix.New()
	r.Use(routix.CORS())
	r.GET("/data", func(c *routix.Context) error {
		return c.JSON(200, nil)
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/data", ""))

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("CORS header missing")
	}
}

func TestParseDuration(t *testing.T) {
	// parseDuration is internal; test it indirectly via RateLimit/Timeout accepting standard formats.
	r := routix.New()
	r.WithRateLimit(100, "30s")
	r.WithTimeout("5s")
	// just confirm no panic; not asserting behavior here
}

func TestRoutes(t *testing.T) {
	r := routix.New()
	r.GET("/a", func(c *routix.Context) error { return nil })
	r.POST("/b", func(c *routix.Context) error { return nil })

	routes := r.Routes()
	if len(routes) != 2 {
		t.Fatalf("expected 2 routes got %d", len(routes))
	}
}

func TestWildcardRoute(t *testing.T) {
	r := routix.New()
	r.GET("/files/*", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{"path": c.Params["*"]})
	})

	w := httptest.NewRecorder()
	r.ServeHTTP(w, newRequest("GET", "/files/docs/api.md", ""))
	if w.Code != 200 {
		t.Fatalf("expected 200 got %d", w.Code)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	r := routix.New()
	r.Use(routix.Recovery())
	r.GET("/panic", func(c *routix.Context) error {
		panic("test panic")
	})

	w := httptest.NewRecorder()
	// should not crash the test process
	r.ServeHTTP(w, newRequest("GET", "/panic", ""))
	if w.Code != 500 {
		t.Fatalf("expected 500 after panic recovery got %d", w.Code)
	}
}
