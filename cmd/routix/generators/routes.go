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
		"<title>" + projectName + " - Routix Framework</title>" +
		"<style>" +
		":root { --primary: #00dc82; --primary-dark: #00b368; --bg-dark: #0f172a; --bg-card: #1e293b; --text: #e2e8f0; --text-muted: #94a3b8; }" +
		"* { margin: 0; padding: 0; box-sizing: border-box; }" +
		"body { font-family: 'Inter', -apple-system, BlinkMacSystemFont, sans-serif; background: var(--bg-dark); min-height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; color: var(--text); }" +
		".bg-gradient { position: fixed; top: 0; left: 0; right: 0; bottom: 0; background: radial-gradient(ellipse at top, rgba(0, 220, 130, 0.15) 0%, transparent 50%), radial-gradient(ellipse at bottom right, rgba(59, 130, 246, 0.1) 0%, transparent 50%); z-index: 0; }" +
		".container { position: relative; z-index: 1; text-align: center; padding: 2rem; max-width: 600px; }" +
		".logo { width: 120px; height: 120px; margin: 0 auto 2rem; background: linear-gradient(135deg, var(--primary) 0%, var(--primary-dark) 100%); border-radius: 24px; display: flex; align-items: center; justify-content: center; font-size: 48px; font-weight: bold; color: white; box-shadow: 0 20px 40px rgba(0, 220, 130, 0.3); animation: float 3s ease-in-out infinite; }" +
		"@keyframes float { 0%, 100% { transform: translateY(0); } 50% { transform: translateY(-10px); } }" +
		"h1 { font-size: 2.5rem; font-weight: 700; margin-bottom: 0.5rem; background: linear-gradient(135deg, var(--primary), #3b82f6); -webkit-background-clip: text; -webkit-text-fill-color: transparent; background-clip: text; }" +
		".subtitle { font-size: 1.1rem; color: var(--text-muted); margin-bottom: 2rem; }" +
		".version { display: inline-block; background: var(--bg-card); padding: 0.5rem 1rem; border-radius: 9999px; font-size: 0.875rem; color: var(--primary); border: 1px solid rgba(0, 220, 130, 0.3); margin-bottom: 2rem; }" +
		".buttons { display: flex; gap: 1rem; justify-content: center; flex-wrap: wrap; margin-bottom: 3rem; }" +
		".btn { display: inline-flex; align-items: center; gap: 0.5rem; padding: 0.875rem 1.5rem; border-radius: 12px; font-size: 1rem; font-weight: 500; text-decoration: none; transition: all 0.2s ease; }" +
		".btn-primary { background: var(--primary); color: var(--bg-dark); }" +
		".btn-primary:hover { background: var(--primary-dark); transform: translateY(-2px); box-shadow: 0 10px 20px rgba(0, 220, 130, 0.3); }" +
		".btn-secondary { background: var(--bg-card); color: var(--text); border: 1px solid rgba(255, 255, 255, 0.1); }" +
		".btn-secondary:hover { background: #2d3a4f; transform: translateY(-2px); }" +
		".features { display: grid; grid-template-columns: repeat(auto-fit, minmax(140px, 1fr)); gap: 1rem; margin-bottom: 3rem; }" +
		".feature { background: var(--bg-card); padding: 1rem; border-radius: 12px; border: 1px solid rgba(255, 255, 255, 0.05); }" +
		".feature-icon { font-size: 1.5rem; margin-bottom: 0.5rem; }" +
		".feature-title { font-size: 0.8rem; font-weight: 600; color: var(--text); }" +
		".footer { position: fixed; bottom: 2rem; left: 0; right: 0; text-align: center; z-index: 1; }" +
		".powered-by { font-size: 0.875rem; color: var(--text-muted); }" +
		".powered-by a { color: var(--primary); text-decoration: none; font-weight: 500; }" +
		".powered-by a:hover { text-decoration: underline; }" +
		"</style>" +
		"</head>" +
		"<body>" +
		"<div class=\"bg-gradient\"></div>" +
		"<div class=\"container\">" +
		"<div class=\"logo\">R</div>" +
		"<h1>" + projectName + " + Routix</h1>" +
		"<p class=\"subtitle\">High-Performance Go Web Framework</p>" +
		"<span class=\"version\">v0.3.8</span>" +
		"<div class=\"buttons\">" +
		"<a href=\"https://github.com/ramusaaa/routix\" class=\"btn btn-primary\" target=\"_blank\">📚 Read Docs</a>" +
		"<a href=\"https://github.com/ramusaaa/routix\" class=\"btn btn-secondary\" target=\"_blank\">⭐ Star on GitHub</a>" +
		"</div>" +
		"<div class=\"features\">" +
		"<div class=\"feature\"><div class=\"feature-icon\">⚡</div><div class=\"feature-title\">Blazing Fast</div></div>" +
		"<div class=\"feature\"><div class=\"feature-icon\">🛠️</div><div class=\"feature-title\">Laravel CLI</div></div>" +
		"<div class=\"feature\"><div class=\"feature-icon\">🔒</div><div class=\"feature-title\">JWT Auth</div></div>" +
		"<div class=\"feature\"><div class=\"feature-icon\">🗄️</div><div class=\"feature-title\">GORM DB</div></div>" +
		"</div>" +
		"</div>" +
		"<footer class=\"footer\"><p class=\"powered-by\">Powered by <a href=\"https://github.com/ramusaaa\">Ramusa Software Corporation</a></p></footer>" +
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
