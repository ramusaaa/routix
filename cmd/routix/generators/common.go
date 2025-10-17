package generators

import (
	"fmt"
	"path/filepath"
)

func GenerateCommonFiles(projectName string, config ProjectConfig) {
	generateGitignore(projectName)
	generateReadme(projectName, config)
	generateMakefile(projectName, config)
	generateStorageKeepFiles(projectName)
	
	if config.UseSwagger {
		GenerateSwaggerFiles(projectName, config)
	}
	
	if config.UseTests {
		GenerateTestFiles(projectName, config)
	}
	
	// Generate routes
	GenerateRoutes(projectName, config)
}

func generateGitignore(projectName string) {
	content := `# Binaries for programs and plugins
*.exe
*.exe~
*.dll
*.so
*.dylib

# Test binary, built with 'go test -c'
*.test

# Output of the go coverage tool, specifically when used with LiteIDE
*.out

# Dependency directories
vendor/

# Go workspace file
go.work

# Environment variables
.env
.env.local
.env.development
.env.testing
.env.production

# Database
*.db
*.sqlite
*.sqlite3

# Logs
*.log
logs/
storage/logs/*
!storage/logs/.gitkeep

# Cache
storage/cache/*
!storage/cache/.gitkeep

# Temporary files
tmp/
temp/

# IDE
.vscode/
.idea/
*.swp
*.swo
*~

# OS generated files
.DS_Store
.DS_Store?
._*
.Spotlight-V100
.Trashes
ehthumbs.db
Thumbs.db

# Coverage reports
coverage.out
coverage.html

# Build artifacts
dist/
build/

# Node modules (if using frontend tools)
node_modules/

# Docker
.dockerignore

# Air (hot reload tool)
.air.toml
tmp/`

	writeFile(filepath.Join(projectName, ".gitignore"), content)
}

func generateReadme(projectName string, config ProjectConfig) {
	content := fmt.Sprintf(`# %s

A modern Go web application built with [Routix](https://github.com/ramusaaa/routix) - Go Web Framework.

## Features

- ⚡ **Fast Performance** - Built with Go for maximum speed
- 🛠️ **Developer Friendly** - CLI and structure
- 📦 **Modular Architecture** - Clean separation of concerns
- 🔐 **Authentication Ready** - JWT-based auth system`, projectName)

	if config.UseDatabase {
		content += `
- 🗄️ **Database Integration** - GORM with migrations and seeders`
	}

	if config.UseCache {
		content += `
- ⚡ **Caching** - Redis integration for better performance`
	}

	if config.UseDocker {
		content += `
- 🐳 **Docker Ready** - Complete containerization setup`
	}

	if config.UseSwagger {
		content += `
- 📚 **API Documentation** - Auto-generated Swagger docs`
	}

	content += `

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Git`

	if config.UseDatabase && config.DatabaseType == "postgres" {
		content += `
- PostgreSQL (or use Docker)`
	} else if config.UseDatabase && config.DatabaseType == "mysql" {
		content += `
- MySQL (or use Docker)`
	}

	if config.UseCache {
		content += `
- Redis (or use Docker)`
	}

	content += `

### Installation

1. **Clone the repository**
   ` + "```" + `bash
   git clone <your-repo-url> ` + projectName + `
   cd ` + projectName + `
   ` + "```" + `

2. **Install dependencies**
   ` + "```" + `bash
   go mod tidy
   ` + "```" + `

3. **Setup environment**
   ` + "```" + `bash
   cp .env.example .env
   # Edit .env with your configuration
   ` + "```" + `

4. **Start development server**
   ` + "```" + `bash
   routix serve
   ` + "```" + `

Your application will be available at: **http://localhost:8080**`

	if config.UseSwagger {
		content += `

📚 **API Documentation**: http://localhost:8080/docs`
	}

	content += `

## Development

### CLI Commands

` + "```" + `bash
# Create new components
routix make:controller UserController --resource
routix make:model User --migration
routix make:middleware Auth
routix make:service EmailService

# Database operations
routix migrate
routix migrate:rollback
routix seed

# Development
routix serve          # Start dev server with hot reload
routix test           # Run tests
routix route:list     # Show all routes

# Build for production
routix build
` + "```" + `

### Project Structure

` + "```" + `
` + projectName + `/
├── app/
│   ├── controllers/     # HTTP controllers
│   ├── models/         # Database models
│   ├── middleware/     # HTTP middleware
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
└── docs/              # Documentation
` + "```" + `

### API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET    | /        | Welcome page |
| GET    | /health  | Health check |`

	if config.UseAuth {
		content += `
| POST   | /auth/register | User registration |
| POST   | /auth/login    | User login |
| POST   | /auth/logout   | User logout |
| GET    | /auth/me       | Get current user |`
	}

	content += `
| GET    | /api/v1/status | API status |

### Testing

` + "```" + `bash
# Run all tests
routix test

# Run specific test types
routix test unit
routix test integration

# Run with coverage
routix test --coverage
` + "```" + `

## Deployment

### Build for Production

` + "```" + `bash
routix build
./app
` + "```" + `

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Built With

- [Routix](https://github.com/ramusaaa/routix) - Go Web Framework
- [GORM](https://gorm.io/) - Go ORM library`

	if config.UseAuth {
		content += `
- [JWT-Go](https://github.com/golang-jwt/jwt) - JWT implementation for Go`
	}

	if config.UseCache {
		content += `
- [go-redis](https://github.com/redis/go-redis) - Redis client for Go`
	}

	content += `

---

**Happy coding! 🚀**`

	writeFile(filepath.Join(projectName, "README.md"), content)
}

func generateMakefile(projectName string, config ProjectConfig) {
	content := `# Makefile for ` + projectName + `

.PHONY: help build run test clean dev migrate seed docker

# Default target
help:
	@echo "Available commands:"
	@echo "  build     - Build the application"
	@echo "  run       - Run the application"
	@echo "  dev       - Start development server"
	@echo "  test      - Run tests"
	@echo "  clean     - Clean build artifacts"
	@echo "  migrate   - Run database migrations"
	@echo "  seed      - Run database seeders"
	@echo "  docker    - Build and run with Docker"

# Build the application
build:
	@echo "Building ` + projectName + `..."
	@go build -o bin/app main.go

# Run the application
run: build
	@echo "Starting ` + projectName + `..."
	@./bin/app

# Start development server
dev:
	@echo "Starting development server..."
	@routix serve

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Database migrations
migrate:
	@echo "Running migrations..."
	@routix migrate

# Database seeders
seed:
	@echo "Running seeders..."
	@routix seed`

	if config.UseDocker {
		content += `

# Docker commands
docker-build:
	@echo "Building Docker image..."
	@docker build -t ` + projectName + ` .

docker-run: docker-build
	@echo "Running with Docker..."
	@docker run -p 8080:8080 ` + projectName + `

docker-compose:
	@echo "Starting with Docker Compose..."
	@docker-compose up -d

docker-compose-dev:
	@echo "Starting development environment with Docker Compose..."
	@docker-compose -f docker-compose.dev.yml up -d`
	}

	content += `

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run

# Generate documentation
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060`

	writeFile(filepath.Join(projectName, "Makefile"), content)
}

func generateStorageKeepFiles(projectName string) {
	// Create .gitkeep files for storage directories
	keepContent := `# This file ensures the directory is tracked by Git
# You can safely delete this file once you have other files in this directory
`

	writeFile(filepath.Join(projectName, "storage", "logs", ".gitkeep"), keepContent)
	writeFile(filepath.Join(projectName, "storage", "cache", ".gitkeep"), keepContent)
}

func GenerateSwaggerFiles(projectName string, config ProjectConfig) {
	// Generate basic swagger configuration
	swaggerContent := `package docs

// @title ` + projectName + ` API
// @version 1.0
// @description This is the API documentation for ` + projectName + `
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main`

	writeFile(filepath.Join(projectName, "docs", "swagger.go"), swaggerContent)
}

func GenerateTestFiles(projectName string, config ProjectConfig) {
	// Generate basic test setup
	testContent := `package tests

import (
	"testing"
	"net/http"
	"net/http/httptest"
	
	"github.com/ramusaaa/routix"
)

func TestWelcomeEndpoint(t *testing.T) {
	// Create router
	r := routix.New()
	
	// Add route
	r.GET("/", func(c *routix.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Hello World",
		})
	})
	
	// Create request
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	
	// Create response recorder
	rr := httptest.NewRecorder()
	
	// Serve request
	r.ServeHTTP(rr, req)
	
	// Check status code
	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}
	
	// Check response body contains expected content
	expected := ` + "`" + `"message":"Hello World"` + "`" + `
	if !contains(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want to contain %v",
			rr.Body.String(), expected)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || 
		s[len(s)-len(substr):] == substr || 
		containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}`

	writeFile(filepath.Join(projectName, "tests", "welcome_test.go"), testContent)
	
	// Generate test helper
	helperContent := `package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	
	"github.com/ramusaaa/routix"
)

// TestHelper provides utilities for testing HTTP endpoints
type TestHelper struct {
	router *routix.Router
	t      *testing.T
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T, router *routix.Router) *TestHelper {
	return &TestHelper{
		router: router,
		t:      t,
	}
}

// GET performs a GET request and returns the response
func (th *TestHelper) GET(path string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		th.t.Fatal(err)
	}
	
	rr := httptest.NewRecorder()
	th.router.ServeHTTP(rr, req)
	
	return rr
}

// POST performs a POST request with JSON body
func (th *TestHelper) POST(path string, body interface{}) *httptest.ResponseRecorder {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		th.t.Fatal(err)
	}
	
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonBody))
	if err != nil {
		th.t.Fatal(err)
	}
	
	req.Header.Set("Content-Type", "application/json")
	
	rr := httptest.NewRecorder()
	th.router.ServeHTTP(rr, req)
	
	return rr
}

// AssertStatus checks if the response has the expected status code
func (th *TestHelper) AssertStatus(rr *httptest.ResponseRecorder, expectedStatus int) {
	if rr.Code != expectedStatus {
		th.t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}
}`

	writeFile(filepath.Join(projectName, "tests", "helper.go"), helperContent)
}