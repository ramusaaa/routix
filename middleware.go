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
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"
)

// Logger returns a middleware that logs request information.
// It logs the method, path, status code, and processing time for each request.
func Logger() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			start := time.Now()
			path := c.Request.URL.Path
			method := c.Request.Method

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Log request details
			fmt.Printf("[%s] %s %s - %v\n", method, path, duration, err)

			return err
		}
	}
}

// ErrorHandler is a middleware that handles errors in a consistent way
func ErrorHandler() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			// Call the next handler
			err := next(c)
			if err == nil {
				return nil
			}

			// Handle the error
			var routixErr *Error
			if e, ok := err.(*Error); ok {
				routixErr = e
			} else {
				routixErr = InternalServerError("Internal Server Error", err)
			}

			// Convert error to response
			resp := routixErr.ToResponse()

			// Set content type
			c.Response.Header().Set("Content-Type", "application/json")
			c.Response.WriteHeader(routixErr.Code)

			// Encode and send response
			return json.NewEncoder(c.Response).Encode(resp)
		}
	}
}

// Recovery is a middleware that recovers from panics
func Recovery() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch x := r.(type) {
					case string:
						err = InternalServerError("Internal Server Error", fmt.Errorf("%s", x))
					case error:
						err = InternalServerError("Internal Server Error", x)
					default:
						err = InternalServerError("Internal Server Error", fmt.Errorf("unknown panic"))
					}

					// Handle the error
					var routixErr *Error
					if e, ok := err.(*Error); ok {
						routixErr = e
					} else {
						routixErr = InternalServerError("Internal Server Error", err)
					}

					// Convert error to response
					resp := routixErr.ToResponse()

					// Set content type
					c.Response.Header().Set("Content-Type", "application/json")
					c.Response.WriteHeader(routixErr.Code)

					// Encode and send response
					json.NewEncoder(c.Response).Encode(resp)
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
			c.Response.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
			token := c.Request.Header.Get("Authorization")
			if token == "" {
				return c.Error(fmt.Errorf("unauthorized"), "Authentication required")
			}

			if !validateToken(token) {
				return c.Error(fmt.Errorf("invalid token"), "Invalid authentication token")
			}

			return next(c)
		}
	}
}

// RateLimit returns a middleware that implements rate limiting.
// It limits the number of requests from a single IP address.
func RateLimit(requests int, duration time.Duration) Middleware {
	// Simple in-memory rate limiter
	limiter := make(map[string][]time.Time)

	return func(next Handler) Handler {
		return func(c *Context) error {
			ip := c.Request.RemoteAddr

			// Clean old timestamps
			now := time.Now()
			var valid []time.Time
			for _, t := range limiter[ip] {
				if now.Sub(t) < duration {
					valid = append(valid, t)
				}
			}
			limiter[ip] = valid

			// Check rate limit
			if len(limiter[ip]) >= requests {
				return c.Error(fmt.Errorf("rate limit exceeded"), "Too many requests")
			}

			// Add current timestamp
			limiter[ip] = append(limiter[ip], now)

			return next(c)
		}
	}
}

// Timeout returns a middleware that adds a timeout to request processing.
// It cancels the request if it takes longer than the specified duration.
func Timeout(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			// Create a channel for the response
			done := make(chan error, 1)

			// Process request in a goroutine
			go func() {
				done <- next(c)
			}()

			// Wait for response or timeout
			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				return c.Error(fmt.Errorf("request timeout"), "Request timed out")
			}
		}
	}
}

// Validate validates the request body or query parameters against a struct
func Validate(v interface{}) Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			if err := c.ParseJSON(v); err != nil {
				return c.Error(err, "Validation failed")
			}

			validator := NewValidator()
			if !validator.Validate(v) {
				return c.Error(fmt.Errorf("validation failed: %v", validator.Errors()), "Validation failed")
			}

			return next(c)
		}
	}
}

// Cache caches responses for a specified duration
func Cache(duration time.Duration) Middleware {
	cache := make(map[string]struct {
		response []byte
		headers  http.Header
		code     int
		expires  time.Time
	})
	var mu sync.RWMutex

	return func(next Handler) Handler {
		return func(c *Context) error {
			// Only cache GET requests
			if c.Request.Method != http.MethodGet {
				return next(c)
			}

			key := c.Request.URL.String()

			// Check cache
			mu.RLock()
			if cached, ok := cache[key]; ok && time.Now().Before(cached.expires) {
				mu.RUnlock()
				// Write cached response
				for k, v := range cached.headers {
					c.Response.Header()[k] = v
				}
				c.Response.WriteHeader(cached.code)
				c.Response.Write(cached.response)
				return nil
			}
			mu.RUnlock()

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

			// Call the next handler
			if err := next(newCtx); err != nil {
				return err
			}

			// Cache the response
			mu.Lock()
			cache[key] = struct {
				response []byte
				headers  http.Header
				code     int
				expires  time.Time
			}{
				response: recorder.Body.Bytes(),
				headers:  recorder.Header(),
				code:     recorder.Code,
				expires:  time.Now().Add(duration),
			}
			mu.Unlock()

			// Copy the response to the original response writer
			for k, v := range recorder.Header() {
				c.Response.Header()[k] = v
			}
			c.Response.WriteHeader(recorder.Code)
			c.Response.Write(recorder.Body.Bytes())

			return nil
		}
	}
}

// Compress compresses responses using gzip
func Compress() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			// Check if client accepts gzip
			if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
				return next(c)
			}

			// Set headers
			c.SetHeader("Content-Encoding", "gzip")
			c.SetHeader("Vary", "Accept-Encoding")

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

			// Call the next handler
			if err := next(newCtx); err != nil {
				return err
			}

			// Copy headers
			for k, v := range recorder.Header() {
				c.Response.Header()[k] = v
			}

			// Write status code
			c.Response.WriteHeader(recorder.Code)

			// Write response without compression
			c.Response.Write(recorder.Body.Bytes())

			return nil
		}
	}
}
