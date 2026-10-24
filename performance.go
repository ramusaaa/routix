package routix

import (
	"net/http/httptest"
	"runtime"
	"sync"
	"time"
)

// Pool for reusing contexts
var contextPool = sync.Pool{
	New: func() interface{} {
		return &Context{
			Params: make(map[string]string),
			Query:  make(map[string]string),
		}
	},
}

// getContext gets a context from the pool
func getContext() *Context {
	return contextPool.Get().(*Context)
}

// putContext returns a context to the pool
func putContext(c *Context) {
	// Clear the context
	for k := range c.Params {
		delete(c.Params, k)
	}
	for k := range c.Query {
		delete(c.Query, k)
	}
	c.Request = nil
	c.Response = nil
	c.Body = nil

	contextPool.Put(c)
}

// FastRouter is an optimized version of Router for high-performance scenarios
type FastRouter struct {
	*Router
	enablePooling bool
}

// NewFastRouter creates a new high-performance router
func NewFastRouter() *FastRouter {
	return &FastRouter{
		Router:        New(),
		enablePooling: true,
	}
}

// DisablePooling disables context pooling (useful for debugging)
func (fr *FastRouter) DisablePooling() *FastRouter {
	fr.enablePooling = false
	return fr
}

// Performance metrics
type Metrics struct {
	RequestCount   int64
	TotalLatency   time.Duration
	MinLatency     time.Duration
	MaxLatency     time.Duration
	ErrorCount     int64
	ActiveRequests int64
	mu             sync.RWMutex
}

// Global metrics instance
var globalMetrics = &Metrics{
	MinLatency: time.Hour, // Initialize with high value
}

// GetGlobalMetrics returns global performance metrics
func GetGlobalMetrics() map[string]interface{} {
	return globalMetrics.GetMetrics()
}

// UpdateMetrics updates performance metrics
func (m *Metrics) UpdateMetrics(latency time.Duration, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RequestCount++
	m.TotalLatency += latency

	if m.RequestCount == 1 || latency < m.MinLatency {
		m.MinLatency = latency
	}
	if latency > m.MaxLatency {
		m.MaxLatency = latency
	}

	if isError {
		m.ErrorCount++
	}
}

// GetMetrics returns current performance metrics
func (m *Metrics) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	minLatency := time.Duration(0)
	errorRate := float64(0)
	if m.RequestCount > 0 {
		avgLatency = m.TotalLatency / time.Duration(m.RequestCount)
		minLatency = m.MinLatency
		errorRate = float64(m.ErrorCount) / float64(m.RequestCount) * 100
	}

	return map[string]interface{}{
		"total_requests":     m.RequestCount,
		"error_count":        m.ErrorCount,
		"active_requests":    m.ActiveRequests,
		"average_latency_ms": avgLatency.Milliseconds(),
		"min_latency_ms":     minLatency.Milliseconds(),
		"max_latency_ms":     m.MaxLatency.Milliseconds(),
		"error_rate":         errorRate,
	}
}

// Performance monitoring middleware
func PerformanceMonitor() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) (err error) {
			start := time.Now()
			completed := false
			globalMetrics.mu.Lock()
			globalMetrics.ActiveRequests++
			globalMetrics.mu.Unlock()
			defer func() {
				globalMetrics.UpdateMetrics(time.Since(start), !completed || err != nil || c.Status() >= 400)
				globalMetrics.mu.Lock()
				globalMetrics.ActiveRequests--
				globalMetrics.mu.Unlock()
			}()
			err = next(c)
			completed = true
			return err
		}
	}
}

// Benchmark utilities
type BenchmarkResult struct {
	RequestsPerSecond float64
	AverageLatency    time.Duration
	TotalRequests     int
	TotalTime         time.Duration
	ErrorRate         float64
	MemoryUsage       int64
}

// LoadTest performs a simple load test
func LoadTest(router *Router, method, path string, concurrent int, duration time.Duration) *BenchmarkResult {
	if router == nil || concurrent <= 0 || duration <= 0 {
		panic("routix: load test requires a router, positive concurrency and duration")
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	start := time.Now()
	deadline := start.Add(duration)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalRequests, failed int
	var totalLatency time.Duration
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count, errors := 0, 0
			var latency time.Duration
			for time.Now().Before(deadline) {
				req := httptest.NewRequest(method, path, nil)
				w := httptest.NewRecorder()
				began := time.Now()
				router.ServeHTTP(w, req)
				latency += time.Since(began)
				count++
				if w.Code >= 400 {
					errors++
				}
			}
			mu.Lock()
			totalRequests += count
			failed += errors
			totalLatency += latency
			mu.Unlock()
		}()
	}
	wg.Wait()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&after)
	result := &BenchmarkResult{TotalRequests: totalRequests, TotalTime: elapsed, MemoryUsage: int64(after.TotalAlloc - before.TotalAlloc)}
	if totalRequests > 0 {
		result.RequestsPerSecond = float64(totalRequests) / elapsed.Seconds()
		result.AverageLatency = totalLatency / time.Duration(totalRequests)
		result.ErrorRate = float64(failed) / float64(totalRequests) * 100
	}
	return result
}
