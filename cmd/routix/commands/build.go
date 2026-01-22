package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func BuildCommand(args []string) {
	fmt.Printf("Building application for production...\n")

	os.Setenv("APP_ENV", "production")

	buildArgs := []string{"build", "-ldflags", "-s -w", "-o", "app", "main.go"}

	cmd := exec.Command("go", buildArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		return
	}

	fmt.Printf("Build completed successfully!\n")
	fmt.Printf("Binary created: ./app\n")
	fmt.Printf("Run with: ./app\n")
}
