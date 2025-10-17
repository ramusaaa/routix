package routix

import (
	"encoding/json"
	"fmt"
	"strconv"
)

func (c *Context) ParseJSON(v interface{}) error {
	if c.Request.Header.Get("Content-Type") != "application/json" {
		return fmt.Errorf("content-type must be application/json")
	}
	return json.NewDecoder(c.Request.Body).Decode(v)
}

func (c *Context) Param(name string) string {
	return c.Params[name]
}

func (c *Context) ParamInt(name string) (int, error) {
	return strconv.Atoi(c.Params[name])
}

func (c *Context) ParamInt64(name string) (int64, error) {
	return strconv.ParseInt(c.Params[name], 10, 64)
}

func (c *Context) QueryParam(name string) string {
	return c.Query[name]
}

func (c *Context) QueryParamDefault(name, defaultValue string) string {
	if value, exists := c.Query[name]; exists && value != "" {
		return value
	}
	return defaultValue
}

func (c *Context) QueryParamInt(name string) (int, error) {
	value := c.Query[name]
	if value == "" {
		return 0, fmt.Errorf("parameter %s not found", name)
	}
	return strconv.Atoi(value)
}

func (c *Context) QueryParamIntDefault(name string, defaultValue int) int {
	if value, err := c.QueryParamInt(name); err == nil {
		return value
	}
	return defaultValue
}

func (c *Context) IsJSON() bool {
	return c.Request.Header.Get("Content-Type") == "application/json"
}

func (c *Context) IsAjax() bool {
	return c.Request.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) ClientIP() string {
	if xff := c.Request.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	if xri := c.Request.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	return c.Request.RemoteAddr
}

func (c *Context) UserAgent() string {
	return c.Request.Header.Get("User-Agent")
}