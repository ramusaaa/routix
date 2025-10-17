package main

import (
	"fmt"
	"os"

	"github.com/ramusaaa/routix/cmd/routix/commands"
)

func main() {
	if len(os.Args) < 2 {
		commands.ShowHelp()
		return
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "new", "create":
		commands.NewProject(args)
	case "make":
		commands.MakeCommand(args)
	case "serve", "dev":
		commands.ServeCommand(args)
	case "build":
		commands.BuildCommand(args)
	case "migrate":
		commands.MigrateCommand(args)
	case "seed":
		commands.SeedCommand(args)
	case "route":
		commands.RouteCommand(args)
	case "test":
		commands.TestCommand(args)
	case "install":
		commands.InstallCommand(args)
	case "version", "--version", "-v":
		commands.ShowVersion()
	case "help", "--help", "-h":
		commands.ShowHelp()
	default:
		fmt.Printf("Unknown command: %s\n", command)
		commands.ShowHelp()
	}
}