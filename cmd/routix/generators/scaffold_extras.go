package generators

import "path/filepath"

func GenerateSwaggerFiles(projectName string, config ProjectConfig) {
	swaggerContent := `package docs

// @title ` + projectName + ` API
// @version 1.0
// @description This is the API documentation for ` + projectName + `
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main`

	writeFile(filepath.Join(projectName, "docs", "swagger.go"), swaggerContent)
}

func GenerateTestFiles(projectName string, config ProjectConfig) {
	testContent := `package tests

import (
	"testing"
	"net/http"
	"net/http/httptest"

	"github.com/ramusaaa/routix"
)

func TestWelcomeEndpoint(t *testing.T) {
	r := routix.New()

	r.GET("/", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{
			"message": "Hello World",
		})
	})

	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v",
			status, http.StatusOK)
	}

	expected := ` + "`" + `"message":"Hello World"` + "`" + `
	if !containsStr(rr.Body.String(), expected) {
		t.Errorf("handler returned unexpected body: got %v want to contain %v",
			rr.Body.String(), expected)
	}
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}`

	writeFile(filepath.Join(projectName, "tests", "welcome_test.go"), testContent)

	helperContent := `package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ramusaaa/routix"
)

type TestHelper struct {
	router *routix.Router
	t      *testing.T
}

func NewTestHelper(t *testing.T, router *routix.Router) *TestHelper {
	return &TestHelper{router: router, t: t}
}

func (th *TestHelper) GET(path string) *httptest.ResponseRecorder {
	req, err := http.NewRequest("GET", path, nil)
	if err != nil {
		th.t.Fatal(err)
	}
	rr := httptest.NewRecorder()
	th.router.ServeHTTP(rr, req)
	return rr
}

func (th *TestHelper) POST(path string, body interface{}) *httptest.ResponseRecorder {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		th.t.Fatal(err)
	}
	req, err := http.NewRequest("POST", path, bytes.NewBuffer(jsonBody))
	if err != nil {
		th.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	th.router.ServeHTTP(rr, req)
	return rr
}

func (th *TestHelper) AssertStatus(rr *httptest.ResponseRecorder, expectedStatus int) {
	if rr.Code != expectedStatus {
		th.t.Errorf("Expected status %d, got %d", expectedStatus, rr.Code)
	}
}`

	writeFile(filepath.Join(projectName, "tests", "helper.go"), helperContent)
}
