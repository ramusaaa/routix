// Package routix provides error handling utilities for HTTP responses.
// It includes standardized error types and helper functions for error management.
package routix

import (
	"fmt"
	"net/http"
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

// HTTPError represents an HTTP error with a status code and message.
// It is used to standardize HTTP error responses.
type HTTPError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
}

// Error implements the error interface for HTTPError.
// It returns the error message.
func (e *HTTPError) Error() string {
	return e.Message
}

// NewValidationError creates a new ValidationError with the given field and message.
// It is used to create standardized validation error responses.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewHTTPError creates a new HTTPError with the given status code and message.
// It is used to create standardized HTTP error responses.
func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
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

// IsHTTPError checks if the given error is an HTTPError.
// It is used to determine if an error should be handled as an HTTP error.
func IsHTTPError(err error) bool {
	_, ok := err.(*HTTPError)
	return ok
}

// GetHTTPStatusCode returns the appropriate HTTP status code for the given error.
// It handles both ValidationError and HTTPError types, defaulting to 500 for unknown errors.
func GetHTTPStatusCode(err error) int {
	if httpErr, ok := err.(*HTTPError); ok {
		return httpErr.StatusCode
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
