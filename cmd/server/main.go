package main

import (
	"os"

	"github.com/nfrund/goby/internal/server"
)

// AppTemplates can be set at build time to force a template loading strategy.
// Example: go build -ldflags "-X 'main.AppTemplates=embed'"
var AppTemplates string

func main() {
	if AppTemplates != "" {
		os.Setenv("APP_TEMPLATES", AppTemplates)
	}
	// Create a new server instance.
	s := server.New()

	// Register all application routes.
	s.RegisterRoutes()

	// Start the server.
	s.Start()
}
