package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ramusaaa/routix"
)

func RequestID() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			requestID := fmt.Sprintf("%d", time.Now().UnixNano())
			c.SetHeader("X-Request-ID", requestID)
			return next(c)
		}
	}
}

func APIKey(key string) routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			apiKey := c.GetHeader("X-API-Key")
			if apiKey != key {
				return c.Error(fmt.Errorf("invalid api key"), "Invalid API key")
			}
			return next(c)
		}
	}
}

func main() {
	// Create a new router
	r := routix.New()

	// Add global middleware
	r.Use(
		routix.Logger(),
		routix.Recovery(),
		routix.CORS(),
		RequestID(),
	)

	// Public routes
	r.GET("/public", func(c *routix.Context) error {
		return c.Success(map[string]interface{}{
			"message": "This is a public endpoint",
		})
	})

	// Protected routes with API key
	api := r.Group("/api")
	api.Use(APIKey("secret-key"))
	{
		api.GET("/users", func(c *routix.Context) error {
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "Ramusa"},
				{"id": 2, "name": "Yilmaz"},
			})
		})

		api.GET("/users/:id", func(c *routix.Context) error {
			return c.Success(map[string]interface{}{
				"id":   c.Params["id"],
				"name": "Ramusa",
			})
		})
	}

	// Rate limited routes
	limited := r.Group("/limited")
	limited.Use(routix.RateLimit(10, time.Minute))
	{
		limited.GET("/data", func(c *routix.Context) error {
			return c.Success(map[string]interface{}{
				"message": "This endpoint is rate limited",
			})
		})
	}

	// Timeout routes
	timeout := r.Group("/timeout")
	timeout.Use(routix.Timeout(5 * time.Second))
	{
		timeout.GET("/slow", func(c *routix.Context) error {
			// Simulate a slow operation
			time.Sleep(6 * time.Second)
			return c.Success(map[string]interface{}{
				"message": "This should timeout",
			})
		})
	}

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
