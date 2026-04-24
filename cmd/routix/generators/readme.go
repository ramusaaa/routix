package generators

import (
	"fmt"
	"path/filepath"
)

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
