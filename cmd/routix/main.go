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
		if err := commands.NewProject(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
		if err := commands.ServeCommand(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case command == "build":
		if err := commands.BuildCommand(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case command == "migrate":
		if err := commands.MigrateCommand(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case command == "seed":
		if err := commands.SeedCommand(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case command == "route" || command == "route:list":
		routeArgs := args
		if command == "route:list" {
			routeArgs = append([]string{"list"}, args...)
		}
		if err := commands.RouteCommand(routeArgs); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case command == "test":
		if err := commands.TestCommand(args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
