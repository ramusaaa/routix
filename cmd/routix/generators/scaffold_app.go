package generators

import "path/filepath"

func GenerateAppStructure(projectName string, config ProjectConfig) {
	generateBaseController(projectName)
	generateWelcomeController(projectName)
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

func (bc *BaseController) Success(c *routix.Context, data any) error {
	return c.JSON(200, map[string]any{
		"status": "success",
		"data":   data,
	})
}

func (bc *BaseController) Error(c *routix.Context, code int, message string) error {
	return c.JSON(code, map[string]any{
		"status":  "error",
		"message": message,
	})
}

func (bc *BaseController) Created(c *routix.Context, data any) error {
	return c.JSON(201, map[string]any{
		"status": "success",
		"data":   data,
	})
}`

	writeFile(filepath.Join(projectName, "app", "controllers", "base_controller.go"), content)
}

func generateWelcomeController(projectName string) {
	content := `package controllers

import (
	"time"

	"github.com/ramusaaa/routix"
)

type WelcomeController struct {
	BaseController
}

func (wc *WelcomeController) Index(c *routix.Context) error {
	return wc.Success(c, map[string]any{
		"message":   "Welcome to ` + projectName + `!",
		"version":   "1.0.0",
		"framework": "Routix v0.4.0",
	})
}

func (wc *WelcomeController) Health(c *routix.Context) error {
	return wc.Success(c, map[string]any{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
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
