package generators

import (
	"fmt"
	"os"
	"path/filepath"
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
	requirements := []string{
		"github.com/ramusaaa/routix v0.4.0",
	}

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

	if config.UseAuth {
		requirements = append(requirements,
			"github.com/golang-jwt/jwt/v5 v5.2.0",
			"golang.org/x/crypto v0.17.0",
		)
	}

	if config.UseCache {
		requirements = append(requirements, "github.com/redis/go-redis/v9 v9.3.0")
	}

	if config.UseWebSocket {
		requirements = append(requirements, "github.com/gorilla/websocket v1.5.1")
	}

	content := fmt.Sprintf("module %s\n\ngo 1.21\n\nrequire (", projectName)
	for _, req := range requirements {
		content += fmt.Sprintf("\n\t%s", req)
	}
	content += "\n)"

	writeFile(filepath.Join(projectName, "go.mod"), content)
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
	AppName   string
	AppEnv    string
	Port      string
	Host      string
	Debug     bool
	AppKey    string
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

func writeFile(path, content string) {
	os.MkdirAll(filepath.Dir(path), 0755)
	file, err := os.Create(path)
	if err != nil {
		fmt.Printf("  error: creating file %s: %v\n", path, err)
		return
	}
	defer file.Close()
	file.WriteString(content)
	fmt.Printf("  + %s\n", path)
}
