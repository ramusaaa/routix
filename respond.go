package routix

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ResponseStatus string

const (
	StatusSuccess ResponseStatus = "success"
	StatusError   ResponseStatus = "error"
)

type BaseResponse struct {
	Status    ResponseStatus `json:"status"`
	Timestamp string         `json:"timestamp"`
}

type SuccessResponse[T any] struct {
	BaseResponse
	Data T `json:"data"`
}

type RespondError struct {
	Status    ResponseStatus `json:"status"`
	Data      interface{}    `json:"data"`
	Timestamp string         `json:"timestamp"`
}

func (e *RespondError) Error() string {
	if data, ok := e.Data.(map[string]interface{}); ok {
		if msg, ok := data["message"].(string); ok {
			return msg
		}
	}
	return "An unexpected error occurred"
}

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

func ConvertError(err error, fallbackMessage string) error {
	if fallbackMessage == "" {
		fallbackMessage = "An unexpected error occurred"
	}

	if respondErr, ok := err.(*RespondError); ok {
		return respondErr
	}

	return &RespondError{
		Status:    StatusError,
		Data:      map[string]string{"message": fallbackMessage},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

type jsonResponseWriter struct {
	response http.ResponseWriter
	status   int
}

func (w jsonResponseWriter) Write(body []byte) (int, error) {
	w.response.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.response.WriteHeader(w.status)
	return w.response.Write(body)
}

func (c *Context) JSON(status int, data interface{}) error {
	previous := c.jsonWriter
	c.jsonWriter = jsonResponseWriter{response: c.Response, status: status}
	defer func() { c.jsonWriter = previous }()
	encoder := json.NewEncoder(&c.jsonWriter)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}

func (c *Context) FastJSON(status int, data interface{}) error {
	writer := jsonResponseWriter{response: c.Response, status: status}
	switch value := data.(type) {
	case string:
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		_, err = writer.Write(encoded)
		return err
	case int:
		_, err := fmt.Fprintf(writer, "%d", value)
		return err
	case bool:
		body := []byte("false")
		if value {
			body = []byte("true")
		}
		_, err := writer.Write(body)
		return err
	default:
		return c.JSON(status, data)
	}
}

func (c *Context) Success(data interface{}) error {
	response, err := Respond(StatusSuccess, data)
	if err != nil {
		return err
	}
	return c.JSON(200, response)
}

func (c *Context) Error(err error, fallbackMessage string) error {
	convertedErr := ConvertError(err, fallbackMessage)
	return c.JSON(400, convertedErr)
}

func (c *Context) Paginated(data interface{}, pageNumber, totalPages int) error {
	response := RespondPaginated(data, pageNumber, totalPages)
	return c.JSON(200, response)
}

func (c *Context) Created(data interface{}) error {
	return c.JSON(http.StatusCreated, map[string]any{"status": "success", "data": data})
}

func (c *Context) Accepted(data interface{}) error {
	return c.JSON(http.StatusAccepted, map[string]any{"status": "success", "data": data})
}

func (c *Context) NoContent() error {
	c.Response.WriteHeader(http.StatusNoContent)
	return nil
}

func (c *Context) BadRequest(message string) error {
	return c.JSON(http.StatusBadRequest, map[string]any{"status": "error", "message": message})
}

func (c *Context) Unauthorized(message string) error {
	if message == "" {
		message = "unauthorized"
	}
	return c.JSON(http.StatusUnauthorized, map[string]any{"status": "error", "message": message})
}

func (c *Context) Forbidden(message string) error {
	if message == "" {
		message = "forbidden"
	}
	return c.JSON(http.StatusForbidden, map[string]any{"status": "error", "message": message})
}

func (c *Context) NotFound(message string) error {
	if message == "" {
		message = "not found"
	}
	return c.JSON(http.StatusNotFound, map[string]any{"status": "error", "message": message})
}
