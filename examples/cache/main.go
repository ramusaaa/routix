package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/ramusaaa/routix"
)

func main() {
	// Create a new router
	r := routix.New()

	// Add global middleware
	r.Use(routix.Logger(), routix.Recovery(), routix.CORS())

	// Cache for static content
	r.GET("/about", func(c *routix.Context) error {
		// 24 hours cache
		c.Cache(24 * time.Hour)
		return c.HTML(http.StatusOK, "<h1>About Us</h1>")
	})

	// Cache for blog posts
	r.GET("/blog/:slug", func(c *routix.Context) error {
		// 1 hour cache
		c.Cache(1 * time.Hour)
		return c.Success(map[string]interface{}{
			"title":   "Blog Post",
			"content": "Content...",
			"slug":    c.Params["slug"],
		})
	})

	// Cache for API group
	api := r.Group("/api")
	{
		// Weather API - 30 minutes cache
		api.GET("/weather/:city", func(c *routix.Context) error {
			c.Cache(30 * time.Minute)
			return c.Success(map[string]interface{}{
				"city":    c.Params["city"],
				"temp":    25,
				"weather": "Sunny",
			})
		})

		// Exchange rates - 15 minutes cache
		api.GET("/exchange-rates", func(c *routix.Context) error {
			c.Cache(15 * time.Minute)
			return c.Success(map[string]interface{}{
				"USD": 28.5,
				"EUR": 31.2,
				"GBP": 36.1,
			})
		})
	}

	// Cache for e-commerce group
	shop := r.Group("/shop")
	{
		// Product list - 1 hour cache
		shop.GET("/products", func(c *routix.Context) error {
			c.Cache(1 * time.Hour)
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "Product 1", "price": 100},
				{"id": 2, "name": "Product 2", "price": 200},
			})
		})

		// Category list - 1 hour cache
		shop.GET("/categories", func(c *routix.Context) error {
			c.Cache(1 * time.Hour)
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "Category 1"},
				{"id": 2, "name": "Category 2"},
			})
		})

		// Popular products - 30 minutes cache
		shop.GET("/popular", func(c *routix.Context) error {
			c.Cache(30 * time.Minute)
			return c.Success([]map[string]interface{}{
				{"id": 1, "name": "Popular Product 1", "price": 150},
				{"id": 2, "name": "Popular Product 2", "price": 250},
			})
		})
	}

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
