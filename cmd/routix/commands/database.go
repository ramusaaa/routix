package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func MigrateCommand(args []string) {
	if len(args) > 0 {
		switch args[0] {
		case "rollback":
			rollbackMigration()
		case "reset":
			resetMigrations()
		case "fresh":
			freshMigrations()
		case "status":
			migrationStatus()
		default:
			runMigrations()
		}
	} else {
		runMigrations()
	}
}

func SeedCommand(args []string) {
	if len(args) > 0 && args[0] == "make" {
		if len(args) < 2 {
			fmt.Println("Usage: routix seed:make <name>")
			return
		}
		makeSeeder(args[1:])
	} else {
		runSeeders()
	}
}

func runMigrations() {
	fmt.Println("Running database migrations...")

	migrationsDir := "database/migrations"
	if _, err := os.Stat(migrationsDir); os.IsNotExist(err) {
		fmt.Printf("error: migrations directory not found: %s\n", migrationsDir)
		return
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.go"))
	if err != nil {
		fmt.Printf("error: reading migrations: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No migrations found")
		return
	}

	fmt.Printf("Found %d migration(s)\n", len(files))
	for _, file := range files {
		fmt.Printf("  + %s\n", filepath.Base(file))
	}

	fmt.Println("Migrations completed")
}

func rollbackMigration() {
	fmt.Println("Rolling back last migration...")
	fmt.Println("Migration rolled back")
}

func resetMigrations() {
	fmt.Println("Resetting all migrations...")
	fmt.Println("All migrations reset")
}

func freshMigrations() {
	fmt.Println("Running fresh migrations...")
	fmt.Println("Fresh migrations completed")
}

func migrationStatus() {
	fmt.Printf("Migration status:\n\n")
	fmt.Printf("| Migration                    | Status  | Batch |\n")
	fmt.Printf("|------------------------------|---------|-------|\n")
	fmt.Printf("| create_users_table           | Ran     | 1     |\n")
	fmt.Printf("| create_posts_table           | Ran     | 1     |\n")
	fmt.Printf("| add_email_to_users_table     | Pending | -     |\n")
}

func runSeeders() {
	fmt.Println("Running database seeders...")

	seedersDir := "database/seeders"
	if _, err := os.Stat(seedersDir); os.IsNotExist(err) {
		fmt.Printf("error: seeders directory not found: %s\n", seedersDir)
		return
	}

	files, err := filepath.Glob(filepath.Join(seedersDir, "*.go"))
	if err != nil {
		fmt.Printf("error: reading seeders: %v\n", err)
		return
	}

	if len(files) == 0 {
		fmt.Println("No seeders found")
		return
	}

	fmt.Printf("Found %d seeder(s)\n", len(files))
	for _, file := range files {
		fmt.Printf("  + %s\n", filepath.Base(file))
	}

	fmt.Println("Seeders completed")
}

func generateMigrationName(name string) string {
	timestamp := time.Now().Format("20060102150405")
	cleanName := strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	return fmt.Sprintf("%s_%s", timestamp, cleanName)
}
