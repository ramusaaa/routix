package generators

import (
	"path/filepath"
	"strings"
)

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
	cfg := config.Load()

`
	if config.UseDatabase {
		content += `	db := database.Connect(cfg)
	defer database.Close(db)

`
	}

	content += `	app := routix.NewAPI().
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

	routes.RegisterAPI(app`
	if config.UseDatabase {
		content += `, db`
	}
	content += `)

	app.Start(":" + cfg.Port)
}`

	return content
}

func generateMinimalMain(_ ProjectConfig) string {
	return `package main

import (
	"github.com/ramusaaa/routix"
)

func main() {
	r := routix.New()

	r.GET("/", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{
			"message": "Hello from Routix!",
		})
	})

	r.Start(":8080")
}`
}

func generateFullstackMain(config ProjectConfig) string {
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
	cfg := config.Load()

`
	if config.UseDatabase {
		content += `	db := database.Connect(cfg)
	defer database.Close(db)

`
	}

	content += `	app := routix.NewAPI().
		Prod().
		JSON().
		CORS()

	app.Static("/static", "./public")

	routes.RegisterWeb(app`
	if config.UseDatabase {
		content += `, db`
	}
	content += `)
	routes.RegisterAPI(app`
	if config.UseDatabase {
		content += `, db`
	}
	content += `)

	app.Start(":" + cfg.Port)
}`

	return content
}

func generateMicroserviceMain(config ProjectConfig) string {
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
	cfg := config.Load()

`
	if config.UseDatabase {
		content += `	db := database.Connect(cfg)
	defer database.Close(db)

`
	}

	content += `	app := routix.NewAPI().
		Prod().
		JSON().
		CORS().
		Health("/health").
		Metrics("/metrics").
		RateLimit(1000, "1m").
		Timeout("30s")

	routes.RegisterAPI(app`
	if config.UseDatabase {
		content += `, db`
	}
	content += `)

	app.Start(":" + cfg.Port)
}`

	return content
}
