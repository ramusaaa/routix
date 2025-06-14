// Package routix provides error handling utilities for HTTP responses.
// It includes standardized error types and helper functions for error management.
package routix

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// ValidationError represents a validation error with a field name and message.
// It is used to provide detailed information about validation failures.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Error implements the error interface for ValidationError.
// It returns a formatted error message including the field name.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []*ValidationError

func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var msg string
	for i, err := range e {
		if i > 0 {
			msg += "; "
		}
		msg += err.Error()
	}
	return msg
}

// NewValidationError creates a new ValidationError with the given field and message.
// It is used to create standardized validation error responses.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// IsValidationError checks if the given error is a ValidationError.
// It is used to determine if an error should be handled as a validation error.
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsValidationErrors checks if an error is a ValidationErrors
func IsValidationErrors(err error) bool {
	_, ok := err.(ValidationErrors)
	return ok
}

// GetHTTPStatusCode returns the appropriate HTTP status code for the given error.
// It handles both ValidationError and Error types, defaulting to 500 for unknown errors.
func GetHTTPStatusCode(err error) int {
	if e, ok := err.(*Error); ok {
		return e.Code
	}
	if _, ok := err.(*ValidationError); ok {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

// WrapError wraps an error with additional context.
// It is used to add more information to existing errors.
func WrapError(err error, message string) error {
	return fmt.Errorf("%s: %w", message, err)
}

// UnwrapError unwraps an error to get its underlying cause.
// It is used to extract the original error from wrapped errors.
func UnwrapError(err error) error {
	type unwrapper interface {
		Unwrap() error
	}
	if u, ok := err.(unwrapper); ok {
		return u.Unwrap()
	}
	return err
}

// Error represents a Routix error with additional context
type Error struct {
	Code    int    // HTTP status code
	Message string // User-friendly error message
	Err     error  // Original error
	Stack   string // Stack trace
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Unwrap returns the wrapped error
func (e *Error) Unwrap() error {
	return e.Err
}

// NewError creates a new Routix error with stack trace
func NewError(code int, message string, err error) *Error {
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(3, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stack strings.Builder
	for {
		frame, more := frames.Next()
		fmt.Fprintf(&stack, "%s\n\t%s:%d\n", frame.Function, frame.File, frame.Line)
		if !more {
			break
		}
	}

	return &Error{
		Code:    code,
		Message: message,
		Err:     err,
		Stack:   stack.String(),
	}
}

// Common error constructors
func BadRequest(message string, err error) *Error {
	return NewError(400, message, err)
}

func Unauthorized(message string, err error) *Error {
	return NewError(401, message, err)
}

func Forbidden(message string, err error) *Error {
	return NewError(403, message, err)
}

func NotFound(message string, err error) *Error {
	return NewError(404, message, err)
}

func MethodNotAllowed(message string, err error) *Error {
	return NewError(405, message, err)
}

func InternalServerError(message string, err error) *Error {
	return NewError(500, message, err)
}

// ErrorResponse represents the structure of error responses
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
	Stack   string `json:"stack,omitempty"`
}

// ToResponse converts an Error to an ErrorResponse
func (e *Error) ToResponse() ErrorResponse {
	resp := ErrorResponse{
		Code:    e.Code,
		Message: e.Message,
	}
	if e.Err != nil {
		resp.Error = e.Err.Error()
	}
	if e.Stack != "" {
		resp.Stack = e.Stack
	}
	return resp
}
