package main

import (
	"net/http"

	"github.com/goflash/flash/v2"
)

func main() {
	app := flash.New()

	// Basic routes for testing
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

	app.GET("/users/:id", func(c flash.Ctx) error {
		return c.JSON(map[string]string{
			"id":   c.Param("id"),
			"name": "John Doe",
		})
	})

	app.GET("/search", func(c flash.Ctx) error {
		q := c.Query("q")
		limit := c.QueryInt("limit", 10)
		return c.JSON(map[string]interface{}{
			"query":   q,
			"limit":   limit,
			"results": []string{"result1", "result2", "result3"},
		})
	})

	println("🚀 Server starting on :8080")
	println("📊 Test endpoints:")
	println("   GET /ping")
	println("   GET /json")
	println("   GET /users/123")
	println("   GET /search?q=test&limit=5")

	if err := http.ListenAndServe(":8080", app); err != nil {
		panic(err)
	}
}
