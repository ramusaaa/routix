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

	// Create a task manager
	taskManager := routix.NewTaskManager()

	// Add global middleware
	r.Use(routix.Logger(), routix.Recovery(), routix.CORS())

	// Start a long-running task
	r.POST("/tasks", func(c *routix.Context) error {
		// Create a new task
		taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
		task := taskManager.CreateTask(taskID)

		// Start the task in a goroutine
		go func() {
			// Simulate work
			for i := 0; i <= 100; i += 10 {
				time.Sleep(500 * time.Millisecond)
				taskManager.UpdateTask(taskID, "processing", float64(i))
			}

			// Complete the task
			taskManager.CompleteTask(taskID, map[string]interface{}{
				"message": "Task completed successfully",
			})
		}()

		return c.Success(map[string]interface{}{
			"task_id": taskID,
			"status":  task.Status,
		})
	})

	// Get task status
	r.GET("/tasks/:id", func(c *routix.Context) error {
		taskID := c.Params["id"]
		task, ok := taskManager.GetTask(taskID)
		if !ok {
			return c.Error(fmt.Errorf("task not found"), "Task not found")
		}

		return c.Success(map[string]interface{}{
			"id":       task.ID,
			"status":   task.Status,
			"progress": task.Progress,
			"result":   task.Result,
			"error":    task.Error,
			"created":  task.CreatedAt,
			"updated":  task.UpdatedAt,
		})
	})

	// Start the server
	fmt.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
