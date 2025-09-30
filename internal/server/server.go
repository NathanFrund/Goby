package server

import (
	"context"
	"io/fs"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/joho/godotenv"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/logging"
	"github.com/nfrund/goby/internal/modules/data"
	"github.com/nfrund/goby/web"
	"github.com/surrealdb/surrealdb.go"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	E         *echo.Echo
	DB        *surrealdb.DB
	Cfg       config.Provider
	Emailer   domain.EmailSender
	UserStore domain.UserRepository

	homeHandler      *handlers.HomeHandler
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	aboutHandler     *handlers.AboutHandler

	htmlHub     *hub.Hub
	dataHub     *hub.Hub
	dataHandler *data.Handler
}

// New creates a new Server instance.
func New() *Server {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		// We don't have slog configured yet, so we use the standard logger here.
		// This is acceptable as it's only for the initial setup.
		log.Println("No .env file found, relying on environment variables")
	}

	cfg := config.New()
	logging.New() // Initialize the structured logger
	emailer, err := email.NewEmailService(cfg)
	if err != nil {
		slog.Error("Failed to initialize email service", "error", err)
		os.Exit(1)
	}

	// Create and run two separate hubs for our two channels.
	htmlHub := hub.NewHub()
	go htmlHub.Run()

	dataHub := hub.NewHub()
	go dataHub.Run()

	// Create stores and handlers, making them dependencies of the server.
	db, err := database.NewDB(context.Background(), cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	userStore := database.NewSurrealUserStore(db, cfg.GetDBNs(), cfg.GetDBDb())

	homeHandler := handlers.NewHomeHandler()
	dataHandler := data.NewHandler(dataHub)
	authHandler := handlers.NewAuthHandler(userStore, emailer, cfg.GetAppBaseURL())
	dashboardHandler := handlers.NewDashboardHandler()
	aboutHandler := &handlers.AboutHandler{}

	e := echo.New()
	e.Use(middleware.Recover())

	// Configure and use session middleware
	store := sessions.NewCookieStore([]byte(cfg.GetSessionSecret()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	e.Use(session.Middleware(store))

	// Serve static files from disk or embedded FS based on APP_STATIC.
	if os.Getenv("APP_STATIC") == "embed" {
		slog.Info("Serving embedded static assets")
		// Create a sub-filesystem that starts from the "static" directory
		// within our embedded assets.
		staticFS, err := fs.Sub(web.FS, "static")
		if err != nil {
			slog.Error("Failed to create sub-filesystem for embedded static assets", "error", err)
			os.Exit(1)
		}
		e.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))
	} else {
		slog.Info("Serving static assets from disk")
		e.Static("/static", "web/static")
	}

	s := &Server{
		E:                e,
		DB:               db,
		Cfg:              cfg,
		Emailer:          emailer,
		UserStore:        userStore,
		homeHandler:      homeHandler,
		authHandler:      authHandler,
		dashboardHandler: dashboardHandler,
		aboutHandler:     aboutHandler,
		htmlHub:          htmlHub,
		dataHub:          dataHub,
		dataHandler:      dataHandler,
	}

	// Register all application routes, including those from modules.
	s.RegisterRoutes()

	return s
}

// Start runs the HTTP server with graceful shutdown.
func (s *Server) Start() {
	addr := s.Cfg.GetServerAddr()

	// Start server in a goroutine so that it doesn't block.
	go func() {
		slog.Info("Starting server", "address", addr)
		if err := s.E.Start(addr); err != nil && err != http.ErrServerClosed {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for an interrupt signal to gracefully shut down the server.
	waitForShutdown()
	slog.Info("Shutting down server...")

	// The context is used to inform the server it has 10 seconds to finish
	// the requests it is currently handling.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Close the database connection.
	s.DB.Close(ctx)

	if err := s.E.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}
}
