<<<<<<< HEAD
# routix
Simple Easy Router with Golang
=======
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

## Examples

Check out the [examples](examples) directory for more usage examples:

- [Basic Usage](examples/basic/main.go)
- [Middleware Usage](examples/middleware/main.go)
- [Cache Usage](examples/cache/main.go)

## License

MIT 
>>>>>>> 5072e9a (Initial commit)
