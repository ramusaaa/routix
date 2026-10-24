package routix

import (
	"context"
	"net/http"
	"sync"
	"time"
)

type ReadinessCheck func(context.Context) error

type readinessResult struct {
	done chan struct{}
	err  error
}

func Readiness(timeout time.Duration, checks map[string]ReadinessCheck) Handler {
	if timeout <= 0 {
		panic("routix: readiness timeout must be positive")
	}
	copied := make(map[string]ReadinessCheck, len(checks))
	for name, check := range checks {
		if name == "" || check == nil {
			panic("routix: readiness requires named checks")
		}
		copied[name] = check
	}
	var mu sync.Mutex
	running := map[string]*readinessResult{}
	return func(c *Context) error {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		tasks := make(map[string]*readinessResult, len(copied))
		for name, check := range copied {
			mu.Lock()
			task := running[name]
			if task == nil {
				task = &readinessResult{done: make(chan struct{})}
				running[name] = task
				go func(name string, check ReadinessCheck, task *readinessResult) {
					checkContext, cancelCheck := context.WithTimeout(context.Background(), timeout)
					var err error
					defer func() {
						if recover() != nil {
							err = context.Canceled
						}
						cancelCheck()
						mu.Lock()
						task.err = err
						close(task.done)
						delete(running, name)
						mu.Unlock()
					}()
					err = check(checkContext)
				}(name, check, task)
			}
			mu.Unlock()
			tasks[name] = task
		}
		ready := true
		states := make(map[string]string, len(tasks))
		for name, task := range tasks {
			select {
			case <-task.done:
				states[name] = "ok"
				if task.err != nil {
					ready = false
					states[name] = "failed"
				}
			case <-ctx.Done():
				ready = false
				states[name] = "timeout"
			}
		}
		if ctx.Err() != nil {
			ready = false
		}
		c.SetHeader("Cache-Control", "no-store")
		status, state := http.StatusOK, "ready"
		if !ready {
			status, state = http.StatusServiceUnavailable, "not_ready"
		}
		return c.JSON(status, map[string]any{"status": state, "checks": states})
	}
}

func (api *APIBuilder) Ready(path string, timeout time.Duration, checks map[string]ReadinessCheck) *APIBuilder {
	if path == "" {
		path = "/ready"
	}
	api.router.GET(path, Readiness(timeout, checks))
	return api
}
