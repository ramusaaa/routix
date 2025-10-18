package generators

import (
	"path/filepath"
)

func GenerateRoutes(projectName string, config ProjectConfig) {
	generateAPIRoutes(projectName, config)
	
	if config.Template == "fullstack" {
		generateWebRoutes(projectName, config)
	}
}

func generateAPIRoutes(projectName string, config ProjectConfig) {
	content := `package routes

import (
	"github.com/ramusaaa/routix"
	"` + projectName + `/app/controllers"`

	if config.UseDatabase {
		content += `
	"gorm.io/gorm"`
	}

	content += `
)

func RegisterAPI(app *routix.APIBuilder`

	if config.UseDatabase {
		content += `, db *gorm.DB`
	}

	content += `) {
	// Initialize controllers
	welcomeController := &controllers.WelcomeController{}`

	if config.UseAuth {
		content += `
	authController := controllers.NewAuthController()`
	}

	content += `

	// Public routes
	app.GET("/", welcomeController.Index)
	app.GET("/health", welcomeController.Health)`

	if config.UseAuth {
		content += `

	// Auth routes
	app.POST("/auth/register", authController.Register)
	app.POST("/auth/login", authController.Login)
	app.POST("/auth/refresh", authController.RefreshToken)
	
	// Protected auth routes (add middleware later)
	app.POST("/auth/logout", authController.Logout)
	app.GET("/auth/me", authController.Me)`
	}

	content += `

	// API v1 routes
	app.V1(func(v1 *routix.Group) {
		// Add your API routes here
		v1.GET("/status", func(c *routix.Context) error {
			return c.JSON(200, map[string]any{
				"status": "ok",
				"version": "1.0.0",
			})
		})
		
		// Example resource routes
		// users := v1.Group("/users")
		// users.Use(middleware.Auth()) // Protect user routes
		// users.GET("", userController.Index)
		// users.POST("", userController.Store)
		// users.GET("/:id", userController.Show)
		// users.PUT("/:id", userController.Update)
		// users.DELETE("/:id", userController.Destroy)
	})
}`

	writeFile(filepath.Join(projectName, "routes", "api.go"), content)
}

func generateWebRoutes(projectName string, config ProjectConfig) {
	content := `package routes

import (
	"github.com/ramusaaa/routix"
	"` + projectName + `/app/controllers"
)

func RegisterWeb(app *routix.APIBuilder) {
	welcomeController := &controllers.WelcomeController{}

	// Web routes
	app.GET("/", func(c *routix.Context) error {
		return c.HTML(200, getWelcomePage("` + projectName + `"))
	})
	
	app.GET("/about", func(c *routix.Context) error {
		return c.HTML(200, getAboutPage("` + projectName + `"))
	})
	
	// API endpoints for frontend
	app.GET("/api/welcome", welcomeController.Index)
}

func getWelcomePage(projectName string) string {
	return "<!DOCTYPE html>" +
		"<html lang=\"en\">" +
		"<head>" +
		"<meta charset=\"UTF-8\">" +
		"<meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">" +
		"<title>Welcome to " + projectName + "</title>" +
		"<style>" +
		"body { font-family: Arial, sans-serif; margin: 0; padding: 40px; background: #f5f5f5; }" +
		".container { max-width: 800px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }" +
		"h1 { color: #333; text-align: center; }" +
		".features { display: grid; grid-template-columns: repeat(auto-fit, minmax(250px, 1fr)); gap: 20px; margin: 30px 0; }" +
		".feature { padding: 20px; background: #f8f9fa; border-radius: 6px; }" +
		"button { background: #007bff; color: white; border: none; padding: 10px 20px; border-radius: 4px; cursor: pointer; }" +
		"</style>" +
		"</head>" +
		"<body>" +
		"<div class=\"container\">" +
		"<h1>Welcome to " + projectName + "!</h1>" +
		"<p>Your Routix application is up and running successfully.</p>" +
		"<div class=\"features\">" +
		"<div class=\"feature\">" +
		"<h3>Fast Performance</h3>" +
		"<p>Built with Go for maximum performance and efficiency.</p>" +
		"</div>" +
		"<div class=\"feature\">" +
		"<h3>Developer Friendly</h3>" +
		"<p>Laravel-inspired CLI tools and project structure.</p>" +
		"</div>" +
		"</div>" +
		"<p>Built with Routix v0.3.8 - Laravel-inspired Go Web Framework</p>" +
		"</div>" +
		"</body>" +
		"</html>"
}

func getAboutPage(projectName string) string {
	return "<!DOCTYPE html>" +
		"<html lang=\"en\">" +
		"<head>" +
		"<meta charset=\"UTF-8\">" +
		"<title>About - " + projectName + "</title>" +
		"<style>" +
		"body { font-family: Arial, sans-serif; margin: 0; padding: 40px; background: #f5f5f5; }" +
		".container { max-width: 800px; margin: 0 auto; background: white; padding: 40px; border-radius: 8px; }" +
		"</style>" +
		"</head>" +
		"<body>" +
		"<div class=\"container\">" +
		"<a href=\"/\">← Back to Home</a>" +
		"<h1>About " + projectName + "</h1>" +
		"<p>This is a Routix-powered application.</p>" +
		"</div>" +
		"</body>" +
		"</html>"
}`

	writeFile(filepath.Join(projectName, "routes", "web.go"), content)
}