package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ramusaaa/routix/cmd/routix/commands"
)

func main() {
	if len(os.Args) < 2 {
		commands.ShowHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch {
	case command == "new" || command == "create":
		commands.NewProject(args)
	case command == "make" || strings.HasPrefix(command, "make:"):
		if strings.HasPrefix(command, "make:") {
			// Handle make:controller, make:model, etc.
			makeType := strings.TrimPrefix(command, "make:")
			makeArgs := append([]string{makeType}, args...)
			commands.MakeCommand(makeArgs)
		} else {
			commands.MakeCommand(args)
		}
	case command == "serve" || command == "dev":
		commands.ServeCommand(args)
	case command == "build":
		commands.BuildCommand(args)
	case command == "migrate":
		commands.MigrateCommand(args)
	case command == "seed":
		commands.SeedCommand(args)
	case command == "route":
		commands.RouteCommand(args)
	case command == "test":
		commands.TestCommand(args)
	case command == "install":
		commands.InstallCommand(args)
	case command == "version" || command == "--version" || command == "-v":
		commands.ShowVersion()
	case command == "help" || command == "--help" || command == "-h":
		commands.ShowHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		commands.ShowHelp()
	}
}