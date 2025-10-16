# Routix

High-performance HTTP router for Go with Express.js-like API.

## Features

- Fast routing with radix tree
- Middleware support  
- Route grouping
- Parameterized routes
- Built-in response helpers
- Cache support
- Error handling
- CORS support
- Rate limiting
- Request validation

## Installation

```bash
go get -u github.com/ramusaaa/routix
```

## Quick Start

```go
package main

import (
    "net/http"
    "github.com/ramusaaa/routix"
)

func main() {
    r := routix.New()
    
    r.GET("/", func(c *routix.Context) error {
        return c.Success("Hello World")
    })
    
    r.GET("/users/:id", func(c *routix.Context) error {
        id, _ := c.ParamInt("id")
        return c.JSON(200, map[string]int{"id": id})
    })
    
    r.Start(":8080")
}
```

## Response Helpers

```go
c.JSON(200, data)
c.Success(data)
c.Error(err, "message")
c.HTML(200, "<h1>Hello</h1>")
c.String(200, "Hello %s", "World")
c.Redirect(302, "/new-location")
c.Paginated(data, 1, 5)
```

## Middleware

```go
r.Use(routix.Logger())
r.Use(routix.Recovery())
r.Use(routix.CORS())
r.Use(routix.RateLimit(100, time.Minute))
r.Use(routix.Timeout(5 * time.Second))
r.Use(routix.Cache(1 * time.Hour))
```

## Route Groups

```go
api := r.Group("/api")
api.GET("/users", getUsers)
api.GET("/users/:id", getUser)

admin := r.Group("/admin")
admin.Use(routix.Auth(validateToken))
```

## Error Handling

```go
r.NotFound(notFoundHandler)
r.MethodNotAllowed(methodNotAllowedHandler)
```

## Cache

```go
r.GET("/data", func(c *routix.Context) error {
    c.Cache(1 * time.Hour)
    return c.Success(data)
})
```

## License

MIT 
