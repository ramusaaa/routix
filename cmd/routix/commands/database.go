package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ramusaaa/routix/cmd/routix/generators"
)

func MigrateCommand(args []string) error {
	if len(args) > 0 && args[0] == "repair" {
		if len(args) != 3 || args[1] == "" || (args[2] != "--applied" && args[2] != "--pending") {
			return fmt.Errorf("usage: routix migrate repair <id> --applied|--pending")
		}
		return runDatabaseCommand("repair", args[1:]...)
	}
	action := "migrate"
	if len(args) > 1 {
		return fmt.Errorf("usage: routix migrate [rollback|reset|fresh|status]")
	}
	if len(args) == 1 {
		switch args[0] {
		case "rollback", "reset", "fresh", "status":
			action = args[0]
		default:
			return fmt.Errorf("unknown migration action: %s", args[0])
		}
	}
	return runDatabaseCommand(action)
}

func SeedCommand(args []string) error {
	if len(args) > 0 && args[0] == "make" {
		if len(args) != 2 {
			return fmt.Errorf("usage: routix seed make <name>")
		}
		return generators.GenerateSeeder(args[1], nil)
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: routix seed")
	}
	return runDatabaseCommand("seed")
}

func runDatabaseCommand(action string, args ...string) error {
	if _, err := os.Stat(filepath.Join("database", "connection.go")); err != nil {
		return fmt.Errorf("database support not found: %w", err)
	}
	command := exec.Command("go", "list", "-m", "-f", "{{.Path}}")
	command.Stderr = os.Stderr
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("read project module: %w", err)
	}
	module := strings.TrimSpace(string(output))
	if module == "" || strings.ContainsAny(module, "\"\\\n\r") {
		return fmt.Errorf("invalid project module")
	}
	directory, err := os.MkdirTemp(".", ".routix-database-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)
	source := generators.GenerateDatabaseRunner(module)
	if err := os.WriteFile(filepath.Join(directory, "main.go"), []byte(source), 0600); err != nil {
		return err
	}
	command = exec.Command("go", append([]string{"run", "./" + filepath.Base(directory), action}, args...)...)
	command.Stdin = os.Stdin
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("database %s failed: %w", action, err)
	}
	if action != "status" {
		fmt.Printf("Database %s completed\n", action)
	}
	return nil
}

func generateMigrationName(name string) string {
	timestamp := time.Now().UTC().Format("20060102150405")
	cleanName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	return fmt.Sprintf("%s_%s", timestamp, cleanName)
}
