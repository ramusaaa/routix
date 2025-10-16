package routix

import (
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
	RequestCount    int64
	TotalLatency    time.Duration
	MinLatency      time.Duration
	MaxLatency      time.Duration
	ErrorCount      int64
	ActiveRequests  int64
	mu              sync.RWMutex
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
	
	if latency < m.MinLatency {
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
	if m.RequestCount > 0 {
		avgLatency = m.TotalLatency / time.Duration(m.RequestCount)
	}
	
	return map[string]interface{}{
		"total_requests":     m.RequestCount,
		"error_count":        m.ErrorCount,
		"active_requests":    m.ActiveRequests,
		"average_latency_ms": avgLatency.Milliseconds(),
		"min_latency_ms":     m.MinLatency.Milliseconds(),
		"max_latency_ms":     m.MaxLatency.Milliseconds(),
		"error_rate":         float64(m.ErrorCount) / float64(m.RequestCount) * 100,
	}
}

// Performance monitoring middleware
func PerformanceMonitor() Middleware {
	return func(next Handler) Handler {
		return func(c *Context) error {
			start := time.Now()
			
			// Increment active requests
			globalMetrics.mu.Lock()
			globalMetrics.ActiveRequests++
			globalMetrics.mu.Unlock()
			
			// Execute handler
			err := next(c)
			
			// Calculate latency and update metrics
			latency := time.Since(start)
			globalMetrics.UpdateMetrics(latency, err != nil)
			
			// Decrement active requests
			globalMetrics.mu.Lock()
			globalMetrics.ActiveRequests--
			globalMetrics.mu.Unlock()
			
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
	var wg sync.WaitGroup
	var totalRequests int64
	var totalLatency time.Duration
	var errors int64
	var mu sync.Mutex
	
	start := time.Now()
	done := make(chan bool)
	
	// Start concurrent workers
	for i := 0; i < concurrent; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					reqStart := time.Now()
					
					// Simulate request (in real implementation, use httptest)
					// This is a simplified version
					time.Sleep(time.Microsecond * 100) // Simulate processing
					
					reqLatency := time.Since(reqStart)
					
					mu.Lock()
					totalRequests++
					totalLatency += reqLatency
					mu.Unlock()
				}
			}
		}()
	}
	
	// Stop after duration
	time.Sleep(duration)
	close(done)
	wg.Wait()
	
	totalTime := time.Since(start)
	
	return &BenchmarkResult{
		RequestsPerSecond: float64(totalRequests) / totalTime.Seconds(),
		AverageLatency:    totalLatency / time.Duration(totalRequests),
		TotalRequests:     int(totalRequests),
		TotalTime:         totalTime,
		ErrorRate:         float64(errors) / float64(totalRequests) * 100,
	}
}