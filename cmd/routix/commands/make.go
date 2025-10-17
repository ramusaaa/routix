package commands

import (
	"fmt"
	"strings"

	"github.com/ramusaaa/routix/cmd/routix/generators"
)

func MakeCommand(args []string) {
	if len(args) < 1 {
		showMakeHelp()
		return
	}

	command := args[0]
	makeArgs := args[1:]

	switch command {
	case "controller":
		makeController(makeArgs)
	case "model":
		makeModel(makeArgs)
	case "middleware":
		makeMiddleware(makeArgs)
	case "migration":
		makeMigration(makeArgs)
	case "seeder":
		makeSeeder(makeArgs)
	case "service":
		makeService(makeArgs)
	case "request":
		makeRequest(makeArgs)
	case "resource":
		makeResource(makeArgs)
	case "test":
		makeTest(makeArgs)
	case "module":
		makeModule(makeArgs)
	case "job":
		makeJob(makeArgs)
	default:
		fmt.Printf("Unknown make command: %s\n", command)
		showMakeHelp()
	}
}

func showMakeHelp() {
	fmt.Println(`
Make Commands:
  routix make:controller <name>   Create a new controller
  routix make:model <name>        Create a new model
  routix make:middleware <name>   Create a new middleware
  routix make:migration <name>    Create a new migration
  routix make:seeder <name>       Create a new seeder
  routix make:service <name>      Create a new service
  routix make:request <name>      Create a new request validator
  routix make:resource <name>     Create a new API resource
  routix make:test <name>         Create a new test
  routix make:module <name>       Create a new module
  routix make:job <name>          Create a new job

Options:
  --resource                      Create resource controller with CRUD methods
  --api                          Create API controller
  --model                        Also create model (for controllers)
  --migration                    Also create migration (for models)

Examples:
  routix make:controller UserController
  routix make:controller UserController --resource --model
  routix make:model User --migration
  routix make:middleware Auth
`)
}

func makeController(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:controller <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("🎮 Creating controller: %s\n", name)

	if err := generators.GenerateController(name, options); err != nil {
		fmt.Printf("❌ Error creating controller: %v\n", err)
		return
	}

	fmt.Printf("✅ Controller created: app/controllers/%s.go\n", strings.ToLower(name))

	if options["model"] {
		modelName := strings.TrimSuffix(name, "Controller")
		if err := generators.GenerateModel(modelName, options); err != nil {
			fmt.Printf("⚠️  Warning: Could not create model: %v\n", err)
		} else {
			fmt.Printf("✅ Model created: app/models/%s.go\n", strings.ToLower(modelName))
		}
	}
}

func makeModel(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:model <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("📊 Creating model: %s\n", name)

	if err := generators.GenerateModel(name, options); err != nil {
		fmt.Printf("❌ Error creating model: %v\n", err)
		return
	}

	fmt.Printf("✅ Model created: app/models/%s.go\n", strings.ToLower(name))

	if options["migration"] {
		migrationName := fmt.Sprintf("create_%s_table", strings.ToLower(name)+"s")
		if err := generators.GenerateMigration(migrationName, options); err != nil {
			fmt.Printf("⚠️  Warning: Could not create migration: %v\n", err)
		} else {
			fmt.Printf("✅ Migration created: database/migrations/%s.go\n", migrationName)
		}
	}
}

func makeMiddleware(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:middleware <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("🛡️  Creating middleware: %s\n", name)

	if err := generators.GenerateMiddlewareFile(name, options); err != nil {
		fmt.Printf("❌ Error creating middleware: %v\n", err)
		return
	}

	fmt.Printf("✅ Middleware created: app/middleware/%s.go\n", strings.ToLower(name))
}

func makeMigration(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:migration <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("🗄️  Creating migration: %s\n", name)

	if err := generators.GenerateMigration(name, options); err != nil {
		fmt.Printf("❌ Error creating migration: %v\n", err)
		return
	}

	fmt.Printf("✅ Migration created: database/migrations/%s.go\n", name)
}

func makeSeeder(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:seeder <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("🌱 Creating seeder: %s\n", name)

	if err := generators.GenerateSeeder(name, options); err != nil {
		fmt.Printf("❌ Error creating seeder: %v\n", err)
		return
	}

	fmt.Printf("✅ Seeder created: database/seeders/%s.go\n", strings.ToLower(name))
}

func makeService(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:service <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("⚙️  Creating service: %s\n", name)

	if err := generators.GenerateService(name, options); err != nil {
		fmt.Printf("❌ Error creating service: %v\n", err)
		return
	}

	fmt.Printf("✅ Service created: app/services/%s.go\n", strings.ToLower(name))
}

func makeRequest(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:request <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("📝 Creating request validator: %s\n", name)

	if err := generators.GenerateRequest(name, options); err != nil {
		fmt.Printf("❌ Error creating request: %v\n", err)
		return
	}

	fmt.Printf("✅ Request created: app/requests/%s.go\n", strings.ToLower(name))
}

func makeResource(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:resource <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("📦 Creating API resource: %s\n", name)

	if err := generators.GenerateResource(name, options); err != nil {
		fmt.Printf("❌ Error creating resource: %v\n", err)
		return
	}

	fmt.Printf("✅ Resource created: app/resources/%s.go\n", strings.ToLower(name))
}

func makeTest(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:test <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("🧪 Creating test: %s\n", name)

	if err := generators.GenerateTest(name, options); err != nil {
		fmt.Printf("❌ Error creating test: %v\n", err)
		return
	}

	fmt.Printf("✅ Test created: tests/%s.go\n", strings.ToLower(name))
}

func makeModule(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:module <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("📦 Creating module: %s\n", name)

	if err := generators.GenerateModule(name, options); err != nil {
		fmt.Printf("❌ Error creating module: %v\n", err)
		return
	}

	fmt.Printf("✅ Module created: app/modules/%s/\n", strings.ToLower(name))
}

func makeJob(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix make:job <name>")
		return
	}

	name := args[0]
	options := parseOptions(args[1:])

	fmt.Printf("💼 Creating job: %s\n", name)

	if err := generators.GenerateJob(name, options); err != nil {
		fmt.Printf("❌ Error creating job: %v\n", err)
		return
	}

	fmt.Printf("✅ Job created: app/jobs/%s.go\n", strings.ToLower(name))
}

func parseOptions(args []string) map[string]bool {
	options := make(map[string]bool)
	
	for _, arg := range args {
		if strings.HasPrefix(arg, "--") {
			option := strings.TrimPrefix(arg, "--")
			options[option] = true
		}
	}
	
	return options
}