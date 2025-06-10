// Package routix provides standardized response handling for HTTP requests.
// It includes support for success, error, and paginated responses with consistent formatting.
package routix

import (
	"encoding/json"
	"time"
)

// ResponseStatus represents the status of a response.
// It can be either "success" or "error".
type ResponseStatus string

const (
	// StatusSuccess indicates a successful response
	StatusSuccess ResponseStatus = "success"
	// StatusError indicates an error response
	StatusError ResponseStatus = "error"
)

// BaseResponse represents the base structure for all responses.
// It includes the response status and timestamp.
type BaseResponse struct {
	Status    ResponseStatus `json:"status"`
	Timestamp string         `json:"timestamp"`
}

// SuccessResponse represents a successful response with generic data.
// The data type is specified using Go's generic type parameter T.
type SuccessResponse[T any] struct {
	BaseResponse
	Data T `json:"data"`
}

// ErrorResponse represents an error response with a message.
// It follows a consistent error format for all error responses.
type ErrorResponse struct {
	BaseResponse
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

// PaginatedResponse represents a paginated response with generic data.
// It includes page information and total page count.
type PaginatedResponse[T any] struct {
	BaseResponse
	Data struct {
		Page       T   `json:"page"`
		PageNumber int `json:"pageNumber"`
		TotalPages int `json:"totalPages"`
	} `json:"data"`
}

// RespondError represents a custom error type for standardized error responses.
// It implements the error interface and provides formatted error messages.
type RespondError struct {
	Status    ResponseStatus `json:"status"`
	Data      interface{}    `json:"data"`
	Timestamp string         `json:"timestamp"`
}

// Error implements the error interface for RespondError.
// It extracts and returns the error message from the response data.
func (e *RespondError) Error() string {
	if data, ok := e.Data.(map[string]interface{}); ok {
		if msg, ok := data["message"].(string); ok {
			return msg
		}
	}
	return "An unexpected error occurred"
}

// Respond creates a response with the given status and data.
// It handles both success and error cases, formatting the response appropriately.
func Respond[T any](status ResponseStatus, data T) (interface{}, error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	if status == StatusError {
		var message string
		switch v := any(data).(type) {
		case string:
			message = v
		case error:
			message = v.Error()
		default:
			message = "An unexpected error occurred"
		}

		return nil, &RespondError{
			Status:    StatusError,
			Data:      map[string]string{"message": message},
			Timestamp: timestamp,
		}
	}

	// Handle string data for success response
	if str, ok := any(data).(string); ok {
		return str, nil
	}

	return SuccessResponse[T]{
		BaseResponse: BaseResponse{
			Status:    StatusSuccess,
			Timestamp: timestamp,
		},
		Data: data,
	}, nil
}

// RespondPaginated creates a paginated response with the given data and page information.
// It formats the response according to the pagination structure.
func RespondPaginated[T any](data T, pageNumber, totalPages int) SuccessResponse[struct {
	Page       T   `json:"page"`
	PageNumber int `json:"pageNumber"`
	TotalPages int `json:"totalPages"`
}] {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	return SuccessResponse[struct {
		Page       T   `json:"page"`
		PageNumber int `json:"pageNumber"`
		TotalPages int `json:"totalPages"`
	}]{
		BaseResponse: BaseResponse{
			Status:    StatusSuccess,
			Timestamp: timestamp,
		},
		Data: struct {
			Page       T   `json:"page"`
			PageNumber int `json:"pageNumber"`
			TotalPages int `json:"totalPages"`
		}{
			Page:       data,
			PageNumber: pageNumber,
			TotalPages: totalPages,
		},
	}
}

// ConvertError converts any error to a RespondError.
// It uses the provided fallback message if the error cannot be converted.
func ConvertError(err error, fallbackMessage string) error {
	if fallbackMessage == "" {
		fallbackMessage = "An unexpected error occurred"
	}

	if respondErr, ok := err.(*RespondError); ok {
		return respondErr
	}

	return &RespondError{
		Status:    StatusError,
		Data:      map[string]string{"message": err.Error()},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

// JSON sends a JSON response with the given status and data.
// It sets the appropriate content type and encodes the data as JSON.
func (c *Context) JSON(status int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json")
	c.Response.WriteHeader(status)
	return json.NewEncoder(c.Response).Encode(data)
}

// Success sends a success response with the given data.
// It automatically formats the response according to the success structure.
func (c *Context) Success(data interface{}) error {
	response, err := Respond(StatusSuccess, data)
	if err != nil {
		return err
	}
	return c.JSON(200, response)
}

// Error sends an error response with the given error and fallback message.
// It converts the error to a standardized error response format.
func (c *Context) Error(err error, fallbackMessage string) error {
	convertedErr := ConvertError(err, fallbackMessage)
	return c.JSON(400, convertedErr)
}

// Paginated sends a paginated response with the given data and page information.
// It formats the response according to the pagination structure.
func (c *Context) Paginated(data interface{}, pageNumber, totalPages int) error {
	response := RespondPaginated(data, pageNumber, totalPages)
	return c.JSON(200, response)
}
