package generators

import (
	"path/filepath"
	"strings"
)

func GenerateDockerFiles(projectName string, config ProjectConfig) {
	generateDockerfile(projectName, config)
	generateDockerCompose(projectName, config)
	generateDockerIgnore(projectName)
}

func generateDockerfile(projectName string, config ProjectConfig) {
	content := `# Build stage
FROM golang:1.26.10-alpine AS builder

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create app directory
WORKDIR /root/

# Copy binary from builder stage
COPY --from=builder /app/main .

# Create storage directories
RUN mkdir -p storage/logs storage/cache

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]`

	if config.UseDatabase && config.DatabaseType == "sqlite" {
		content = strings.Replace(content, "apk add --no-cache git ca-certificates", "apk add --no-cache git ca-certificates gcc musl-dev", 1)
		content = strings.Replace(content, "CGO_ENABLED=0", "CGO_ENABLED=1", 1)
	}

	writeFile(filepath.Join(projectName, "Dockerfile"), content)
}

func generateDockerCompose(projectName string, config ProjectConfig) {
	content := `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      - APP_ENV=production
      - APP_PORT=8080
      - APP_HOST=0.0.0.0`

	if config.UseDatabase {
		content += `
      - DB_HOST=database`

		switch config.DatabaseType {
		case "postgres":
			content += `
      - DB_PORT=5432
      - DB_DATABASE=` + projectName + `
      - DB_USERNAME=postgres
      - DB_PASSWORD=password`
		case "mysql":
			content += `
      - DB_PORT=3306
      - DB_DATABASE=` + projectName + `
      - DB_USERNAME=root
      - DB_PASSWORD=password`
		}
	}

	if config.UseCache {
		content += `
      - REDIS_HOST=redis
      - REDIS_PORT=6379`
	}

	if config.UseCache || config.UseDatabase && config.DatabaseType != "sqlite" {
		content += `
    depends_on:`
	}

	if config.UseDatabase && config.DatabaseType != "sqlite" {
		content += `
      - database`
	}

	if config.UseCache {
		content += `
      - redis`
	}

	content += `
    volumes:
      - ./storage:/root/storage
    restart: unless-stopped
    networks:
      - app-network`

	if config.UseDatabase {
		switch config.DatabaseType {
		case "postgres":
			content += `

  database:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ` + projectName + `
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped
    networks:
      - app-network`
		case "mysql":
			content += `

  database:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: ` + projectName + `
      MYSQL_ROOT_PASSWORD: password
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql
    restart: unless-stopped
    networks:
      - app-network`
		}
	}

	if config.UseCache {
		content += `

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    volumes:
      - redis_data:/data
    restart: unless-stopped
    networks:
      - app-network`
	}

	content += `

networks:
  app-network:
    driver: bridge`

	if config.UseCache || config.UseDatabase && config.DatabaseType != "sqlite" {
		content += "\n\nvolumes:"
	}
	if config.UseDatabase {
		switch config.DatabaseType {
		case "postgres":
			content += `
  postgres_data:`
		case "mysql":
			content += `
  mysql_data:`
		}
	}

	if config.UseCache {
		content += `
  redis_data:`
	}

	writeFile(filepath.Join(projectName, "docker-compose.yml"), content)

	// Also create development docker-compose
	devContent := `version: '3.8'

services:
  app:
    build: .
    ports:
      - "8080:8080"
    env_file:
      - .env
    environment:
      - APP_ENV=development
      - APP_DEBUG=true
      - APP_PORT=8080
      - APP_HOST=0.0.0.0
    volumes:
      - .:/root/app
      - ./storage:/root/storage`
	if config.UseCache || config.UseDatabase && config.DatabaseType != "sqlite" {
		devContent += `
    depends_on:`
	}

	if config.UseDatabase && config.DatabaseType != "sqlite" {
		devContent += `
      - database`
	}

	if config.UseCache {
		devContent += `
      - redis`
	}

	devContent += `
    networks:
      - app-network`

	if config.UseDatabase {
		switch config.DatabaseType {
		case "postgres":
			devContent += `

  database:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ` + projectName + `_dev
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
    ports:
      - "5432:5432"
    volumes:
      - postgres_dev_data:/var/lib/postgresql/data
    networks:
      - app-network`
		case "mysql":
			devContent += `

  database:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: ` + projectName + `_dev
      MYSQL_ROOT_PASSWORD: password
    ports:
      - "3306:3306"
    volumes:
      - mysql_dev_data:/var/lib/mysql
    networks:
      - app-network`
		}
	}

	if config.UseCache {
		devContent += `

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    networks:
      - app-network`
	}

	devContent += `

networks:
  app-network:
    driver: bridge`

	if config.UseDatabase && config.DatabaseType != "sqlite" {
		devContent += "\n\nvolumes:"
	}
	if config.UseDatabase {
		switch config.DatabaseType {
		case "postgres":
			devContent += `
  postgres_dev_data:`
		case "mysql":
			devContent += `
  mysql_dev_data:`
		}
	}

	writeFile(filepath.Join(projectName, "docker-compose.dev.yml"), devContent)
}

func generateDockerIgnore(projectName string) {
	content := `# Git
.git
.gitignore

# Documentation
README.md

# Development files
.env
.env.*
!.env.example
*.log

# IDE
.vscode/
.idea/
*.swp
*.swo

# OS
.DS_Store
Thumbs.db

# Dependencies
vendor/

# Build artifacts
*.exe
*.dll
*.so
*.dylib

# Test files
*_test.go
coverage.out

# Temporary files
tmp/
temp/

# Storage (keep structure but not content)
storage/logs/*
storage/cache/*
!storage/logs/.gitkeep
!storage/cache/.gitkeep`

	writeFile(filepath.Join(projectName, ".dockerignore"), content)
}
