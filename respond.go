package routix

import (
	"encoding/json"
	"fmt"
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
		Data:      map[string]string{"message": err.Error()},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (c *Context) JSON(status int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(status)
	encoder := json.NewEncoder(c.Response)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(data)
}

func (c *Context) FastJSON(status int, data interface{}) error {
	c.Response.Header().Set("Content-Type", "application/json; charset=utf-8")
	c.Response.WriteHeader(status)
	
	switch v := data.(type) {
	case string:
		_, err := c.Response.Write([]byte(`"` + v + `"`))
		return err
	case int:
		_, err := fmt.Fprintf(c.Response, "%d", v)
		return err
	case bool:
		if v {
			_, err := c.Response.Write([]byte("true"))
			return err
		}
		_, err := c.Response.Write([]byte("false"))
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
