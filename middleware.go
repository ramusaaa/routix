// Package routix provides a collection of middleware functions for HTTP request processing.
// The middleware package includes common middleware for logging, authentication,
// rate limiting, caching, and more.
//
// Features:
// - Request logging with timing
// - Error handling and recovery
// - CORS support
// - Authentication middleware
// - Rate limiting
// - Request timeout
// - Response caching
// - Response compression
// - Request validation
//
// Example usage:
//
//	router := routix.New()
//	router.Use(
//	    routix.Logger(),
//	    routix.Recovery(),
//	    routix.CORS(),
//	    routix.RateLimit(100, time.Minute),
//	)
//
//	// Protected routes
//	auth := router.Group("/api")
//	auth.Use(routix.Auth(validateToken))
//	auth.GET("/users", getUsers)
//
//	// Routes with validation
//	router.POST("/users", routix.Validate(&User{}), createUser)
package routix

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"math"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Logger returns a middleware that logs request method, path, status code, and latency.
func Logger() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			start := time.Now()
			path := c.Request.URL.Path
			method := c.Request.Method

			err := next(c)

			status := c.Status()
			duration := time.Since(start)

			color := "\033[32m" // green
			if status >= 500 {
				color = "\033[31m" // red
			} else if status >= 400 {
				color = "\033[33m" // yellow
			}
			fmt.Printf("%s%d\033[0m  %-7s %s  %v\n", color, status, method, path, duration)

			return err
		}
	}
}

// ErrorHandler is a middleware that handles errors in a consistent way
func ErrorHandler() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error { writeHandlerError(c, next(c)); return nil }
	}
}

func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (err error) {
			defer func() {
				if recovered := recover(); recovered != nil {
					if recovered == http.ErrAbortHandler {
						panic(recovered)
					}
					writeHandlerError(c, InternalServerError("Internal Server Error", nil))
					err = nil
				}
			}()
			return next(c)
		}
	}
}

// CORS returns a middleware that handles Cross-Origin Resource Sharing.
// It sets appropriate CORS headers for cross-origin requests.
func CORS() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			c.Response.Header().Set("Access-Control-Allow-Origin", "*")
			c.Response.Header().Set("Access-Control-Allow-Methods", "GET, HEAD, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				c.Response.WriteHeader(http.StatusOK)
				return nil
			}

			return next(c)
		}
	}
}

// Auth returns a middleware that handles authentication.
// It checks for a valid Authorization header and validates the token.
func Auth(validateToken func(string) bool) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			parts := strings.Fields(c.GetHeader("Authorization"))
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || validateToken == nil || !validateToken(parts[1]) {
				c.SetHeader("WWW-Authenticate", "Bearer")
				return c.Unauthorized("Invalid authentication token")
			}
			return next(c)
		}
	}
}

// RateLimit returns a middleware that implements rate limiting.
// It limits the number of requests from a single IP address.
func RateLimit(requests int, duration time.Duration) Middleware {
	if requests <= 0 || duration <= 0 {
		panic("routix: rate limit and duration must be positive")
	}
	type entry struct{ timestamps []time.Time }
	var mu sync.Mutex
	limiter := make(map[string]*entry)
	nextCleanup := time.Now().Add(duration)
	return func(next Handler) Handler {
		return func(c *Context) error {
			ip := c.Request.RemoteAddr
			if host, _, err := net.SplitHostPort(ip); err == nil {
				ip = host
			}
			now := time.Now()
			mu.Lock()
			if !now.Before(nextCleanup) {
				for key, e := range limiter {
					if len(e.timestamps) == 0 || now.Sub(e.timestamps[len(e.timestamps)-1]) >= duration {
						delete(limiter, key)
					}
				}
				nextCleanup = now.Add(duration)
			}
			e := limiter[ip]
			if e == nil {
				e = &entry{}
				limiter[ip] = e
			}
			valid := e.timestamps[:0]
			for _, timestamp := range e.timestamps {
				if now.Sub(timestamp) < duration {
					valid = append(valid, timestamp)
				}
			}
			e.timestamps = valid
			if len(valid) >= requests {
				retry := int(math.Ceil(duration.Seconds() - now.Sub(valid[0]).Seconds()))
				mu.Unlock()
				c.SetHeader("Retry-After", strconv.Itoa(retry))
				return c.JSON(http.StatusTooManyRequests, map[string]any{"status": "error", "message": "Too many requests"})
			}
			e.timestamps = append(e.timestamps, now)
			mu.Unlock()
			return next(c)
		}
	}
}

// Timeout returns a middleware that adds a timeout to request processing.
// It cancels the request if it takes longer than the specified duration.
func Timeout(timeout time.Duration) Middleware {
	if timeout <= 0 {
		panic("routix: timeout must be positive")
	}
	return func(next Handler) Handler {
		return func(c *Context) error {
			local := *c
			local.Params = cloneMap(c.Params)
			local.Query = cloneMap(c.Query)
			local.Body = cloneMap(c.Body)
			local.values = cloneMap(c.values)
			completed := make(chan struct{}, 1)
			handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				local.Request = req
				local.Writer = &responseWriter{ResponseWriter: w}
				local.Response = local.Writer
				writeHandlerError(&local, next(&local))
				completed <- struct{}{}
			})
			c.SetHeader("Content-Type", "application/json; charset=utf-8")
			http.TimeoutHandler(handler, timeout, `{"status":"error","message":"Request timed out"}`).ServeHTTP(c.Response, c.Request.Clone(c.Request.Context()))
			select {
			case <-completed:
				c.values = local.values
			default:
			}
			return nil
		}
	}
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Validate validates the request body or query parameters against a struct
func Validate(v interface{}) Middleware {
	typ := reflect.TypeOf(v)
	if typ == nil || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		panic("routix: validation requires a pointer to a struct")
	}
	return func(next Handler) Handler {
		return func(c *Context) error {
			value := reflect.New(typ.Elem()).Interface()
			if err := c.ParseJSON(value); err != nil {
				return c.BadRequest("Validation failed")
			}
			validator := NewValidator()
			if !validator.Validate(value) {
				return c.BadRequest(fmt.Sprintf("validation failed: %v", validator.Errors()))
			}
			c.Set("validated", value)
			return next(c)
		}
	}
}

// Cache caches responses for a specified duration
func Cache(duration time.Duration) Middleware {
	type cachedResponse struct {
		body    []byte
		headers http.Header
		code    int
		expires time.Time
	}
	cache := make(map[string]cachedResponse)
	var nextCleanup time.Time
	var mu sync.Mutex
	return func(next Handler) Handler {
		return func(c *Context) error {
			if duration <= 0 || c.Request.Method != http.MethodGet || c.GetHeader("Authorization") != "" || c.GetHeader("Cookie") != "" || c.GetHeader("Range") != "" || c.GetHeader("Cache-Control") != "" || c.GetHeader("If-None-Match") != "" || c.GetHeader("If-Modified-Since") != "" {
				return next(c)
			}
			key := c.Request.Host + "\n" + c.Request.URL.RequestURI() + "\n" + c.GetHeader("Accept-Encoding") + "\n" + c.GetHeader("Origin")
			mu.Lock()
			now := time.Now()
			if !now.Before(nextCleanup) {
				for key, item := range cache {
					if !now.Before(item.expires) {
						delete(cache, key)
					}
				}
				nextCleanup = now.Add(duration)
			}
			item, ok := cache[key]
			if ok && !now.Before(item.expires) {
				delete(cache, key)
				ok = false
			}
			mu.Unlock()
			if ok {
				for k, v := range item.headers {
					c.Response.Header()[k] = append([]string(nil), v...)
				}
				c.Response.WriteHeader(item.code)
				_, err := c.Response.Write(item.body)
				return err
			}
			originalWriter, originalResponse := c.Writer, c.Response
			capture := &cacheWriter{ResponseWriter: originalResponse}
			c.Writer = &responseWriter{ResponseWriter: capture}
			c.Response = c.Writer
			defer func() { c.Writer = originalWriter; c.Response = originalResponse }()
			err := next(c)
			ttl := cacheTTL(capture.Header().Get("Cache-Control"), duration)
			if err == nil && c.Status() == http.StatusOK && !capture.tooLarge && !capture.flushed && len(capture.Header().Values("Set-Cookie")) == 0 && capture.Header().Get("Vary") == "" && capture.Header().Get("Expires") == "" && ttl > 0 {
				item = cachedResponse{body: bytes.Clone(capture.body.Bytes()), headers: capture.Header().Clone(), code: c.Status(), expires: time.Now().Add(ttl)}
				mu.Lock()
				if len(cache) >= 1024 {
					for old := range cache {
						delete(cache, old)
						break
					}
				}
				cache[key] = item
				mu.Unlock()
			}
			return err
		}
	}
}

func cacheTTL(control string, duration time.Duration) time.Duration {
	for _, directive := range strings.Split(strings.ToLower(control), ",") {
		name, value, hasValue := strings.Cut(strings.TrimSpace(directive), "=")
		switch strings.TrimSpace(name) {
		case "private", "no-store", "no-cache":
			return 0
		case "max-age", "s-maxage":
			seconds, err := strconv.ParseInt(strings.Trim(strings.TrimSpace(value), "\""), 10, 64)
			if !hasValue || err != nil || seconds <= 0 {
				return 0
			}
			if seconds <= int64(duration/time.Second) {
				duration = time.Duration(seconds) * time.Second
			}
		}
	}
	return duration
}

type cacheWriter struct {
	http.ResponseWriter
	body     bytes.Buffer
	tooLarge bool
	flushed  bool
}

func (w *cacheWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *cacheWriter) Write(b []byte) (int, error) {
	n, err := w.ResponseWriter.Write(b)
	if !w.tooLarge {
		if w.body.Len()+n > 1<<20 {
			w.tooLarge = true
			w.body.Reset()
		} else {
			_, _ = w.body.Write(b[:n])
		}
	}
	return n, err
}
func (w *cacheWriter) Flush() {
	w.flushed = true
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func Compress() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			addVary(c.Response.Header(), "Accept-Encoding")
			if c.Request.Method == http.MethodHead || c.GetHeader("Range") != "" || !acceptsGzip(c.GetHeader("Accept-Encoding")) {
				return next(c)
			}
			originalWriter, originalResponse := c.Writer, c.Response
			writer := &gzipResponseWriter{ResponseWriter: originalResponse}
			c.Writer = &responseWriter{ResponseWriter: writer}
			c.Response = c.Writer
			defer func() { c.Writer = originalWriter; c.Response = originalResponse }()
			err := next(c)
			if writer.gzip != nil {
				if closeErr := writer.gzip.Close(); err == nil {
					err = closeErr
				}
			}
			return err
		}
	}
}

func addVary(header http.Header, value string) {
	for _, line := range header.Values("Vary") {
		for _, part := range strings.Split(line, ",") {
			if strings.EqualFold(strings.TrimSpace(part), value) || strings.TrimSpace(part) == "*" {
				return
			}
		}
	}
	if current := header.Get("Vary"); current != "" {
		header.Set("Vary", strings.Join(header.Values("Vary"), ", ")+", "+value)
	} else {
		header.Set("Vary", value)
	}
}

func acceptsGzip(header string) bool {
	wildcard := false
	for _, entry := range strings.Split(header, ",") {
		parts := strings.Split(entry, ";")
		coding := strings.TrimSpace(parts[0])
		quality := 1.0
		for _, part := range parts[1:] {
			name, value, ok := strings.Cut(strings.TrimSpace(part), "=")
			if ok && strings.EqualFold(name, "q") {
				q, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
				if err != nil || q < 0 || q > 1 || math.IsNaN(q) {
					quality = 0
				} else {
					quality = q
				}
			}
		}
		if strings.EqualFold(coding, "gzip") {
			return quality > 0
		}
		if coding == "*" {
			wildcard = quality > 0
		}
	}
	return wildcard
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gzip    *gzip.Writer
	written bool
}

func (w *gzipResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *gzipResponseWriter) WriteHeader(code int) {
	if code >= 100 && code < 200 && code != http.StatusSwitchingProtocols {
		w.ResponseWriter.WriteHeader(code)
		return
	}
	if w.written {
		return
	}
	w.written = true
	if code >= 200 && code != http.StatusNoContent && code != http.StatusNotModified && code != http.StatusResetContent && code != http.StatusPartialContent && w.Header().Get("Content-Encoding") == "" && !strings.Contains(strings.ToLower(w.Header().Get("Cache-Control")), "no-transform") && !strings.HasPrefix(w.Header().Get("Content-Type"), "text/event-stream") {
		w.Header().Del("Content-Length")
		w.Header().Set("Content-Encoding", "gzip")
		w.gzip = gzip.NewWriter(w.ResponseWriter)
	}
	w.ResponseWriter.WriteHeader(code)
}
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		if w.Header().Get("Content-Type") == "" {
			w.Header().Set("Content-Type", http.DetectContentType(b))
		}
		w.WriteHeader(http.StatusOK)
	}
	if w.gzip != nil {
		return w.gzip.Write(b)
	}
	return w.ResponseWriter.Write(b)
}
func (w *gzipResponseWriter) Flush() {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	if w.gzip != nil {
		_ = w.gzip.Flush()
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}
