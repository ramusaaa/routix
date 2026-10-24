package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/ramusaaa/routix/cmd/routix/generators"
)

type ProjectConfig = generators.ProjectConfig

func NewProject(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: routix new <project-name>")
	}

	projectName := args[0]
	if !regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`).MatchString(projectName) {
		return fmt.Errorf("project name must contain letters, digits, underscores or hyphens")
	}
	if _, err := os.Lstat(projectName); err == nil {
		return fmt.Errorf("target already exists: %s", projectName)
	} else if !os.IsNotExist(err) {
		return err
	}
	fmt.Printf("Creating new Routix project: %s\n\n", projectName)

	config := generators.ProjectConfig{Name: projectName}
	askProjectQuestions(&config)
	config = generators.NormalizeProjectConfig(config)

	if err := os.MkdirAll(projectName, 0755); err != nil {
		return fmt.Errorf("create project directory: %w", err)
	}

	fmt.Printf("\nGenerating project structure...\n")
	generateProjectStructure(projectName, config)

	fmt.Printf("\nInstalling dependencies...\n")
	if err := runGoModTidy(projectName); err != nil {
		return err
	}

	fmt.Printf("\nProject created successfully!\n\n")
	printNextSteps(projectName, config)
	return nil
}

func askProjectQuestions(config *generators.ProjectConfig) {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("Select a project template:")
	fmt.Println("  1. API Only (RESTful API)")
	fmt.Println("  2. Full-stack (API + Frontend)")
	fmt.Println("  3. Microservice (Production-ready)")
	fmt.Println("  4. Minimal (Basic setup)")

	templateChoice := readChoice(reader, "Choose template (1-4)", []string{"1", "2", "3", "4"}, "1")
	switch templateChoice {
	case "2":
		config.Template = "fullstack"
	case "3":
		config.Template = "microservice"
	case "4":
		config.Template = "minimal"
	default:
		config.Template = "api"
	}

	fmt.Print("\nAdd database support? [Y/n]: ")
	config.UseDatabase = readBoolWithValidation(reader, true)

	if config.UseDatabase {
		fmt.Println("  Select database:")
		fmt.Println("  1. PostgreSQL (recommended)")
		fmt.Println("  2. MySQL")
		fmt.Println("  3. SQLite (development)")

		dbChoice := readChoice(reader, "Choose database (1-3)", []string{"1", "2", "3"}, "1")
		switch dbChoice {
		case "2":
			config.DatabaseType = "mysql"
		case "3":
			config.DatabaseType = "sqlite"
		default:
			config.DatabaseType = "postgres"
		}
	}

	fmt.Print("Add authentication (JWT + bcrypt)? [Y/n]: ")
	config.UseAuth = readBoolWithValidation(reader, true)
	if config.UseAuth && !config.UseDatabase {
		fmt.Println("Authentication requires persistent storage; SQLite will be included.")
	}

	fmt.Print("Add caching (Redis)? [y/N]: ")
	config.UseCache = readBoolWithValidation(reader, false)

	fmt.Print("Add job queue system? [y/N]: ")
	config.UseQueue = readBoolWithValidation(reader, false)

	fmt.Print("Add WebSocket support? [y/N]: ")
	config.UseWebSocket = readBoolWithValidation(reader, false)

	fmt.Print("Add Docker support? [Y/n]: ")
	config.UseDocker = readBoolWithValidation(reader, true)

	fmt.Print("Add Swagger documentation? [Y/n]: ")
	config.UseSwagger = readBoolWithValidation(reader, true)

	fmt.Print("Add testing setup? [Y/n]: ")
	config.UseTests = readBoolWithValidation(reader, true)

	fmt.Print("Add CORS support? [Y/n]: ")
	config.UseCORS = readBoolWithValidation(reader, true)

	fmt.Print("Add rate limiting? [Y/n]: ")
	config.UseRateLimit = readBoolWithValidation(reader, true)
}

func readInput(reader *bufio.Reader, defaultValue string) string {
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultValue
	}
	return input
}

func readBool(reader *bufio.Reader, defaultValue bool) bool {
	input := readInput(reader, "")
	if input == "" {
		return defaultValue
	}
	return strings.ToLower(input) == "y" || strings.ToLower(input) == "yes"
}

func readBoolWithValidation(reader *bufio.Reader, defaultValue bool) bool {
	for {
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		if input == "" {
			return defaultValue
		}

		if input == "y" || input == "yes" {
			return true
		}

		if input == "n" || input == "no" {
			return false
		}

		fmt.Print("please enter 'y' or 'n': ")
	}
}

func readChoice(reader *bufio.Reader, prompt string, validChoices []string, defaultChoice string) string {
	for {
		fmt.Printf("%s [%s]: ", prompt, defaultChoice)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "" {
			return defaultChoice
		}

		for _, choice := range validChoices {
			if input == choice {
				return input
			}
		}

		fmt.Printf("invalid choice, select from: %s\n", strings.Join(validChoices, ", "))
	}
}

func generateProjectStructure(projectName string, config generators.ProjectConfig) {
	config = generators.NormalizeProjectConfig(config)
	createDirectoryStructure(projectName, config)

	generators.GenerateGoMod(projectName, config)
	generators.GenerateMain(projectName, config)
	generators.GenerateEnv(projectName, config)
	generators.GenerateConfig(projectName, config)

	generators.GenerateAppStructure(projectName, config)

	if config.UseDatabase {
		generators.GenerateDatabaseFiles(projectName, config)
	}

	if config.UseAuth {
		generators.GenerateAuthFiles(projectName, config)
	}

	generators.GenerateMiddleware(projectName, config)

	if config.UseDocker {
		generators.GenerateDockerFiles(projectName, config)
	}

	if config.UseSwagger {
		generators.GenerateSwaggerFiles(projectName, config)
	}

	if config.UseTests {
		generators.GenerateTestFiles(projectName, config)
	}

	generators.GenerateCommonFiles(projectName, config)
}

func createDirectoryStructure(projectName string, config generators.ProjectConfig) {
	dirs := []string{
		"app/controllers",
		"app/models",
		"app/middleware",
		"app/services",
		"app/requests",
		"app/resources",
		"config",
		"database/migrations",
		"database/seeders",
		"routes",
		"storage/logs",
		"storage/cache",
		"tests/unit",
		"tests/integration",
		"docs",
		"scripts",
	}

	if config.UseQueue {
		dirs = append(dirs, "app/jobs", "storage/queue")
	}

	if config.UseWebSocket {
		dirs = append(dirs, "app/websocket")
	}

	if config.Template == "fullstack" {
		dirs = append(dirs, "public", "resources/views", "resources/assets")
	}

	for _, dir := range dirs {
		fullPath := fmt.Sprintf("%s/%s", projectName, dir)
		os.MkdirAll(fullPath, 0755)
		fmt.Printf("  + %s/\n", dir)
	}
}

func runGoModTidy(projectName string) error {
	command := exec.Command("go", "mod", "tidy")
	command.Dir = projectName
	output, err := command.CombinedOutput()
	if err != nil {
		return fmt.Errorf("install project dependencies: %w\n%s", err, output)
	}
	build := exec.Command("go", "build", "./...")
	build.Dir = projectName
	if output, err := build.CombinedOutput(); err != nil {
		return fmt.Errorf("validate generated project: %w\n%s", err, output)
	}
	fmt.Println("  + dependencies installed and project build verified")
	return nil
}

func printNextSteps(projectName string, config generators.ProjectConfig) {
	fmt.Printf("Next steps:\n\n")
	fmt.Printf("  1. cd %s\n\n", projectName)

	if config.UseDatabase {
		fmt.Printf("  2. Set up database:\n")
		fmt.Printf("     - Update .env with your database credentials\n")
		fmt.Printf("     - Run: routix migrate\n\n")
	}

	fmt.Printf("  3. Start development server:\n")
	fmt.Printf("     routix serve\n\n")

	fmt.Printf("Your API will be available at: http://localhost:8080\n")

	if config.UseSwagger {
		fmt.Printf("API docs: http://localhost:8080/docs\n")
	}

	fmt.Printf("\nUseful commands:\n")
	fmt.Printf("  routix make:controller UserController\n")
	fmt.Printf("  routix make:model User\n")
	fmt.Printf("  routix make:migration create_users_table\n")
	fmt.Printf("  routix route:list\n")
	fmt.Printf("  routix test\n")
}
