package main

import (
	"os"

	"github.com/nfrund/goby/internal/server"
)

// AppTemplates can be set at build time to force a template loading strategy.
// Example: go build -ldflags "-X 'main.AppTemplates=embed'"
var AppTemplates string

// AppStatic can be set at build time to force an asset loading strategy.
// Example: go build -ldflags "-X 'main.AppStatic=embed'"
var AppStatic string

func main() {
	if AppTemplates != "" {
		os.Setenv("APP_TEMPLATES", AppTemplates)
	}
	if AppStatic != "" {
		os.Setenv("APP_STATIC", AppStatic)
	}
	// Create a new server instance.
	s := server.New()

	// Register all application routes.
	s.RegisterRoutes()

	// Start the server.
	s.Start()
}
