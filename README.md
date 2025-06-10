# Routix

A high-performance HTTP router with an Express.js-like API for Go. It features fast routing, middleware support, and a clean, intuitive interface.

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
    // Create a new router
    r := routix.New()

    // Add global middleware
    r.Use(routix.Logger(), routix.Recovery(), routix.CORS())

    // Define routes
    r.GET("/", func(c *routix.Context) error {
        return c.JSON(http.StatusOK, map[string]string{
            "message": "Hello, World!",
        })
    })

    // Start the server
    http.ListenAndServe(":8080", r)
}
```

## Response Helpers

Routix provides several response helpers to make it easy to send different types of responses:

```go
// JSON response
c.JSON(http.StatusOK, map[string]interface{}{
    "name": "John",
    "age":  30,
})

// Success response with standardized format
c.Success(map[string]interface{}{
    "id":   1,
    "name": "John",
})

// Error response with standardized format
c.Error(fmt.Errorf("not found"), "User not found")

// HTML response
c.HTML(http.StatusOK, "<h1>Hello, World!</h1>")

// Plain text response
c.String(http.StatusOK, "Hello, %s!", "World")

// Redirect response
c.Redirect(http.StatusFound, "/new-location")

// Paginated response
c.Paginated([]map[string]interface{}{
    {"id": 1, "name": "John"},
    {"id": 2, "name": "Jane"},
}, 1, 5) // page 1 of 5
```

## Middleware

Routix comes with several built-in middleware functions:

```go
// Logger middleware - logs request information
r.Use(routix.Logger())

// Recovery middleware - recovers from panics
r.Use(routix.Recovery())

// CORS middleware - handles cross-origin requests
r.Use(routix.CORS())

// Rate limiting middleware - limits requests per IP
r.Use(routix.RateLimit(100, time.Minute))

// Timeout middleware - adds timeout to requests
r.Use(routix.Timeout(5 * time.Second))

// Cache middleware - caches responses
r.Use(routix.Cache(1 * time.Hour))

// Compression middleware - compresses responses
r.Use(routix.Compress())
```

## Route Groups

Route groups allow you to organize related routes and apply middleware to multiple routes:

```go
// API group with prefix
api := r.Group("/api")
{
    // /api/users
    api.GET("/users", func(c *routix.Context) error {
        return c.Success([]map[string]interface{}{
            {"id": 1, "name": "John"},
            {"id": 2, "name": "Jane"},
        })
    })

    // /api/users/:id
    api.GET("/users/:id", func(c *routix.Context) error {
        return c.Success(map[string]interface{}{
            "id": c.Params["id"],
        })
    })
}

// Group with middleware
admin := r.Group("/admin")
admin.Use(routix.Auth(func(token string) bool {
    return token == "valid-token"
}))
```

## Error Handling

Routix provides standardized error handling:

```go
// Custom 404 handler
r.NotFound(func(c *routix.Context) error {
    return c.Error(fmt.Errorf("not found"), "Resource not found")
})

// Custom 405 handler
r.MethodNotAllowed(func(c *routix.Context) error {
    return c.Error(fmt.Errorf("method not allowed"), "Method not allowed")
})

// Error response format
{
    "status": "error",
    "data": {
        "message": "Resource not found"
    },
    "timestamp": "2024-03-20T12:00:00Z"
}
```

## Cache Usage

Routix provides built-in cache support:

```go
// Cache a response for 1 hour
r.GET("/about", func(c *routix.Context) error {
    c.Cache(1 * time.Hour)
    return c.HTML(http.StatusOK, "<h1>About Us</h1>")
})

// Cache API responses
api := r.Group("/api")
{
    // Weather API - 30 minutes cache
    api.GET("/weather/:city", func(c *routix.Context) error {
        c.Cache(30 * time.Minute)
        return c.Success(map[string]interface{}{
            "city":    c.Params["city"],
            "temp":    25,
            "weather": "Sunny",
        })
    })
}
```

## Examples

Check out the [examples](examples) directory for more usage examples:

- [Basic Usage](examples/basic/main.go)
- [Middleware Usage](examples/middleware/main.go)
- [Response Types](examples/respond/main.go)
- [Cache Usage](examples/cache/main.go)
- [Task Usage](examples/tasks/main.go)

## License

MIT 
