package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/ramusaaa/routix"
)

func RouteCommand(args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("usage: routix route:list or routix route list")
	}
	if len(args) == 1 && args[0] != "list" {
		return fmt.Errorf("unsupported route action: %s", args[0])
	}
	return listRoutes()
}

func listRoutes() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	routes, err := inspectProjectRoutes(ctx, ".")
	if err != nil {
		return err
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "Registered Routes:\n\nMethod\tURI"); err != nil {
		return err
	}
	for _, route := range routes {
		if _, err := fmt.Fprintf(writer, "%s\t%s\n", route.Method, route.Path); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(writer, "\nTotal routes: %d\n", len(routes)); err != nil {
		return err
	}
	return writer.Flush()
}

func inspectProjectRoutes(ctx context.Context, directory string) ([]routix.RouteInfo, error) {
	command := exec.CommandContext(ctx, "go", "list", "-f", "{{.Name}}", ".")
	command.Dir = directory
	var diagnostics bytes.Buffer
	command.Stderr = &diagnostics
	output, err := command.Output()
	if err != nil {
		return nil, fmt.Errorf("load application: %w\n%s", err, diagnostics.String())
	}
	if strings.TrimSpace(string(output)) != "main" {
		return nil, fmt.Errorf("route:list must run in an application main package")
	}
	command = exec.CommandContext(ctx, "go", "doc", "-short", "github.com/ramusaaa/routix.Router.WriteRoutes")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("application Routix version does not support route:list; update github.com/ramusaaa/routix: %w\n%s", err, output)
	}
	temporary, err := os.MkdirTemp("", "routix-route-list-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temporary)
	target := filepath.Join(temporary, "routes.json")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		return nil, err
	}
	binary := filepath.Join(temporary, "application.exe")
	command = exec.CommandContext(ctx, "go", "build", "-o", binary, ".")
	command.Dir = directory
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build application for route inspection: %w\n%s", err, output)
	}
	command = exec.CommandContext(ctx, binary)
	command.Dir = directory
	command.Env = append(os.Environ(), routix.RouteListFileEnv+"="+target)
	if output, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("initialize application routes: %w\n%s", err, output)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, err
	}
	var routes []routix.RouteInfo
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, fmt.Errorf("application did not export routes; use Router.Start, APIBuilder.Start or APIBuilder.StartTLS: %w", err)
	}
	return routes, nil
}
