package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func TestCommand(args []string) {
	testType := "all"
	verbose := false
	coverage := false

	for _, arg := range args {
		switch arg {
		case "unit":
			testType = "unit"
		case "integration":
			testType = "integration"
		case "--verbose", "-v":
			verbose = true
		case "--coverage", "-c":
			coverage = true
		}
	}

	runTests(testType, verbose, coverage)
}

func BuildCommand(args []string) {
	fmt.Printf("🏗️  Building application for production...\n")

	os.Setenv("APP_ENV", "production")

	buildArgs := []string{"build", "-ldflags", "-s -w", "-o", "app", "main.go"}
	
	cmd := exec.Command("go", buildArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Build failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Build completed successfully!\n")
	fmt.Printf("📦 Binary created: ./app\n")
	fmt.Printf("🚀 Run with: ./app\n")
}

func InstallCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix install <package>")
		return
	}

	packageName := args[0]
	fmt.Printf("📦 Installing package: %s\n", packageName)

	cmd := exec.Command("go", "get", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Installation failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Package installed successfully!\n")
	
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Run()
}

func runTests(testType string, verbose, coverage bool) {
	fmt.Printf("🧪 Running %s tests...\n", testType)

	var testPath string
	switch testType {
	case "unit":
		testPath = "./tests/unit/..."
	case "integration":
		testPath = "./tests/integration/..."
	default:
		testPath = "./tests/..."
	}

	if _, err := os.Stat("tests"); os.IsNotExist(err) {
		fmt.Printf("❌ Tests directory not found\n")
		fmt.Printf("💡 Run 'routix make:test ExampleTest' to create your first test\n")
		return
	}

	args := []string{"test"}
	
	if verbose {
		args = append(args, "-v")
	}
	
	if coverage {
		args = append(args, "-cover", "-coverprofile=coverage.out")
	}
	
	args = append(args, testPath)

	cmd := exec.Command("go", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("❌ Tests failed\n")
		return
	}

	fmt.Printf("✅ All tests passed!\n")

	if coverage {
		fmt.Printf("📊 Coverage report generated: coverage.out\n")
		fmt.Printf("💡 View coverage: go tool cover -html=coverage.out\n")
	}
}

func showTestHelp() {
	fmt.Println(`
Test Commands:
  routix test              Run all tests
  routix test unit         Run unit tests only
  routix test integration  Run integration tests only

Options:
  --verbose, -v           Verbose output
  --coverage, -c          Generate coverage report

Examples:
  routix test --verbose
  routix test unit --coverage
  routix test integration
`)
}