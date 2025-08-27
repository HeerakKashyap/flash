package main

import (
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/goflash/flash/v2"
	"github.com/valyala/fasthttp"
)

func main() {
	// Create a Flash app
	app := flash.New()

	// Add some simple routes for benchmarking
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

	app.GET("/health", func(c flash.Ctx) error {
		return c.JSON(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	log.Printf("Starting servers with %d CPU cores", runtime.NumCPU())

	// Start FastHTTP server in a goroutine
	go func() {
		log.Println("FastHTTP server starting on :8081")
		if err := fasthttp.ListenAndServe(":8081", app.(*flash.DefaultApp).ServeFastHTTP); err != nil {
			log.Fatalf("FastHTTP server failed: %v", err)
		}
	}()

	// Start net/http server
	log.Println("net/http server starting on :8080")
	if err := http.ListenAndServe(":8080", app); err != nil {
		log.Fatalf("net/http server failed: %v", err)
	}
}
