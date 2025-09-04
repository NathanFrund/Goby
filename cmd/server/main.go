package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nfrund/goby/internal/templates"
)

func main() {
	e := echo.New()

	// Configure Echo to not use rate limiting
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Skipper: func(c echo.Context) bool {
			// Skip logging for health check endpoints
			return c.Path() == "/health"
		},
	}))
	e.Use(middleware.Recover())

	// Template rendering
	e.Renderer = templates.New()

	// Static files
	e.Static("/static", "static")

	// Add a simple health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.String(200, "OK")
	})

	// Routes
	e.GET("/", func(c echo.Context) error {
		return c.Render(200, "home", nil)
	})

	// Start server
	e.Logger.Info("Starting server on :8080")
	e.Logger.Fatal(e.Start(":8080"))
}
