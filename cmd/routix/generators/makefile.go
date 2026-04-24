package generators

import "path/filepath"

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
