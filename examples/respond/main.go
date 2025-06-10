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

	// JSON response example
	r.GET("/json", func(c *routix.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"name": "John",
			"age":  30,
			"city": "New York",
		})
	})

	// Success response example
	r.GET("/success", func(c *routix.Context) error {
		return c.Success(map[string]interface{}{
			"id":       1,
			"name":     "John",
			"email":    "john@example.com",
			"isActive": true,
		})
	})

	// Error response example
	r.GET("/error", func(c *routix.Context) error {
		return c.Error(fmt.Errorf("user not found"), "User with ID 123 not found")
	})

	// HTML response example
	r.GET("/html", func(c *routix.Context) error {
		return c.HTML(http.StatusOK, `
			<!DOCTYPE html>
			<html>
				<head>
					<title>Hello World</title>
				</head>
				<body>
					<h1>Hello, World!</h1>
					<p>This is an HTML response from Routix.</p>
				</body>
			</html>
		`)
	})

	// String response example
	r.GET("/string", func(c *routix.Context) error {
		return c.String(http.StatusOK, "Hello, %s! The time is %s",
			"World",
			time.Now().Format("15:04:05"),
		)
	})

	// Redirect response example
	r.GET("/redirect", func(c *routix.Context) error {
		return c.Redirect(http.StatusFound, "/success")
	})

	// Paginated response example
	r.GET("/paginated", func(c *routix.Context) error {
		// Simulate paginated data
		items := []map[string]interface{}{
			{"id": 1, "name": "Item 1"},
			{"id": 2, "name": "Item 2"},
			{"id": 3, "name": "Item 3"},
			{"id": 4, "name": "Item 4"},
			{"id": 5, "name": "Item 5"},
		}
		return c.Paginated(items, 1, 2) // page 1 of 2
	})

	// File response example
	r.GET("/file", func(c *routix.Context) error {
		c.SetHeader("Content-Type", "text/plain")
		c.SetHeader("Content-Disposition", "attachment; filename=example.txt")
		return c.String(http.StatusOK, "This is a file download example")
	})

	// Custom headers example
	r.GET("/headers", func(c *routix.Context) error {
		c.SetHeader("X-Custom-Header", "custom-value")
		c.SetHeader("X-Request-ID", "12345")
		return c.Success(map[string]string{
			"message": "Check the response headers",
		})
	})

	// Cookie example
	r.GET("/cookie", func(c *routix.Context) error {
		cookie := &http.Cookie{
			Name:     "session",
			Value:    "abc123",
			Path:     "/",
			Expires:  time.Now().Add(24 * time.Hour),
			HttpOnly: true,
		}
		c.SetCookie(cookie)
		return c.Success(map[string]string{
			"message": "Cookie set successfully",
		})
	})

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
