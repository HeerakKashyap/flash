package performance

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/goflash/flash/v2"
)

// BenchmarkHighPressure_SimpleHandler tests performance under high concurrency
func BenchmarkHighPressure_SimpleHandler(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	// Pre-create requests to avoid allocation overhead in benchmark
	numWorkers := runtime.NumCPU() * 4 // High concurrency
	requests := make([]*http.Request, numWorkers)
	for i := 0; i < numWorkers; i++ {
		requests[i] = httptest.NewRequest(http.MethodGet, "/ping", nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		workerID := 0
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, requests[workerID%numWorkers])
			workerID++
		}
	})
}

// BenchmarkHighPressure_JSONResponse tests JSON performance under high concurrency
func BenchmarkHighPressure_JSONResponse(b *testing.B) {
	app := flash.New()
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "hello world",
			"status":  "ok",
			"count":   42,
		})
	})

	numWorkers := runtime.NumCPU() * 4
	requests := make([]*http.Request, numWorkers)
	for i := 0; i < numWorkers; i++ {
		requests[i] = httptest.NewRequest(http.MethodGet, "/json", nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		workerID := 0
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, requests[workerID%numWorkers])
			workerID++
		}
	})
}

// BenchmarkHighPressure_MixedWorkload tests mixed operations under pressure
func BenchmarkHighPressure_MixedWorkload(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"id":   c.Query("id"),
			"data": "response",
		})
	})
	app.GET("/params/:id", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "id="+c.Param("id"))
	})

	// Mix of different request types
	requests := []*http.Request{
		httptest.NewRequest(http.MethodGet, "/ping", nil),
		httptest.NewRequest(http.MethodGet, "/json?id=123", nil),
		httptest.NewRequest(http.MethodGet, "/params/456", nil),
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		reqIdx := 0
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, requests[reqIdx%len(requests)])
			reqIdx++
		}
	})
}

// BenchmarkRPS_PureLoad tests maximum throughput
func BenchmarkRPS_PureLoad(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	// Warm up the pools
	for i := 0; i < 100; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Measure pure throughput
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkRPS_StressTest simulates realistic high RPS scenarios
func BenchmarkRPS_StressTest(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "hello",
			"id":      c.Query("id"),
		})
	})

	// Simulate high concurrency with realistic request patterns
	numWorkers := runtime.NumCPU() * 8
	requestsPerWorker := b.N / numWorkers
	if requestsPerWorker < 1 {
		requestsPerWorker = 1
	}

	var totalRequests int64
	var totalDuration int64

	b.ResetTimer()
	b.ReportAllocs()

	start := time.Now()
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < requestsPerWorker; j++ {
				reqStart := time.Now()

				rec := httptest.NewRecorder()
				var req *http.Request

				// Mix of different endpoints (realistic traffic pattern)
				if j%3 == 0 {
					req = httptest.NewRequest(http.MethodGet, "/json?id=123", nil)
				} else {
					req = httptest.NewRequest(http.MethodGet, "/ping", nil)
				}

				app.ServeHTTP(rec, req)

				atomic.AddInt64(&totalRequests, 1)
				atomic.AddInt64(&totalDuration, int64(time.Since(reqStart)))
			}
		}(i)
	}

	wg.Wait()
	totalTime := time.Since(start)

	if totalRequests > 0 {
		rps := float64(totalRequests) / totalTime.Seconds()
		avgLatency := time.Duration(totalDuration / totalRequests)

		b.ReportMetric(rps, "req/sec")
		b.ReportMetric(float64(avgLatency.Nanoseconds()), "avg-latency-ns")
		b.ReportMetric(float64(numWorkers), "workers")
	}
}

// BenchmarkMemoryPressure tests performance under memory pressure
func BenchmarkMemoryPressure(b *testing.B) {
	app := flash.New()
	app.GET("/memory", func(c flash.Ctx) error {
		// Create some memory pressure
		data := make([]byte, 1024) // 1KB allocation per request
		for i := range data {
			data[i] = byte(i % 256)
		}
		return c.JSON(map[string]interface{}{
			"size":     len(data),
			"checksum": int(data[0]) + int(data[len(data)-1]),
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/memory", nil)

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, req)
		}
	})
}

// BenchmarkConcurrentRoutes tests performance with many concurrent routes
func BenchmarkConcurrentRoutes(b *testing.B) {
	app := flash.New()

	// Register many routes to test routing performance
	for i := 0; i < 100; i++ {
		route := fmt.Sprintf("/route%d/:id", i)
		app.GET(route, func(c flash.Ctx) error {
			return c.JSON(map[string]string{
				"route": c.Route(),
				"id":    c.Param("id"),
			})
		})
	}

	// Test requests to different routes
	requests := make([]*http.Request, 10)
	for i := 0; i < 10; i++ {
		requests[i] = httptest.NewRequest(http.MethodGet,
			fmt.Sprintf("/route%d/test%d", i*10, i), nil)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		reqIdx := 0
		for pb.Next() {
			rec := httptest.NewRecorder()
			app.ServeHTTP(rec, requests[reqIdx%len(requests)])
			reqIdx++
		}
	})
}
