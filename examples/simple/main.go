package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/ramusaaa/routix"
)

// User model
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name" validate:"required,min=2"`
	Email string `json:"email" validate:"required,email"`
}

func main() {
	// Ultra basit başlangıç - tek satırda middleware'ler dahil
	app := routix.Quick()

	// Basit route tanımlama
	app.GET("/", func(c *routix.Context) error {
		return c.Success("Merhaba Dünya!")
	})

	// Fluent API ile route tanımlama
	app.Route("GET", "/hello").JSON(map[string]string{
		"message": "Hello from Routix!",
	})

	// Parametre ile route
	app.GET("/users/:id", func(c *routix.Context) error {
		id, _ := c.ParamInt("id")
		return c.Success(map[string]interface{}{
			"id":   id,
			"name": fmt.Sprintf("User %d", id),
		})
	})

	// Query parametreleri
	app.GET("/search", func(c *routix.Context) error {
		query := c.QueryParamDefault("q", "")
		page := c.QueryParamIntDefault("page", 1)
		
		return c.Success(map[string]interface{}{
			"query": query,
			"page":  page,
			"results": []string{"Result 1", "Result 2"},
		})
	})

	// RESTful resource - tek satırda CRUD
	app.Resource("/users", routix.ResourceController{
		Index: func(c *routix.Context) error {
			return c.Success([]User{
				{ID: 1, Name: "Ali", Email: "ali@example.com"},
				{ID: 2, Name: "Ayşe", Email: "ayse@example.com"},
			})
		},
		Show: func(c *routix.Context) error {
			id, _ := c.ParamInt("id")
			return c.Success(User{
				ID:    id,
				Name:  "Ali",
				Email: "ali@example.com",
			})
		},
		Create: func(c *routix.Context) error {
			var user User
			if err := c.ParseJSON(&user); err != nil {
				return c.Error(err, "Geçersiz JSON")
			}
			
			// Validation
			validator := routix.NewValidator()
			if !validator.Validate(user) {
				return c.Error(fmt.Errorf("validation failed"), "Doğrulama hatası")
			}
			
			user.ID = 123 // Simulated ID
			return c.Success(user)
		},
	})

	// API grubu - otomatik JSON header
	api := app.API("/api/v1")
	{
		api.GET("/status", func(c *routix.Context) error {
			return c.Success(map[string]string{
				"status": "OK",
				"version": "1.0.0",
			})
		})
	}

	// Static dosyalar
	app.Static("/static", "./static")

	// Method chaining ile middleware
	app.Use(routix.Logger()).
		Use(routix.Recovery()).
		WithRateLimit(100, "1m").
		WithCache("5m")

	fmt.Println("🚀 Server başlatılıyor: http://localhost:8080")
	fmt.Println("📚 Endpoints:")
	fmt.Println("  GET  /")
	fmt.Println("  GET  /hello")
	fmt.Println("  GET  /users/:id")
	fmt.Println("  GET  /search?q=test&page=1")
	fmt.Println("  GET  /users")
	fmt.Println("  POST /users")
	fmt.Println("  GET  /users/:id")
	fmt.Println("  GET  /api/v1/status")
	
	log.Fatal(http.ListenAndServe(":8080", app))
}