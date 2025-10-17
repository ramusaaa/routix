package commands

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

func ServeCommand(args []string) {
	port := "8080"
	host := "localhost"
	
	for i, arg := range args {
		if strings.HasPrefix(arg, "--port=") {
			port = strings.TrimPrefix(arg, "--port=")
		} else if arg == "--port" && i+1 < len(args) {
			port = args[i+1]
		} else if strings.HasPrefix(arg, "--host=") {
			host = strings.TrimPrefix(arg, "--host=")
		} else if arg == "--host" && i+1 < len(args) {
			host = args[i+1]
		}
	}

	fmt.Printf("🔥 Starting Routix development server...\n")
	fmt.Printf("🌐 Server will be available at: http://%s:%s\n", host, port)
	fmt.Printf("👀 Watching for file changes...\n\n")

	os.Setenv("APP_ENV", "development")
	os.Setenv("APP_PORT", port)
	os.Setenv("APP_HOST", host)

	startFileWatcher()
}

func startFileWatcher() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		fmt.Printf("❌ Error creating file watcher: %v\n", err)
		return
	}
	defer watcher.Close()

	watchDirs := []string{
		".",
		"app",
		"config",
		"routes",
	}

	for _, dir := range watchDirs {
		if _, err := os.Stat(dir); err == nil {
			addDirToWatcher(watcher, dir)
		}
	}

	var cmd *exec.Cmd
	restartServer := func() {
		if cmd != nil && cmd.Process != nil {
			fmt.Printf("🔄 Restarting server...\n")
			cmd.Process.Kill()
			cmd.Wait()
		}

		fmt.Printf("🚀 Starting server...\n")
		cmd = exec.Command("go", "run", "main.go")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Start()
	}

	restartServer()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		fmt.Printf("\n🛑 Shutting down server...\n")
		if cmd != nil && cmd.Process != nil {
			cmd.Process.Kill()
		}
		os.Exit(0)
	}()

	debounce := time.NewTimer(0)
	debounce.Stop()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if shouldRestart(event) {
				debounce.Reset(500 * time.Millisecond)
			}

		case <-debounce.C:
			restartServer()

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("⚠️  File watcher error: %v\n", err)
		}
	}
}

func addDirToWatcher(watcher *fsnotify.Watcher, dir string) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && !shouldIgnoreDir(path) {
			watcher.Add(path)
		}

		return nil
	})
}

func shouldIgnoreDir(path string) bool {
	ignoreDirs := []string{
		".git",
		"node_modules",
		"vendor",
		"storage/logs",
		"storage/cache",
		"tmp",
		"temp",
	}

	for _, ignore := range ignoreDirs {
		if strings.Contains(path, ignore) {
			return true
		}
	}

	return false
}

func shouldRestart(event fsnotify.Event) bool {
	if event.Op&fsnotify.Write == 0 {
		return false
	}

	ext := filepath.Ext(event.Name)
	restartExts := []string{".go", ".env", ".yaml", ".yml", ".json"}

	for _, restartExt := range restartExts {
		if ext == restartExt {
			fmt.Printf("📝 File changed: %s\n", event.Name)
			return true
		}
	}

	return false
}