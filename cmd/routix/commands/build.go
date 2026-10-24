package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func BuildCommand(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: routix build")
	}
	fmt.Printf("Building application for production...\n")

	buildArgs := []string{"build", "-ldflags", "-s -w", "-o", "app", "."}

	cmd := exec.Command("go", buildArgs...)
	cmd.Env = append(os.Environ(), "APP_ENV=production")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("build application: %w", err)
	}

	fmt.Printf("Build completed successfully!\n")
	fmt.Printf("Binary created: ./app\n")
	fmt.Printf("Run with: ./app\n")
	return nil
}
