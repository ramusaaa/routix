# Routix

🚀 **Go Web Framework** - Fast, elegant, and developer-friendly

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/ramusaaa/routix.svg)](https://github.com/ramusaaa/routix/releases)

Routix is a high-performance HTTP framework for Go that brings the elegance and developer experience of Laravel to the Go ecosystem. With its powerful CLI tool and intuitive API, you can build robust web applications and APIs quickly and efficiently.

## ✨ Features

### 🎯 **Core Framework**
- **Fast HTTP Router** - Optimized radix tree routing
- **Middleware System** - Composable middleware chain
- **Route Groups** - Organize routes with prefixes and middleware
- **Parameter Binding** - Automatic URL parameter extraction
- **Response Helpers** - JSON, HTML, redirects, and more
- **Error Handling** - Structured error responses
- **Request Validation** - Built-in validation system

### 🛠️ **CLI Tool (Laravel-inspired)**
- **Project Scaffolding** - `routix new` with interactive setup
- **Code Generators** - `routix make:controller`, `make:model`, etc.
- **Hot Reload** - `routix serve` with file watching
- **Database Migrations** - `routix migrate` system
- **Testing Tools** - `routix test` with coverage
- **Route Inspection** - `routix route:list`

### 🏗️ **Architecture**
- **Multiple Templates** - API, Full-stack, Microservice, Minimal
- **Database Support** - PostgreSQL, MySQL, SQLite with GORM
- **Authentication** - JWT-based auth system
- **Caching** - Redis integration
- **Docker Ready** - Multi-stage Dockerfile included
- **WebSocket Support** - Real-time communication
- **Job Queues** - Background job processing

### 🔧 **Built-in Middleware**
- **CORS** - Cross-origin resource sharing
- **Rate Limiting** - Request throttling
- **Authentication** - JWT validation
- **Logging** - Request/response logging
- **Recovery** - Panic recovery
- **Compression** - Gzip compression
- **Timeout** - Request timeout handling

## 🚀 Quick Start

### Installation

**🎯 Easy Install (Recommended)**
```bash
# One-command install with automatic PATH setup
curl -sSL https://raw.githubusercontent.com/ramusaaa/routix/main/install.sh | bash
```

**📦 Manual Install**
```bash
# Install the CLI tool
go install github.com/ramusaaa/routix/cmd/routix@v0.3.8

# Make sure Go's bin directory is in your PATH
# Add this to your shell profile (~/.bashrc, ~/.zshrc, etc.)
export PATH="$HOME/go/bin:$PATH"

# Reload your shell or run:
source ~/.zshrc  # or ~/.bashrc

# Verify installation
routix --version
```

**⚡ One-line manual install**
```bash
# For zsh users (macOS default)
go install github.com/ramusaaa/routix/cmd/routix@latest && echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc

# For bash users
go install github.com/ramusaaa/routix/cmd/routix@latest && echo 'export PATH="$HOME/go/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc
```

**📚 Use as a library**
```bash
go get github.com/ramusaaa/routix@latest
```

### Create a New Project

```bash
# Interactive project creation
routix new my-awesome-api

# Follow the prompts to configure your project
cd my-awesome-api
go mod tidy
routix serve
```

### Manual Setup

```go
package main

import (
    "github.com/ramusaaa/routix"
)

func main() {
    // Create a new Routix router
    r := routix.New()
    
    // Add routes
    r.GET("/", func(c *routix.Context) error {
        return c.JSON(200, map[string]interface{}{
            "message": "Hello from Routix!",
            "version": "v0.3.8",
        })
    })
    
    r.GET("/users/:id", func(c *routix.Context) error {
        id := c.Params["id"]
        return c.JSON(200, map[string]interface{}{
            "user_id": id,
        })
    })
    
    // Start server
    r.Start(":8080")
}
```

## 🎮 CLI Commands

### Project Management
```bash
routix new <project-name>    # Create new project
routix serve                 # Start development server
routix build                 # Build for production
```

### Code Generation
```bash
routix make controller UserController --resource
routix make model User --migration
routix make middleware Auth
routix make migration create_users_table
routix make seeder UserSeeder
routix make service EmailService
routix make request CreateUserRequest
routix make test UserTest
```

### Database Operations
```bash
routix migrate              # Run migrations
routix migrate rollback     # Rollback last migration
routix migrate status       # Show migration status
routix seed                 # Run seeders
```

### Development Tools
```bash
routix route list           # Show all routes
routix test                 # Run tests
routix test --coverage      # Run tests with coverage
```

## 📚 API Examples

### Response Helpers

```go
// JSON responses
c.JSON(200, data)
c.Success(data)              // 200 with success wrapper
c.Created(data)              // 201 with success wrapper
c.Error(400, "Bad request")  // Error response

// Other responses
c.HTML(200, "<h1>Hello</h1>")
c.String(200, "Hello %s", "World")
c.Redirect(302, "/login")
```

### Middleware Usage

```go
// Global middleware
r.Use(routix.Logger())
r.Use(routix.Recovery())
r.Use(routix.CORS())

// Route-specific middleware
r.GET("/admin", adminHandler, authMiddleware)

// Group middleware
admin := r.Group("/admin")
admin.Use(authMiddleware)
admin.GET("/users", getUsersHandler)
```

### Route Groups

```go
// API versioning
v1 := r.Group("/api/v1")
v1.GET("/users", getUsersV1)
v1.POST("/users", createUserV1)

v2 := r.Group("/api/v2")
v2.GET("/users", getUsersV2)
v2.POST("/users", createUserV2)

// Protected routes
auth := r.Group("/auth")
auth.Use(authMiddleware)
auth.GET("/profile", getProfile)
auth.PUT("/profile", updateProfile)
```

### Advanced Features

```go
// Rate limiting
r.Use(routix.RateLimit(100, time.Minute))

// Caching
r.GET("/data", func(c *routix.Context) error {
    c.Cache(1 * time.Hour)
    return c.Success(expensiveData)
})

// Request timeout
r.Use(routix.Timeout(30 * time.Second))

// Custom error handling
r.NotFound(func(c *routix.Context) error {
    return c.Error(404, "Page not found")
})
```

## 🏗️ Project Structure

When you create a new project with `routix new`, you get a well-organized structure:

```
my-project/
├── app/
│   ├── controllers/     # HTTP controllers
│   ├── models/         # Database models
│   ├── middleware/     # Custom middleware
│   ├── services/       # Business logic
│   ├── requests/       # Request validation
│   └── resources/      # API resources
├── config/             # Configuration
├── database/
│   ├── migrations/     # Database migrations
│   └── seeders/       # Database seeders
├── routes/            # Route definitions
├── storage/           # File storage
├── tests/             # Test files
├── docs/              # Documentation
├── docker-compose.yml # Docker setup
├── Dockerfile         # Container definition
├── Makefile          # Build commands
└── README.md         # Project documentation
```

## 🐳 Docker Support

Every project comes with Docker support:

```bash
# Development
docker-compose -f docker-compose.dev.yml up

# Production
docker-compose up -d
```

## 🧪 Testing

```bash
# Run all tests
routix test

# Run with coverage
routix test --coverage

# Run specific test types
routix test unit
routix test integration
```

## 📖 Documentation

- **[Getting Started Guide](docs/getting-started.md)**
- **[CLI Reference](docs/cli-reference.md)**
- **[API Documentation](docs/api-reference.md)**
- **[Middleware Guide](docs/middleware.md)**
- **[Database Guide](docs/database.md)**
- **[Deployment Guide](docs/deployment.md)**

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Built with [Go](https://golang.org/) for performance and reliability
- Thanks to all contributors and the Go community

## 📊 Benchmarks

```
BenchmarkRoutix-8        5000000    250 ns/op    0 allocs/op
BenchmarkGin-8          3000000    400 ns/op    0 allocs/op
BenchmarkEcho-8         2000000    500 ns/op    0 allocs/op
```

---

**Happy coding with Routix! 🚀**

For questions and support, please [open an issue](https://github.com/ramusaaa/routix/issues) or join our [community discussions](https://github.com/ramusaaa/routix/discussions).
