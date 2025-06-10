// Package routix provides utility functions for common HTTP operations.
// It includes helpers for request validation, response formatting, and error handling.
package routix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ValidateStruct validates a struct using its tags.
// It returns a slice of validation errors if any validation fails.
func ValidateStruct(v interface{}) []error {
	var errors []error
	val := reflect.ValueOf(v)

	// Handle pointers
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Must be a struct
	if val.Kind() != reflect.Struct {
		return []error{fmt.Errorf("value must be a struct")}
	}

	// Iterate through fields
	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		value := val.Field(i)

		// Get validation tags
		tag := field.Tag.Get("validate")
		if tag == "" {
			continue
		}

		// Parse validation rules
		rules := strings.Split(tag, ",")
		for _, rule := range rules {
			parts := strings.Split(rule, "=")
			ruleName := parts[0]
			var ruleValue string
			if len(parts) > 1 {
				ruleValue = parts[1]
			}

			// Apply validation rules
			switch ruleName {
			case "required":
				if value.IsZero() {
					errors = append(errors, fmt.Errorf("%s is required", field.Name))
				}
			case "min":
				min, _ := strconv.Atoi(ruleValue)
				switch value.Kind() {
				case reflect.String:
					if len(value.String()) < min {
						errors = append(errors, fmt.Errorf("%s must be at least %d characters", field.Name, min))
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if value.Int() < int64(min) {
						errors = append(errors, fmt.Errorf("%s must be at least %d", field.Name, min))
					}
				}
			case "max":
				max, _ := strconv.Atoi(ruleValue)
				switch value.Kind() {
				case reflect.String:
					if len(value.String()) > max {
						errors = append(errors, fmt.Errorf("%s must be at most %d characters", field.Name, max))
					}
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					if value.Int() > int64(max) {
						errors = append(errors, fmt.Errorf("%s must be at most %d", field.Name, max))
					}
				}
			case "email":
				if value.Kind() == reflect.String {
					if !strings.Contains(value.String(), "@") {
						errors = append(errors, fmt.Errorf("%s must be a valid email address", field.Name))
					}
				}
			}
		}
	}

	return errors
}

// ParseJSON parses a JSON request body into a struct.
// It returns an error if the body cannot be parsed or validated.
func ParseJSON(r *http.Request, v interface{}) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if errors := ValidateStruct(v); len(errors) > 0 {
		return fmt.Errorf("validation failed: %v", errors)
	}

	return nil
}

// Must panics if the given error is not nil.
// It is used to handle critical errors that should stop program execution.
func Must(err error) {
	if err != nil {
		panic(err)
	}
}

// MustValidate panics if struct validation fails.
// It is used to ensure required fields are present and valid.
func MustValidate(v interface{}) {
	if errors := ValidateStruct(v); len(errors) > 0 {
		panic(fmt.Errorf("validation failed: %v", errors))
	}
}

// GetQueryParam gets a query parameter with a default value.
// It returns the parameter value if present, otherwise the default value.
func GetQueryParam(r *http.Request, key, defaultValue string) string {
	if value := r.URL.Query().Get(key); value != "" {
		return value
	}
	return defaultValue
}

// GetHeader gets a header value with a default value.
// It returns the header value if present, otherwise the default value.
func GetHeader(r *http.Request, key, defaultValue string) string {
	if value := r.Header.Get(key); value != "" {
		return value
	}
	return defaultValue
}

// GetCookie gets a cookie value with a default value.
// It returns the cookie value if present, otherwise the default value.
func GetCookie(r *http.Request, name, defaultValue string) string {
	if cookie, err := r.Cookie(name); err == nil {
		return cookie.Value
	}
	return defaultValue
}

// ParseJSON parses the request body as JSON into the given struct
func (c *Context) ParseJSON(v interface{}) error {
	if c.Request.Body == nil {
		return fmt.Errorf("request body is empty")
	}

	decoder := json.NewDecoder(c.Request.Body)
	if err := decoder.Decode(v); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	return nil
}

// Task represents an asynchronous task
type Task struct {
	ID        string
	Status    string
	Progress  float64
	Result    interface{}
	Error     error
	CreatedAt time.Time
	UpdatedAt time.Time
}

// TaskManager manages asynchronous tasks
type TaskManager struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewTaskManager creates a new task manager
func NewTaskManager() *TaskManager {
	return &TaskManager{
		tasks: make(map[string]*Task),
	}
}

// CreateTask creates a new task
func (tm *TaskManager) CreateTask(id string) *Task {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	task := &Task{
		ID:        id,
		Status:    "pending",
		Progress:  0,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	tm.tasks[id] = task
	return task
}

// GetTask gets a task by ID
func (tm *TaskManager) GetTask(id string) (*Task, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	task, ok := tm.tasks[id]
	return task, ok
}

// UpdateTask updates a task's status and progress
func (tm *TaskManager) UpdateTask(id string, status string, progress float64) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Status = status
		task.Progress = progress
		task.UpdatedAt = time.Now()
	}
}

// CompleteTask marks a task as completed
func (tm *TaskManager) CompleteTask(id string, result interface{}) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Status = "completed"
		task.Progress = 100
		task.Result = result
		task.UpdatedAt = time.Now()
	}
}

// FailTask marks a task as failed
func (tm *TaskManager) FailTask(id string, err error) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if task, ok := tm.tasks[id]; ok {
		task.Status = "failed"
		task.Error = err
		task.UpdatedAt = time.Now()
	}
}
