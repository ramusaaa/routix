package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func InstallCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix install <package>")
		return
	}

	packageName := args[0]
	fmt.Printf("Installing package: %s\n", packageName)

	cmd := exec.Command("go", "get", packageName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Installation failed: %v\n", err)
		return
	}

	fmt.Printf("Package installed successfully!")

	cmd = exec.Command("go", "mod", "tidy")
	cmd.Run()
}
