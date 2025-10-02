package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/logging"
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

	// 1. Load config and logger first.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}
	cfg := config.New()
	logging.New()

	// 2. Create all primary dependencies.
	// Errors are handled here at the top level of the application.
	db, err := database.NewDB(context.Background(), cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}

	emailer, err := email.NewEmailService(cfg)
	if err != nil {
		slog.Error("Failed to initialize email service", "error", err)
		os.Exit(1)
	}

	htmlHub := hub.NewHub()
	dataHub := hub.NewHub()

	// 3. Create the server by passing the option functions.
	s, err := server.New(
		server.WithConfig(cfg),
		server.WithDB(db, cfg.GetDBNs(), cfg.GetDBDb()),
		server.WithEmailer(emailer),
		server.WithHubs(htmlHub, dataHub),
	)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// Start the server.
	s.Start()
}
