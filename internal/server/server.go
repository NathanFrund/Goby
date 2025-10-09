package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/handlers"
	appmiddleware "github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/websocket"
	"github.com/nfrund/goby/web"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	E         *echo.Echo
	DB        database.Client
	Cfg       config.Provider
	Emailer   domain.EmailSender
	UserStore domain.UserRepository
	Renderer  rendering.Renderer

	homeHandler      *handlers.HomeHandler
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	aboutHandler     *handlers.AboutHandler
	modules          []module.Module
	bridge           websocket.Bridge
	PubSub           pubsub.Publisher
}

func setupErrorHandling(e *echo.Echo) {
	// 1. Recover Middleware: CRITICAL for Panics
	// This catches any panic that occurs during request handling, prevents the Go app
	// from crashing, and logs the full stack trace to your console.
	e.Use(middleware.Recover())

	// 2. Custom HTTP Error Handler: CRITICAL for Unhandled Errors
	// This intercepts errors returned by handlers (e.g., 'return err') or by Echo's internal systems.
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return // Cannot write headers after the response is committed.
		}

		// Try to cast the error to a standard Echo HTTPError
		he, ok := err.(*echo.HTTPError)
		if !ok {
			// If it's not an Echo HTTPError, it's an unexpected internal error.
			slog.Error("Internal Server Error (Unhandled)",
				"error", err.Error(),
				"method", c.Request().Method,
				"path", c.Path(),
				"remote_ip", c.RealIP(),
				// Log the Request ID if available (from middleware.RequestID)
				"request_id", c.Response().Header().Get(echo.HeaderXRequestID),
			)
			// Ensure we still return a standard 500 response
			he = &echo.HTTPError{Code: http.StatusInternalServerError, Message: http.StatusText(http.StatusInternalServerError)}
		}

		// Log all 5xx errors returned by handlers as errors, and 4xx as warnings.
		if he.Code >= 500 {
			slog.Error("HTTP Error",
				"status", he.Code,
				"message", he.Message,
				"path", c.Path(),
				"method", c.Request().Method,
			)
		} else if he.Code >= 400 {
			slog.Warn("Client Error",
				"status", he.Code,
				"message", he.Message,
				"path", c.Path(),
				"method", c.Request().Method,
			)
		}

		// Respond to the client (we'll just use JSON for errors for simplicity)
		c.JSON(he.Code, map[string]interface{}{"error": he.Message})
	}
}

// New creates a new Server instance by applying functional options.
func New(reg *registry.Registry) (*Server, error) {
	e := echo.New()
	setupErrorHandling(e)

	// Resolve core server dependencies from the registry with proper error handling
	cfg, err := registry.Get[config.Provider](reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get config: %w", err)
	}

	db, err := registry.Get[database.Client](reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get database client: %w", err)
	}

	emailer, err := registry.Get[domain.EmailSender](reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get emailer: %w", err)
	}

	// Get the echo.Renderer for the Echo instance.
	echoRenderer, err := registry.Get[echo.Renderer](reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get echo.Renderer: %w", err)
	}
	e.Renderer = echoRenderer

	ps, err := registry.Get[pubsub.Publisher](reg)
	if err != nil {
		return nil, fmt.Errorf("failed to get pubsub: %w", err)
	}

	s := &Server{
		E:       e,
		Cfg:     cfg,
		DB:      db,
		Emailer: emailer,
		// Get the custom rendering.Renderer for internal use by modules.
		Renderer:  registry.MustGet[rendering.Renderer](reg),
		PubSub:    ps,
		UserStore: registry.MustGet[domain.UserRepository](reg),
	}

	// Configure and use session middleware
	store := sessions.NewCookieStore([]byte(s.Cfg.GetSessionSecret()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	s.E.Use(session.Middleware(store))

	s.initHandlers()

	// This will be populated by InitModules now.
	s.initCoreServices()

	// Serve static files from disk or embedded FS based on APP_STATIC.
	if os.Getenv("APP_STATIC") == "embed" {
		slog.Info("Serving embedded static assets")
		staticFS, err := fs.Sub(web.FS, "static")
		if err != nil {
			return nil, err
		}
		s.E.GET("/static/*", echo.WrapHandler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))))
	} else {
		slog.Info("Serving static assets from disk")
		s.E.Static("/static", "web/static")
	}

	return s, nil
}

// initCoreServices initializes services that are part of the server itself.
func (s *Server) initCoreServices() {
	// The bridge is a core part of the server's real-time architecture.
	if s.PubSub != nil {
		s.bridge = websocket.NewBridge(s.PubSub)
		slog.Info("WebSocket bridge initialized.")
	}
}

func (s *Server) InitModules(modules []module.Module, reg *registry.Registry) {
	s.modules = modules

	// The bridge is now retrieved from the registry, where it was placed by main.go
	s.bridge = registry.MustGet[websocket.Bridge](reg)

	// --- Phase 1: Register all module services ---
	for _, mod := range modules {
		if err := mod.Register(reg); err != nil {
			slog.Error("Failed to register module", "module", mod.Name(), "error", err)
			// In a real app, you might want to os.Exit(1) here.
		}
	}

	// --- Phase 2: Boot all modules ---
	// Now that all services are registered, modules can safely resolve dependencies.
	protected := s.E.Group("/app")
	protected.Use(appmiddleware.Auth(s.UserStore)) // Auth middleware for all module routes

	for _, mod := range modules {
		// Create a dedicated sub-group for each module under the /app prefix.
		group := protected.Group("/" + mod.Name())
		if err := mod.Boot(group, reg); err != nil {
			slog.Error("Failed to boot module", "module", mod.Name(), "error", err)
		}
	}
}

// initHandlers initializes all handler structs using the Server's dependencies.
func (s *Server) initHandlers() {
	s.homeHandler = handlers.NewHomeHandler()
	s.authHandler = handlers.NewAuthHandler(s.UserStore, s.Emailer, s.Cfg.GetAppBaseURL())
	s.dashboardHandler = handlers.NewDashboardHandler()
	s.aboutHandler = &handlers.AboutHandler{}
}

// Start runs the HTTP server with graceful shutdown.
func (s *Server) Start() {
	addr := s.Cfg.GetServerAddr()

	// Start server in a goroutine so that it doesn't block.
	// Also start the hubs, which are background services of the server.
	if s.bridge != nil {
		go s.bridge.Run()
	}

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
	if s.DB != nil {
		s.DB.Close()
	}

	if err := s.E.Shutdown(ctx); err != nil {
		slog.Error("Server shutdown failed", "error", err)
	}
}
