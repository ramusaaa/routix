package generators

import (
	"path/filepath"
)

func GenerateMiddleware(projectName string, config ProjectConfig) {
	generateCORSMiddleware(projectName)
	generateLoggingMiddleware(projectName)
	
	if config.UseRateLimit {
		generateRateLimitMiddleware(projectName)
	}
	
	if config.UseAuth {
		// Auth middleware is generated in auth.go
	}
}

func generateCORSMiddleware(projectName string) {
	content := `package middleware

import (
	"github.com/ramusaaa/routix"
)

func CORS() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			// Set CORS headers
			c.SetHeader("Access-Control-Allow-Origin", "*")
			c.SetHeader("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.SetHeader("Access-Control-Allow-Headers", "Content-Type, Authorization")
			c.SetHeader("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if c.Request.Method == "OPTIONS" {
				return c.JSON(200, map[string]interface{}{
					"status": "ok",
				})
			}

			return next(c)
		}
	}
}

func CORSWithConfig(origins []string, methods []string, headers []string) routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			origin := c.GetHeader("Origin")
			
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range origins {
				if allowedOrigin == "*" || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				c.SetHeader("Access-Control-Allow-Origin", origin)
			}

			// Set other CORS headers
			if len(methods) > 0 {
				methodsStr := ""
				for i, method := range methods {
					if i > 0 {
						methodsStr += ", "
					}
					methodsStr += method
				}
				c.SetHeader("Access-Control-Allow-Methods", methodsStr)
			}

			if len(headers) > 0 {
				headersStr := ""
				for i, header := range headers {
					if i > 0 {
						headersStr += ", "
					}
					headersStr += header
				}
				c.SetHeader("Access-Control-Allow-Headers", headersStr)
			}

			c.SetHeader("Access-Control-Max-Age", "86400")

			if c.Request.Method == "OPTIONS" {
				return c.JSON(200, map[string]interface{}{
					"status": "ok",
				})
			}

			return next(c)
		}
	}
}`

	writeFile(filepath.Join(projectName, "app", "middleware", "cors.go"), content)
}

func generateLoggingMiddleware(projectName string) {
	content := `package middleware

import (
	"log"
	"time"

	"github.com/ramusaaa/routix"
)

func Logger() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Log request
			duration := time.Since(start)
			method := c.Request.Method
			path := c.Request.URL.Path
			userAgent := c.GetHeader("User-Agent")

			log.Printf("[%s] %s %s - %v - %s",
				method,
				path,
				duration,
				c.Request.RemoteAddr,
				userAgent,
			)

			return err
		}
	}
}

func DetailedLogger() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			start := time.Now()

			// Process request
			err := next(c)

			// Detailed logging
			duration := time.Since(start)
			method := c.Request.Method
			path := c.Request.URL.Path
			query := c.Request.URL.RawQuery
			userAgent := c.GetHeader("User-Agent")
			referer := c.GetHeader("Referer")
			ip := c.Request.RemoteAddr

			logData := map[string]interface{}{
				"method":     method,
				"path":       path,
				"query":      query,
				"duration":   duration.String(),
				"ip":         ip,
				"user_agent": userAgent,
				"referer":    referer,
				"timestamp":  start.Format(time.RFC3339),
			}

			if err != nil {
				logData["error"] = err.Error()
				log.Printf("ERROR: %+v", logData)
			} else {
				log.Printf("INFO: %+v", logData)
			}

			return err
		}
	}
}`

	writeFile(filepath.Join(projectName, "app", "middleware", "logger.go"), content)
}

func generateRateLimitMiddleware(projectName string) {
	content := `package middleware

import (
	"sync"
	"time"

	"github.com/ramusaaa/routix"
)

type RateLimiter struct {
	requests map[string][]time.Time
	mutex    sync.RWMutex
	limit    int
	window   time.Duration
}

func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

func (rl *RateLimiter) Allow(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	
	// Clean old requests
	if requests, exists := rl.requests[key]; exists {
		var validRequests []time.Time
		for _, reqTime := range requests {
			if now.Sub(reqTime) < rl.window {
				validRequests = append(validRequests, reqTime)
			}
		}
		rl.requests[key] = validRequests
	}

	// Check if limit exceeded
	if len(rl.requests[key]) >= rl.limit {
		return false
	}

	// Add current request
	rl.requests[key] = append(rl.requests[key], now)
	return true
}

var globalRateLimiter = NewRateLimiter(100, time.Minute)

func RateLimit(limit int, window time.Duration) routix.Middleware {
	limiter := NewRateLimiter(limit, window)
	
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			// Use IP as key (you might want to use user ID for authenticated requests)
			key := c.Request.RemoteAddr
			
			if !limiter.Allow(key) {
				return c.JSON(429, map[string]interface{}{
					"status":  "error",
					"message": "Rate limit exceeded",
				})
			}

			return next(c)
		}
	}
}

func RateLimitByIP(limit int, window time.Duration) routix.Middleware {
	return RateLimit(limit, window)
}

func RateLimitByUser(limit int, window time.Duration) routix.Middleware {
	limiter := NewRateLimiter(limit, window)
	
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			// Try to get user ID from context (set by auth middleware)
			key := c.Request.RemoteAddr // Fallback to IP
			
			// TODO: Get user ID from JWT token or session
			// if userID := getUserIDFromContext(c); userID != "" {
			//     key = userID
			// }
			
			if !limiter.Allow(key) {
				return c.JSON(429, map[string]interface{}{
					"status":  "error",
					"message": "Rate limit exceeded",
				})
			}

			return next(c)
		}
	}
}`

	writeFile(filepath.Join(projectName, "app", "middleware", "rate_limit.go"), content)
}