package server

import (
	"context"
	"log"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/templates"
	"github.com/surrealdb/surrealdb.go"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	E   *echo.Echo
	DB  *surrealdb.DB
	Cfg *config.Config
}

// New creates a new Server instance.
func New() *Server {
	// Load environment variables from .env file if it exists.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}

	cfg := config.New()
	db, err := database.NewDB(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Serve static files from the "web/static" directory.
	e.Static("/static", "web/static")

	// Setup template renderer
	e.Renderer = templates.NewRenderer("web/src/templates")

	return &Server{E: e, DB: db, Cfg: cfg}
}
