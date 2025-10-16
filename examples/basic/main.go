package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ramusaaa/routix"
)

func main() {
	r := routix.New()

	r.Use(routix.Logger(), routix.Recovery(), routix.CORS())

	r.GET("/", func(c *routix.Context) error {
		return c.Success("Hello, World!")
	})

	r.GET("/users/:id", func(c *routix.Context) error {
		return c.Success(map[string]interface{}{
			"id":   c.Params["id"],
			"name": "John Doe",
		})
	})

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

	api := r.Group("/api")
	{
		api.GET("/v1/users", func(c *routix.Context) error {
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "John"},
				{"id": 2, "name": "Jane"},
			})
		})
	}

	r.NotFound(func(c *routix.Context) error {
		return c.Error(fmt.Errorf("not found"), "Resource not found")
	})

	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}