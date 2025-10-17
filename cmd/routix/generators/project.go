package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ProjectConfig struct {
	Name         string
	Template     string
	UseDatabase  bool
	DatabaseType string
	UseAuth      bool
	UseCache     bool
	UseQueue     bool
	UseWebSocket bool
	UseDocker    bool
	UseSwagger   bool
	UseTests     bool
	UseCORS      bool
	UseRateLimit bool
}

func GenerateGoMod(projectName string, config ProjectConfig) {
	// Start with basic requirements
	requirements := []string{
		"github.com/ramusaaa/routix v0.2.3",
	}

	// Add database dependencies
	if config.UseDatabase {
		requirements = append(requirements, "gorm.io/gorm v1.25.5")
		
		switch config.DatabaseType {
		case "postgres":
			requirements = append(requirements, "gorm.io/driver/postgres v1.5.4")
		case "mysql":
			requirements = append(requirements, "gorm.io/driver/mysql v1.5.2")
		case "sqlite":
			requirements = append(requirements, "gorm.io/driver/sqlite v1.5.4")
		}
	}

	// Add auth dependencies
	if config.UseAuth {
		requirements = append(requirements, 
			"github.com/golang-jwt/jwt/v5 v5.2.0",
			"golang.org/x/crypto v0.17.0",
		)
	}

	// Add cache dependencies
	if config.UseCache {
		requirements = append(requirements, "github.com/redis/go-redis/v9 v9.3.0")
	}

	// Add WebSocket dependencies
	if config.UseWebSocket {
		requirements = append(requirements, "github.com/gorilla/websocket v1.5.1")
	}

	// Build the go.mod content
	content := fmt.Sprintf(`module %s

go 1.21

require (`, projectName)

	for i, req := range requirements {
		if i == 0 {
			content += fmt.Sprintf("\n\t%s", req)
		} else {
			content += fmt.Sprintf("\n\t%s", req)
		}
	}

	content += "\n)"

	writeFile(filepath.Join(projectName, "go.mod"), content)
}

func GenerateMain(projectName string, config ProjectConfig) {
	var content string

	switch config.Template {
	case "minimal":
		content = generateMinimalMain(config)
	case "fullstack":
		content = generateFullstackMain(config)
	case "microservice":
		content = generateMicroserviceMain(config)
	default:
		content = generateAPIMain(config)
	}

	writeFile(filepath.Join(projectName, "main.go"), content)
}

func generateAPIMain(config ProjectConfig) string {
	imports := []string{
		`"github.com/ramusaaa/routix"`,
		`"` + config.Name + `/config"`,
		`"` + config.Name + `/routes"`,
	}

	if config.UseDatabase {
		imports = append(imports, `"`+config.Name+`/database"`)
	}

	content := `package main

import (
	` + strings.Join(imports, "\n\t") + `
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database
`

	if config.UseDatabase {
		content += `	db := database.Connect(cfg)
	defer database.Close(db)

`
	}

	content += `	// Create Routix application
	app := routix.NewAPI().
		Prod().
		JSON()`

	if config.UseCORS {
		content += `.
		CORS()`
	}

	if config.UseRateLimit {
		content += `.
		RateLimit(1000, "1m")`
	}

	content += `

	// Register routes
	routes.RegisterAPI(app`

	if config.UseDatabase {
		content += `, db`
	}

	content += `)

	// Start server
	app.Start(":" + cfg.Port)
}`

	return content
}

func generateMinimalMain(config ProjectConfig) string {
	return `package main

import (
	"github.com/ramusaaa/routix"
)

func main() {
	r := routix.New()

	r.GET("/", func(c *routix.Context) error {
		return c.JSON(200, map[string]interface{}{
			"message": "Hello from Routix!",
		})
	})

	r.Start(":8080")
}`
}

func generateFullstackMain(config ProjectConfig) string {
	return `package main

import (
	"github.com/ramusaaa/routix"
	"` + config.Name + `/config"
	"` + config.Name + `/routes"
)

func main() {
	cfg := config.Load()

	app := routix.NewAPI().
		Prod().
		JSON().
		CORS()

	// Serve static files
	app.Static("/static", "./public")

	// Register routes
	routes.RegisterWeb(app)
	routes.RegisterAPI(app)

	app.Start(":" + cfg.Port)
}`
}

func generateMicroserviceMain(config ProjectConfig) string {
	return `package main

import (
	"github.com/ramusaaa/routix"
	"` + config.Name + `/config"
	"` + config.Name + `/routes"
)

func main() {
	cfg := config.Load()

	app := routix.NewAPI().
		Prod().
		JSON().
		CORS().
		Health("/health").
		Metrics("/metrics").
		RateLimit(1000, "1m").
		Timeout("30s")

	routes.RegisterAPI(app)

	app.Start(":" + cfg.Port)
}`
}

func GenerateEnv(projectName string, config ProjectConfig) {
	content := `# Application
APP_NAME=` + projectName + `
APP_ENV=development
APP_PORT=8080
APP_HOST=localhost
APP_DEBUG=true

# Security
APP_KEY=your-secret-key-change-this-in-production
JWT_SECRET=your-jwt-secret-change-this-in-production
`

	if config.UseDatabase {
		switch config.DatabaseType {
		case "postgres":
			content += `
# Database (PostgreSQL)
DB_CONNECTION=postgres
DB_HOST=localhost
DB_PORT=5432
DB_DATABASE=` + projectName + `
DB_USERNAME=postgres
DB_PASSWORD=password
`
		case "mysql":
			content += `
# Database (MySQL)
DB_CONNECTION=mysql
DB_HOST=localhost
DB_PORT=3306
DB_DATABASE=` + projectName + `
DB_USERNAME=root
DB_PASSWORD=password
`
		case "sqlite":
			content += `
# Database (SQLite)
DB_CONNECTION=sqlite
DB_DATABASE=./storage/database.sqlite
`
		}
	}

	if config.UseCache {
		content += `
# Redis Cache
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
`
	}

	writeFile(filepath.Join(projectName, ".env"), content)
	writeFile(filepath.Join(projectName, ".env.example"), content)
}

func GenerateConfig(projectName string, config ProjectConfig) {
	content := `package config

import (
	"os"
)

type Config struct {
	AppName  string
	AppEnv   string
	Port     string
	Host     string
	Debug    bool
	AppKey   string
	JWTSecret string`

	if config.UseDatabase {
		content += `
	Database DatabaseConfig`
	}

	if config.UseCache {
		content += `
	Redis    RedisConfig`
	}

	content += `
}`

	if config.UseDatabase {
		content += `

type DatabaseConfig struct {
	Connection string
	Host       string
	Port       string
	Database   string
	Username   string
	Password   string
}`
	}

	if config.UseCache {
		content += `

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}`
	}

	content += `

func Load() *Config {
	return &Config{
		AppName:   getEnv("APP_NAME", "` + projectName + `"),
		AppEnv:    getEnv("APP_ENV", "development"),
		Port:      getEnv("APP_PORT", "8080"),
		Host:      getEnv("APP_HOST", "localhost"),
		Debug:     getEnv("APP_DEBUG", "true") == "true",
		AppKey:    getEnv("APP_KEY", "your-secret-key"),
		JWTSecret: getEnv("JWT_SECRET", "your-jwt-secret"),`

	if config.UseDatabase {
		content += `
		Database: DatabaseConfig{
			Connection: getEnv("DB_CONNECTION", "` + config.DatabaseType + `"),
			Host:       getEnv("DB_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			Database:   getEnv("DB_DATABASE", "` + projectName + `"),
			Username:   getEnv("DB_USERNAME", "postgres"),
			Password:   getEnv("DB_PASSWORD", "password"),
		},`
	}

	if config.UseCache {
		content += `
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       0,
		},`
	}

	content += `
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}`

	writeFile(filepath.Join(projectName, "config", "config.go"), content)
}

func GenerateAppStructure(projectName string, config ProjectConfig) {
	// Generate base controller
	generateBaseController(projectName)
	
	// Generate welcome controller
	generateWelcomeController(projectName)
	
	// Generate base model if database is used
	if config.UseDatabase {
		generateBaseModel(projectName)
	}
}

func generateBaseController(projectName string) {
	content := `package controllers

import (
	"github.com/ramusaaa/routix"
)

type BaseController struct{}

func (bc *BaseController) Success(c *routix.Context, data interface{}) error {
	return c.JSON(200, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

func (bc *BaseController) Error(c *routix.Context, code int, message string) error {
	return c.JSON(code, map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}

func (bc *BaseController) Created(c *routix.Context, data interface{}) error {
	return c.JSON(201, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}`

	writeFile(filepath.Join(projectName, "app", "controllers", "base_controller.go"), content)
}

func generateWelcomeController(projectName string) {
	content := `package controllers

import (
	"github.com/ramusaaa/routix"
)

type WelcomeController struct {
	BaseController
}

func (wc *WelcomeController) Index(c *routix.Context) error {
	return wc.Success(c, map[string]interface{}{
		"message": "Welcome to ` + projectName + `!",
		"version": "1.0.0",
		"framework": "Routix v0.2.0",
	})
}

func (wc *WelcomeController) Health(c *routix.Context) error {
	return wc.Success(c, map[string]interface{}{
		"status": "healthy",
		"timestamp": "` + time.Now().Format(time.RFC3339) + `",
	})
}`

	writeFile(filepath.Join(projectName, "app", "controllers", "welcome_controller.go"), content)
}

func generateBaseModel(projectName string) {
	content := `package models

import (
	"time"
	"gorm.io/gorm"
)

type BaseModel struct {
	ID        uint           ` + "`" + `gorm:"primarykey" json:"id"` + "`" + `
	CreatedAt time.Time      ` + "`" + `json:"created_at"` + "`" + `
	UpdatedAt time.Time      ` + "`" + `json:"updated_at"` + "`" + `
	DeletedAt gorm.DeletedAt ` + "`" + `gorm:"index" json:"deleted_at,omitempty"` + "`" + `
}`

	writeFile(filepath.Join(projectName, "app", "models", "base_model.go"), content)
}

func writeFile(path, content string) {
	os.MkdirAll(filepath.Dir(path), 0755)
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("❌ Error creating file %s: %v\n", path, err)
		return
	}
	defer file.Close()
	
	file.WriteString(content)
	fmt.Printf("  ✓ Created %s\n", path)
}