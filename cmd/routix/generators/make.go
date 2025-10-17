package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func GenerateController(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "controllers", filename)

	var content string
	if options["resource"] {
		content = generateResourceController(name)
	} else {
		content = generateBasicController(name)
	}

	return writeFileWithError(path, content)
}

func GenerateModel(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "models", filename)

	content := generateModelContent(name)
	return writeFileWithError(path, content)
}

func GenerateMiddlewareFile(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + ".go"
	path := filepath.Join("app", "middleware", filename)

	content := generateMiddlewareContent(name)
	return writeFileWithError(path, content)
}

func GenerateMigration(name string, options map[string]bool) error {
	timestamp := time.Now().Format("20060102150405")
	filename := fmt.Sprintf("%s_%s.go", timestamp, strings.ToLower(name))
	path := filepath.Join("database", "migrations", filename)

	content := generateMigrationContent(name)
	return writeFileWithError(path, content)
}

func GenerateSeeder(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_seeder.go"
	path := filepath.Join("database", "seeders", filename)

	content := generateSeederContent(name)
	return writeFileWithError(path, content)
}

func GenerateService(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_service.go"
	path := filepath.Join("app", "services", filename)

	content := generateServiceContent(name)
	return writeFileWithError(path, content)
}

func GenerateRequest(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_request.go"
	path := filepath.Join("app", "requests", filename)

	content := generateRequestContent(name)
	return writeFileWithError(path, content)
}

func GenerateResource(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_resource.go"
	path := filepath.Join("app", "resources", filename)

	content := generateResourceContent(name)
	return writeFileWithError(path, content)
}

func GenerateTest(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_test.go"
	path := filepath.Join("tests", filename)

	content := generateTestContent(name)
	return writeFileWithError(path, content)
}

func GenerateModule(name string, options map[string]bool) error {
	modulePath := filepath.Join("app", "modules", strings.ToLower(name))
	
	dirs := []string{
		"controllers",
		"models",
		"services",
		"routes",
	}

	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(modulePath, dir), 0755)
	}

	moduleContent := generateModuleContent(name)
	return writeFileWithError(filepath.Join(modulePath, "module.go"), moduleContent)
}

func GenerateJob(name string, options map[string]bool) error {
	filename := strings.ToLower(name) + "_job.go"
	path := filepath.Join("app", "jobs", filename)

	content := generateJobContent(name)
	return writeFileWithError(path, content)
}

func generateBasicController(name string) string {
	return fmt.Sprintf(`package controllers

import (
	"github.com/ramusaaa/routix"
)

type %s struct {
	BaseController
}

func (ctrl *%s) Index(c *routix.Context) error {
	return ctrl.Success(c, map[string]interface{}{
		"message": "Hello from %s",
	})
}`, name, name, name)
}

func generateResourceController(name string) string {
	modelName := strings.TrimSuffix(name, "Controller")
	return fmt.Sprintf(`package controllers

import (
	"github.com/ramusaaa/routix"
)

type %s struct {
	BaseController
}

// GET /resource
func (ctrl *%s) Index(c *routix.Context) error {
	// TODO: Implement index logic
	return ctrl.Success(c, []interface{}{})
}

// POST /resource
func (ctrl *%s) Store(c *routix.Context) error {
	// TODO: Implement store logic
	return ctrl.Created(c, map[string]interface{}{
		"message": "%s created successfully",
	})
}

// GET /resource/{id}
func (ctrl *%s) Show(c *routix.Context) error {
	id := c.Params["id"]
	// TODO: Implement show logic
	return ctrl.Success(c, map[string]interface{}{
		"id": id,
	})
}

// PUT /resource/{id}
func (ctrl *%s) Update(c *routix.Context) error {
	id := c.Params["id"]
	// TODO: Implement update logic
	return ctrl.Success(c, map[string]interface{}{
		"id": id,
		"message": "%s updated successfully",
	})
}

// DELETE /resource/{id}
func (ctrl *%s) Destroy(c *routix.Context) error {
	id := c.Params["id"]
	// TODO: Implement destroy logic
	return ctrl.Success(c, map[string]interface{}{
		"message": "%s deleted successfully",
	})
}`, name, name, name, modelName, name, name, modelName, name, modelName)
}

func generateModelContent(name string) string {
	return fmt.Sprintf(`package models

import (
	"gorm.io/gorm"
)

type %s struct {
	BaseModel
	// Add your fields here
	Name string ` + "`" + `gorm:"not null" json:"name"` + "`" + `
}

func (m *%s) TableName() string {
	return "%s"
}

// Model methods
func (m *%s) BeforeCreate(tx *gorm.DB) error {
	// Add any logic before creating
	return nil
}

func (m *%s) AfterCreate(tx *gorm.DB) error {
	// Add any logic after creating
	return nil
}`, name, name, strings.ToLower(name)+"s", name, name)
}

func generateMiddlewareContent(name string) string {
	return fmt.Sprintf(`package middleware

import (
	"github.com/ramusaaa/routix"
)

func %s() routix.Middleware {
	return func(next routix.Handler) routix.Handler {
		return func(c *routix.Context) error {
			// TODO: Implement %s middleware logic
			
			// Continue to next handler
			return next(c)
		}
	}
}`, name, name)
}

func generateMigrationContent(name string) string {
	return fmt.Sprintf(`package migrations

import (
	"gorm.io/gorm"
)

func init() {
	RegisterMigration(&Migration{
		ID:   "%s",
		Up:   up_%s,
		Down: down_%s,
	})
}

func up_%s(db *gorm.DB) error {
	// TODO: Implement migration up logic
	return nil
}

func down_%s(db *gorm.DB) error {
	// TODO: Implement migration down logic
	return nil
}`, name, strings.ReplaceAll(name, " ", "_"), strings.ReplaceAll(name, " ", "_"), strings.ReplaceAll(name, " ", "_"), strings.ReplaceAll(name, " ", "_"))
}

func generateSeederContent(name string) string {
	return fmt.Sprintf(`package seeders

import (
	"gorm.io/gorm"
)

type %sSeeder struct{}

func (s *%sSeeder) Run(db *gorm.DB) error {
	// TODO: Implement seeder logic
	return nil
}`, name, name)
}

func generateServiceContent(name string) string {
	return fmt.Sprintf(`package services

type %sService struct {
	// Add dependencies here
}

func New%sService() *%sService {
	return &%sService{}
}

func (s *%sService) Process() error {
	// TODO: Implement service logic
	return nil
}`, name, name, name, name, name)
}

func generateRequestContent(name string) string {
	return fmt.Sprintf(`package requests

import (
	"github.com/ramusaaa/routix"
)

type %sRequest struct {
	// Add validation fields here
	Name string ` + "`" + `json:"name" validate:"required"` + "`" + `
}

func (r *%sRequest) Validate(c *routix.Context) error {
	// TODO: Implement validation logic
	return nil
}`, name, name)
}

func generateResourceContent(name string) string {
	return fmt.Sprintf(`package resources

type %sResource struct {
	ID   uint   ` + "`" + `json:"id"` + "`" + `
	Name string ` + "`" + `json:"name"` + "`" + `
	// Add other fields here
}

func New%sResource(data interface{}) *%sResource {
	// TODO: Transform data to resource
	return &%sResource{}
}

func New%sCollection(data []interface{}) []*%sResource {
	var resources []*%sResource
	for _, item := range data {
		resources = append(resources, New%sResource(item))
	}
	return resources
}`, name, name, name, name, name, name, name, name)
}

func generateTestContent(name string) string {
	return fmt.Sprintf(`package tests

import (
	"testing"
)

func Test%s(t *testing.T) {
	// TODO: Implement test logic
	t.Log("Test %s")
}`, name, name)
}

func generateModuleContent(name string) string {
	return fmt.Sprintf(`package %s

import (
	"github.com/ramusaaa/routix"
)

type %sModule struct {
	// Add module dependencies here
}

func New%sModule() *%sModule {
	return &%sModule{}
}

func (m *%sModule) RegisterRoutes(r *routix.Router) {
	// TODO: Register module routes
}

func (m *%sModule) Boot() error {
	// TODO: Module initialization logic
	return nil
}`, strings.ToLower(name), name, name, name, name, name, name)
}

func generateJobContent(name string) string {
	return fmt.Sprintf(`package jobs

type %sJob struct {
	// Add job data here
}

func New%sJob() *%sJob {
	return &%sJob{}
}

func (j *%sJob) Handle() error {
	// TODO: Implement job logic
	return nil
}

func (j *%sJob) Failed(err error) {
	// TODO: Handle job failure
}`, name, name, name, name, name, name)
}

func writeFileWithError(path, content string) error {
	os.MkdirAll(filepath.Dir(path), 0755)
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	return err
}