package server

import (
	"context"
	"log"
	"log/slog"
	"os"

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
	"github.com/nfrund/goby/internal/modules/chat"
	"github.com/nfrund/goby/internal/modules/data"
	"github.com/nfrund/goby/internal/modules/wargame"
	"github.com/nfrund/goby/internal/templates"
	"github.com/surrealdb/surrealdb.go"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	E                *echo.Echo
	DB               *surrealdb.DB
	Cfg              config.Provider
	Emailer          domain.EmailSender
	userStore        domain.UserRepository
	homeHandler      *handlers.HomeHandler
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	htmlHub          *hub.Hub
	dataHub          *hub.Hub
	chatHandler      *chat.Handler
	dataHandler      *data.Handler
	wargameEngine    *wargame.Engine
}

// New creates a new Server instance.
func New() *Server {
	// Load environment variables from .env file if it exists
	if err := godotenv.Load(); err != nil {
		// We don't have slog configured yet, so we use the standard logger here.
		// This is acceptable as it's only for the initial setup.
		log.Println("No .env file found, relying on environment variables")
	}

	logging.New() // Initialize the structured logger
	cfg := config.New()
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

	// Create and run two separate hubs for our two channels.
	htmlHub := hub.NewHub()
	go htmlHub.Run()

	dataHub := hub.NewHub()
	go dataHub.Run()

	// Create stores and handlers, making them dependencies of the server.
	userStore := database.NewSurrealUserStore(db, cfg.GetDBNs(), cfg.GetDBDb())
	homeHandler := handlers.NewHomeHandler()
	dataHandler := data.NewHandler(dataHub)
	authHandler := handlers.NewAuthHandler(userStore, emailer, cfg.GetAppBaseURL())
	dashboardHandler := handlers.NewDashboardHandler()

	e := echo.New()
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Configure and use session middleware
	store := sessions.NewCookieStore([]byte(cfg.GetSessionSecret()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	e.Use(session.Middleware(store))

	// Serve static files from the "web/static" directory.
	e.Static("/static", "web/static")

	// Setup template renderer
	renderer := templates.NewRenderer("web/src/templates")
	e.Renderer = renderer

	// Create the handler for our chat module, injecting the HTML hub and renderer.
	chatHandler := chat.NewHandler(htmlHub, renderer)

	// Create the wargame engine, injecting both hubs and the renderer.
	wargameEngine := wargame.NewEngine(htmlHub, dataHub, renderer)

	return &Server{
		E:                e,
		DB:               db,
		Cfg:              cfg,
		Emailer:          emailer,
		userStore:        userStore,
		homeHandler:      homeHandler,
		authHandler:      authHandler,
		dashboardHandler: dashboardHandler,
		htmlHub:          htmlHub,
		dataHub:          dataHub,
		chatHandler:      chatHandler,
		dataHandler:      dataHandler,
		wargameEngine:    wargameEngine,
	}
}

// UserStore is a getter for the server's user store, useful for testing.
func (s *Server) UserStore() domain.UserRepository {
	return s.userStore
}

// WargameEngine is a getter for the server's wargame engine, useful for testing.
func (s *Server) WargameEngine() *wargame.Engine {
	return s.wargameEngine
}
