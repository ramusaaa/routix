package routix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (c *Context) readJSONBody(limit int64) error {
	if c.bodyRead {
		return c.bodyErr
	}
	c.bodyRead = true
	if c.Request.Body == nil {
		c.bodyErr = io.EOF
		return c.bodyErr
	}
	if limit <= 0 {
		limit = 1 << 20
	}
	original := c.Request.Body
	c.rawBody, c.bodyErr = io.ReadAll(http.MaxBytesReader(c.Response, original, limit))
	c.Request.Body = struct {
		io.Reader
		io.Closer
	}{bytes.NewReader(c.rawBody), original}
	if c.bodyErr != nil {
		return c.bodyErr
	}
	if !json.Valid(c.rawBody) {
		c.bodyErr = fmt.Errorf("invalid JSON body")
		return c.bodyErr
	}
	if len(bytes.TrimSpace(c.rawBody)) > 0 && bytes.TrimSpace(c.rawBody)[0] == '{' {
		c.bodyErr = json.Unmarshal(c.rawBody, &c.Body)
	}
	return c.bodyErr
}

func (c *Context) ParseJSON(v interface{}) error {
	if !c.IsJSON() {
		return fmt.Errorf("content-type must be application/json")
	}
	if err := c.readJSONBody(c.bodyLimit); err != nil {
		return err
	}
	return json.Unmarshal(c.rawBody, v)
}

// Cache sets the Cache-Control header so browsers and proxies cache this response.
func (c *Context) Cache(duration time.Duration) {
	c.Response.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", int(duration.Seconds())))
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
	ct, _, err := mime.ParseMediaType(c.Request.Header.Get("Content-Type"))
	return err == nil && (ct == "application/json" || strings.HasPrefix(ct, "application/") && strings.HasSuffix(ct, "+json"))
}

func (c *Context) IsAjax() bool {
	return c.Request.Header.Get("X-Requested-With") == "XMLHttpRequest"
}

func (c *Context) ClientIP() string {
	if c.clientIP != "" {
		return c.clientIP
	}
	return remoteIP(c.Request.RemoteAddr)
}

func (c *Context) UserAgent() string {
	return c.Request.Header.Get("User-Agent")
}
