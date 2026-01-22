package main

import (
	"log"

	"github.com/ramusaaa/routix"
)

// Task example demonstrating background task management
func main() {
	// Create a new API using the builder
	app := routix.NewAPI().
		Prod().
		JSON().
		CORS()

	// Health check endpoint
	app.GET("/health", func(c *routix.Context) error {
		return c.Success(map[string]any{
			"status": "healthy",
		})
	})

	// Example: Start a task (demonstrates async pattern)
	app.POST("/tasks", func(c *routix.Context) error {
		// In a real application, you would create a task and return its ID
		// Task management would be handled by your application logic
		return c.Success(map[string]any{
			"task_id": "example-task-123",
			"status":  "pending",
			"message": "Task created - implement your own task management logic",
		})
	})

	// Example: Get task status
	app.GET("/tasks/:id", func(c *routix.Context) error {
		taskID := c.Params["id"]
		return c.Success(map[string]any{
			"id":      taskID,
			"status":  "completed",
			"message": "This is an example response - implement your own task lookup",
		})
	})

	// Start the server
	log.Fatal(app.Start(":8080"))
}
