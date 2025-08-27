package performance

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/goflash/flash/v2"
)

// BenchmarkBaseline_SimpleHandler tests basic string response performance
func BenchmarkBaseline_SimpleHandler(b *testing.B) {
	app := flash.New()
	app.GET("/ping", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "pong")
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_JSONResponse tests JSON serialization performance
func BenchmarkBaseline_JSONResponse(b *testing.B) {
	app := flash.New()
	app.GET("/json", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"message": "hello world",
			"status":  "ok",
			"count":   42,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/json", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_PathParams tests path parameter extraction performance
func BenchmarkBaseline_PathParams(b *testing.B) {
	app := flash.New()
	app.GET("/users/:id/posts/:postId", func(c flash.Ctx) error {
		id := c.Param("id")
		postId := c.Param("postId")
		return c.JSON(map[string]string{
			"userId": id,
			"postId": postId,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/users/123/posts/456", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_QueryParams tests query parameter parsing performance
func BenchmarkBaseline_QueryParams(b *testing.B) {
	app := flash.New()
	app.GET("/search", func(c flash.Ctx) error {
		q := c.Query("q")
		limit := c.QueryInt("limit", 10)
		offset := c.QueryInt("offset", 0)
		return c.JSON(map[string]interface{}{
			"query":  q,
			"limit":  limit,
			"offset": offset,
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/search?q=flash&limit=20&offset=10", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_JSONBinding tests request body binding performance
func BenchmarkBaseline_JSONBinding(b *testing.B) {
	app := flash.New()
	app.POST("/users", func(c flash.Ctx) error {
		var user struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Age   int    `json:"age"`
		}
		if err := c.BindJSON(&user); err != nil {
			return err
		}
		return c.JSON(user)
	})

	body := `{"name":"John Doe","email":"john@example.com","age":30}`

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_Middleware tests middleware overhead
func BenchmarkBaseline_Middleware(b *testing.B) {
	app := flash.New()

	// Add some middleware
	app.Use(func(next flash.Handler) flash.Handler {
		return func(c flash.Ctx) error {
			c.Header("X-Custom", "value")
			return next(c)
		}
	})

	app.Use(func(next flash.Handler) flash.Handler {
		return func(c flash.Ctx) error {
			// Simulate some processing
			_ = c.Query("test")
			return next(c)
		}
	})

	app.GET("/middleware", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "middleware test")
	})

	req := httptest.NewRequest(http.MethodGet, "/middleware", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_ErrorHandling tests error handling performance
func BenchmarkBaseline_ErrorHandling(b *testing.B) {
	app := flash.New()
	app.GET("/error", func(c flash.Ctx) error {
		return c.String(http.StatusInternalServerError, "error occurred")
	})

	req := httptest.NewRequest(http.MethodGet, "/error", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_LargeJSON tests performance with larger JSON payloads
func BenchmarkBaseline_LargeJSON(b *testing.B) {
	app := flash.New()
	app.GET("/large", func(c flash.Ctx) error {
		// Create a larger response
		data := make(map[string]interface{})
		for i := 0; i < 100; i++ {
			data[fmt.Sprintf("field_%d", i)] = fmt.Sprintf("value_%d", i)
		}
		return c.JSON(data)
	})

	req := httptest.NewRequest(http.MethodGet, "/large", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_OptimizedMiddleware tests optimized middleware performance
func BenchmarkBaseline_OptimizedMiddleware(b *testing.B) {
	app := flash.New()

	// Add middleware using regular API (automatically optimized internally)
	app.Use(func(next flash.Handler) flash.Handler {
		return func(c flash.Ctx) error {
			c.Header("X-Custom", "value")
			return next(c)
		}
	})

	app.Use(func(next flash.Handler) flash.Handler {
		return func(c flash.Ctx) error {
			// Simulate some processing
			_ = c.Query("test")
			return next(c)
		}
	})

	app.GET("/optimizedmiddleware", func(c flash.Ctx) error {
		return c.String(http.StatusOK, "optimized middleware test")
	})

	req := httptest.NewRequest(http.MethodGet, "/optimizedmiddleware", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_HeaderHandling tests header setting performance
func BenchmarkBaseline_HeaderHandling(b *testing.B) {
	app := flash.New()
	app.GET("/headers", func(c flash.Ctx) error {
		// Set multiple headers using regular method
		c.Header("X-Custom-1", "value1")
		c.Header("X-Custom-2", "value2")
		c.Header("Cache-Control", "max-age=3600")
		c.Header("Content-Type", "application/json")
		return c.JSON(map[string]string{"message": "headers test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/headers", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_OptimizedHeaderHandling tests optimized header setting performance
func BenchmarkBaseline_OptimizedHeaderHandling(b *testing.B) {
	app := flash.New()
	app.GET("/optimizedheaders", func(c flash.Ctx) error {
		// Set multiple headers using optimized methods (same API, faster implementation)
		c.Header("X-Custom-1", "value1")
		c.Header("X-Custom-2", "value2")
		c.SetMaxAge(3600)      // Optimized cache control
		c.SetContentTypeJSON() // Optimized content type
		return c.JSON(map[string]string{"message": "optimized headers test"})
	})

	req := httptest.NewRequest(http.MethodGet, "/optimizedheaders", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}

// BenchmarkBaseline_NetHTTPCompatibility tests net/http handler compatibility
func BenchmarkBaseline_NetHTTPCompatibility(b *testing.B) {
	app := flash.New()

	// Mount a standard net/http handler - should work seamlessly
	app.HandleHTTP("GET", "/nethttp", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("net/http handler working"))
	}))

	// Also test that the Flash app itself implements http.Handler
	var _ http.Handler = app // Compile-time check

	req := httptest.NewRequest(http.MethodGet, "/nethttp", nil)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		app.ServeHTTP(rec, req)
	}
}
