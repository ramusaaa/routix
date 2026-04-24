package commands

import (
	"fmt"
	"os"
	"os/exec"
)

func InstallCommand(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: routix install <package[@version]>")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  routix install github.com/some/package")
		fmt.Println("  routix install github.com/some/package@v1.2.3")
		return
	}

	pkg := args[0]
	fmt.Printf("Installing %s...\n", pkg)

	// Use 'go get' inside a module to add the dependency.
	get := exec.Command("go", "get", pkg)
	get.Stdout = os.Stdout
	get.Stderr = os.Stderr
	if err := get.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	// Tidy after adding so go.sum stays consistent.
	tidy := exec.Command("go", "mod", "tidy")
	tidy.Stdout = os.Stdout
	tidy.Stderr = os.Stderr
	if err := tidy.Run(); err != nil {
		fmt.Printf("warning: go mod tidy failed: %v\n", err)
		return
	}

	fmt.Printf("installed: %s\n", pkg)
}
