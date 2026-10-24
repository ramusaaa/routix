package generators

import (
	"path/filepath"
	"strings"
)

func GenerateMain(projectName string, config ProjectConfig) {
	var content string

	switch config.Template {
	case "minimal":
		if config.UseDatabase || config.UseAuth {
			content = generateAPIMain(config)
		} else {
			content = generateMinimalMain(config)
		}
	case "fullstack":
		content = generateFullstackMain(config)
	case "microservice":
		content = generateMicroserviceMain(config)
	default:
		content = generateAPIMain(config)
	}

	if config.UseSwagger {
		content = strings.Replace(content, "import (", "import (\n\t\""+config.Name+"/docs\"", 1)
		content = strings.Replace(content, "\tif err := app.Start(", "\tdocs.Register(app.Build())\n\n\tif err := app.Start(", 1)
		content = strings.Replace(content, "\tif err := r.Start(", "\tdocs.Register(r)\n\n\tif err := r.Start(", 1)
	}

	writeFile(filepath.Join(projectName, "main.go"), content)
}

func generateAPIMain(config ProjectConfig) string {
	imports := []string{
		`"github.com/ramusaaa/routix"`,
		`"log"`,
		`"net"`,
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

	if err := app.Start(net.JoinHostPort(cfg.Host, cfg.Port)); err != nil {
 log.Fatal(err)
 }
}`

	return content
}

func generateMinimalMain(config ProjectConfig) string {
	return `package main

import (
	"github.com/ramusaaa/routix"
 "log"
 "net"
 "` + config.Name + `/config"
)

func main() {
 cfg := config.Load()
	r := routix.New()

	r.GET("/", func(c *routix.Context) error {
		return c.JSON(200, map[string]any{
			"message": "Hello from Routix!",
		})
	})

	if err := r.Start(net.JoinHostPort(cfg.Host, cfg.Port)); err != nil {
 log.Fatal(err)
 }
}`
}

func generateFullstackMain(config ProjectConfig) string {
	imports := []string{
		`"github.com/ramusaaa/routix"`,
		`"log"`,
		`"net"`,
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

	routes.RegisterWeb(app)
	routes.RegisterAPI(app`
	if config.UseDatabase {
		content += `, db`
	}
	content += `)

	if err := app.Start(net.JoinHostPort(cfg.Host, cfg.Port)); err != nil {
 log.Fatal(err)
 }
}`

	return content
}

func generateMicroserviceMain(config ProjectConfig) string {
	imports := []string{
		`"github.com/ramusaaa/routix"`,
		`"log"`,
		`"net"`,
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

	if err := app.Start(net.JoinHostPort(cfg.Host, cfg.Port)); err != nil {
 log.Fatal(err)
 }
}`

	return content
}
