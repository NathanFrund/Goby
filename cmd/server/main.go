package main

import (
	"github.com/nfrund/goby/internal/server"
)

func main() {
	// Create a new server instance.
	s := server.New()

	// Register all application routes.
	s.RegisterRoutes()

	// Start the server.
	s.Start(":8080")
}
