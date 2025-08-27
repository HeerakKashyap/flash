package performance

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/goflash/flash/v2"
	"github.com/valyala/fasthttp"
)

// BenchmarkFastHTTP_SimpleHandler tests ultra-fast simple handler performance using fasthttp
func BenchmarkFastHTTP_SimpleHandler(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	// Get the fasthttp handler
	handler := app.(*flash.DefaultApp).ServeFastHTTP

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/ping")
		ctx.Request.Header.SetMethod("GET")

		for pb.Next() {
			// Reset context for each request
			ctx.Response.Reset()
			ctx.Request.SetRequestURI("/ping")
			ctx.Request.Header.SetMethod("GET")

			handler(ctx)
		}
	})
}

// BenchmarkFastHTTP_JSONResponse tests JSON response performance using fasthttp
func BenchmarkFastHTTP_JSONResponse(b *testing.B) {
	app := flash.New()
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "hello world",
			"status":  "ok",
			"count":   42,
		})
	})

	// Get the fasthttp handler
	handler := app.(*flash.DefaultApp).ServeFastHTTP

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		ctx := &fasthttp.RequestCtx{}

		for pb.Next() {
			// Reset context for each request
			ctx.Response.Reset()
			ctx.Request.SetRequestURI("/json")
			ctx.Request.Header.SetMethod("GET")

			handler(ctx)
		}
	})
}

// BenchmarkComparison_NetHTTP_vs_FastHTTP compares both transports side by side
func BenchmarkComparison_NetHTTP_vs_FastHTTP(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	b.Run("NetHTTP", func(b *testing.B) {
		req := &http.Request{}
		req.Method = "GET"
		req.URL = &url.URL{Path: "/ping"}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			w := &mockResponseWriter{}
			app.ServeHTTP(w, req)
		}
	})

	b.Run("FastHTTP", func(b *testing.B) {
		handler := app.(*flash.DefaultApp).ServeFastHTTP

		b.ResetTimer()
		b.ReportAllocs()

		b.RunParallel(func(pb *testing.PB) {
			ctx := &fasthttp.RequestCtx{}

			for pb.Next() {
				ctx.Response.Reset()
				ctx.Request.SetRequestURI("/ping")
				ctx.Request.Header.SetMethod("GET")

				handler(ctx)
			}
		})
	})
}

// mockResponseWriter is a minimal response writer for benchmarking
type mockResponseWriter struct {
	headers http.Header
	status  int
	written int
}

func (w *mockResponseWriter) Header() http.Header {
	if w.headers == nil {
		w.headers = make(http.Header)
	}
	return w.headers
}

func (w *mockResponseWriter) Write(data []byte) (int, error) {
	w.written += len(data)
	return len(data), nil
}

func (w *mockResponseWriter) WriteHeader(status int) {
	w.status = status
}

// BenchmarkFastHTTP_HighPressure tests performance under high concurrency
func BenchmarkFastHTTP_HighPressure(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "hello world",
			"status":  "ok",
			"count":   42,
		})
	})

	handler := app.(*flash.DefaultApp).ServeFastHTTP

	b.ResetTimer()
	b.ReportAllocs()

	// Use higher parallelism
	b.SetParallelism(100)

	b.RunParallel(func(pb *testing.PB) {
		ctx := &fasthttp.RequestCtx{}
		paths := []string{"/ping", "/json"}
		pathIndex := 0

		for pb.Next() {
			ctx.Response.Reset()
			ctx.Request.SetRequestURI(paths[pathIndex%len(paths)])
			ctx.Request.Header.SetMethod("GET")

			handler(ctx)

			pathIndex++
		}
	})
}
