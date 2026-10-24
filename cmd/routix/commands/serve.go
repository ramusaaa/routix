package commands

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

func ServeCommand(args []string) error {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	port := flags.Int("port", 8080, "HTTP port")
	host := flags.String("host", "localhost", "HTTP host")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 || *port < 1 || *port > 65535 || strings.TrimSpace(*host) == "" {
		return fmt.Errorf("usage: routix serve [--host=localhost] [--port=8080]; port must be between 1 and 65535")
	}

	fmt.Printf("Starting Routix development server...\n")
	fmt.Printf("http://%s\n", net.JoinHostPort(*host, strconv.Itoa(*port)))
	fmt.Printf("Watching for file changes...\n\n")

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	environment := append(os.Environ(), "APP_ENV=development", "APP_PORT="+strconv.Itoa(*port), "APP_HOST="+*host)
	return startFileWatcher(ctx, environment)
}

type developmentBuild struct {
	path string
	err  error
}

func startFileWatcher(ctx context.Context, environment []string) error {
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("find Go compiler: %w", err)
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create file watcher: %w", err)
	}
	defer watcher.Close()
	if err := addDirToWatcher(watcher, "."); err != nil {
		return fmt.Errorf("watch application: %w", err)
	}
	temporary, err := os.MkdirTemp("", "routix-serve-")
	if err != nil {
		return err
	}
	buildContext, cancelBuild := context.WithCancel(ctx)
	builds := make(chan developmentBuild, 1)
	building, pending := false, false
	generation := 0
	var process *exec.Cmd
	var exited chan error
	var runningPath string
	defer func() {
		cancelBuild()
		if building {
			<-builds
		}
		stopDevelopmentProcess(process, exited)
		os.RemoveAll(temporary)
	}()

	startBuild := func() {
		building = true
		generation++
		path := filepath.Join(temporary, fmt.Sprintf("application-%d.exe", generation))
		fmt.Println("Building application...")
		go func() {
			command := exec.CommandContext(buildContext, "go", "build", "-o", path, ".")
			command.Env = environment
			command.Stdout = os.Stdout
			command.Stderr = os.Stderr
			builds <- developmentBuild{path: path, err: command.Run()}
		}()
	}
	debounce := time.NewTimer(time.Hour)
	debounce.Stop()
	defer debounce.Stop()
	startBuild()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nShutting down...")
			return nil
		case result := <-builds:
			building = false
			if ctx.Err() != nil {
				return nil
			}
			if pending {
				pending = false
				os.Remove(result.path)
				startBuild()
				continue
			}
			if result.err != nil {
				os.Remove(result.path)
				fmt.Fprintf(os.Stderr, "Build failed: %v; waiting for changes\n", result.err)
				continue
			}
			stopDevelopmentProcess(process, exited)
			process = nil
			exited = nil
			if ctx.Err() != nil {
				return nil
			}
			if runningPath != "" {
				os.Remove(runningPath)
			}
			process = exec.Command(result.path)
			process.Env = environment
			process.Stdin = os.Stdin
			process.Stdout = os.Stdout
			process.Stderr = os.Stderr
			if err := process.Start(); err != nil {
				process = nil
				exited = nil
				return fmt.Errorf("start application: %w", err)
			}
			runningPath = result.path
			exited = make(chan error, 1)
			go func(command *exec.Cmd, done chan error) { done <- command.Wait() }(process, exited)
			fmt.Println("Application started")
		case err := <-exited:
			process = nil
			exited = nil
			if err != nil {
				fmt.Fprintf(os.Stderr, "Application exited: %v; waiting for changes\n", err)
			} else {
				fmt.Println("Application stopped; waiting for changes")
			}
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("file watcher closed")
			}
			if shouldIgnoreDir(event.Name) {
				continue
			}
			changed := shouldRestart(event)
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					if err := addDirToWatcher(watcher, event.Name); err != nil {
						return fmt.Errorf("watch new directory: %w", err)
					}
					changed = true
				}
			}
			if changed {
				debounce.Reset(500 * time.Millisecond)
			}
		case <-debounce.C:
			if building {
				pending = true
			} else {
				startBuild()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("file watcher error channel closed")
			}
			return fmt.Errorf("watch files: %w", err)
		}
	}
}

func stopDevelopmentProcess(command *exec.Cmd, exited <-chan error) {
	if command == nil || command.Process == nil {
		return
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		command.Process.Kill()
	}
	timer := time.NewTimer(12 * time.Second)
	defer timer.Stop()
	select {
	case <-exited:
	case <-timer.C:
		command.Process.Kill()
		<-exited
	}
}

func addDirToWatcher(watcher *fsnotify.Watcher, dir string) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if shouldIgnoreDir(path) {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

func shouldIgnoreDir(path string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	for i, part := range parts {
		switch part {
		case ".git", "node_modules", "vendor", "tmp", "temp":
			return true
		case "storage":
			if i+1 < len(parts) && (parts[i+1] == "logs" || parts[i+1] == "cache") {
				return true
			}
		}
	}
	return false
}

func shouldRestart(event fsnotify.Event) bool {
	if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 || shouldIgnoreDir(event.Name) {
		return false
	}
	if event.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		return true
	}
	name := filepath.Base(event.Name)
	if name == ".env" || strings.HasPrefix(name, ".env.") || name == "go.mod" || name == "go.sum" {
		return true
	}
	switch filepath.Ext(event.Name) {
	case ".go", ".yaml", ".yml", ".json", ".html", ".tmpl", ".tpl":
		return true
	}
	return false
}
