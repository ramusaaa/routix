package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ramusaaa/routix"
)

func main() {
	// Create a new router
	r := routix.New()

	// Add global middleware
	r.Use(routix.Logger(), routix.Recovery(), routix.CORS())

	// Simple route
	r.GET("/hello", func(c *routix.Context) error {
		return c.Success("Hello, World!")
	})

	// Route with parameters
	r.GET("/users/:id", func(c *routix.Context) error {
		return c.Success(map[string]interface{}{
			"id":   c.Params["id"],
			"name": "Ramusa Yilmaz",
		})
	})

	// Route with query parameters
	r.GET("/search", func(c *routix.Context) error {
		query := c.Query["q"]
		return c.Success(map[string]interface{}{
			"query": query,
			"results": []string{
				"Result 1",
				"Result 2",
			},
		})
	})

	// Route with JSON body
	r.POST("/users", func(c *routix.Context) error {
		var user struct {
			Name  string `json:"name" validate:"required"`
			Email string `json:"email" validate:"required,email"`
		}

		if err := c.ParseJSON(&user); err != nil {
			return c.Error(err, "Invalid request body")
		}

		return c.Success(map[string]interface{}{
			"message": "User created",
			"user":    user,
		})
	})

	// Route group
	api := r.Group("/api")
	{
		// /api/v1/users
		api.GET("/v1/users", func(c *routix.Context) error {
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "Ramusa"},
				{"id": 2, "name": "Yilmaz"},
			})
		})

		// /api/v1/users/:id
		api.GET("/v1/users/:id", func(c *routix.Context) error {
			return c.Success(map[string]interface{}{
				"id":   c.Params["id"],
				"name": "Ramusa",
			})
		})
	}

	// Custom error handlers
	r.NotFound(func(c *routix.Context) error {
		return c.Error(fmt.Errorf("not found"), "Resource not found")
	})

	r.MethodNotAllowed(func(c *routix.Context) error {
		return c.Error(fmt.Errorf("method not allowed"), "Method not allowed")
	})

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
