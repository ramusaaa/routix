package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func TestCommand(args []string) error {
	testType := "all"
	verbose := false
	coverage := false

	selected := false
	for _, arg := range args {
		switch arg {
		case "all", "unit", "integration":
			if selected && testType != arg {
				return fmt.Errorf("choose one test scope: all, unit or integration")
			}
			testType = arg
			selected = true
		case "--verbose", "-v":
			verbose = true
		case "--coverage", "-c":
			coverage = true
		case "--help", "-h":
			showTestHelp()
			return nil
		default:
			return fmt.Errorf("unknown test argument: %s", arg)
		}
	}

	return runTests(testType, verbose, coverage)
}

func runTests(testType string, verbose, coverage bool) error {
	fmt.Printf("Running %s tests...\n", testType)

	var testPath string
	switch testType {
	case "unit":
		testPath = "./tests/unit/..."
	case "integration":
		testPath = "./tests/integration/..."
	default:
		testPath = "./..."
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
		return fmt.Errorf("run %s tests: %w", testType, err)
	}

	fmt.Printf("All tests passed!\n")

	if coverage {
		fmt.Printf("Coverage report generated: coverage.out\n")
		fmt.Printf("View coverage: go tool cover -html=coverage.out\n")
	}
	return nil
}

func showTestHelp() {
	fmt.Print(`
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
` + "\n")
}
