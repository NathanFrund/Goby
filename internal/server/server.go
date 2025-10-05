package server

import (
	"context"
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
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/modules/data"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/websocket"
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
	Renderer  echo.Renderer

	homeHandler      *handlers.HomeHandler
	authHandler      *handlers.AuthHandler
	dashboardHandler *handlers.DashboardHandler
	aboutHandler     *handlers.AboutHandler

	htmlHub     *hub.Hub
	dataHub     *hub.Hub
	dataHandler *data.Handler
	wsBridge    *websocket.WebsocketBridge
	newBridge   *websocket.Bridge
	PubSub      pubsub.Publisher
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

// ServerOption is a function that configures a Server.
type ServerOption func(*Server) error

// WithConfig is an option to set the configuration provider.
func WithConfig(cfg config.Provider) ServerOption {
	return func(s *Server) error {
		s.Cfg = cfg
		return nil
	}
}

// WithDB is an option to set the database connection and the UserStore.
func WithDB(db *surrealdb.DB, ns, dbName string) ServerOption {
	return func(s *Server) error {
		s.DB = db
		s.UserStore = database.NewSurrealUserStore(db, ns, dbName)
		return nil
	}
}

// WithEmailer is an option to set the email sender.
func WithEmailer(emailer domain.EmailSender) ServerOption {
	return func(s *Server) error {
		s.Emailer = emailer
		return nil
	}
}

// WithHubs is an option to set the WebSocket hubs.
func WithHubs(htmlHub, dataHub *hub.Hub) ServerOption {
	return func(s *Server) error {
		s.htmlHub = htmlHub
		s.dataHub = dataHub
		return nil
	}
}

// WithRenderer is an option to set the component renderer.
func WithRenderer(renderer echo.Renderer) ServerOption {
	return func(s *Server) error {
		s.Renderer = renderer
		return nil
	}
}

// WithPubSub is an option to set the Pub/Sub service.
func WithPubSub(pubSub pubsub.Publisher) ServerOption {
	return func(s *Server) error {
		s.PubSub = pubSub
		return nil
	}
}

// WithWebsocketBridge is an option to set the new WebSocket bridge.
func WithWebsocketBridge(bridge *websocket.WebsocketBridge) ServerOption {
	return func(s *Server) error {
		s.wsBridge = bridge
		return nil
	}
}

// WithNewBridge is an option to set the new V2 WebSocket bridge.
func WithNewBridge(bridge *websocket.Bridge) ServerOption {
	return func(s *Server) error {
		s.newBridge = bridge
		return nil
	}
}

// New creates a new Server instance by applying functional options.
func New(opts ...ServerOption) (*Server, error) {
	e := echo.New()
	setupErrorHandling(e)

	s := &Server{
		E: e,
	}

	// Loop through the provided options and apply them.
	for _, opt := range opts {
		if err := opt(s); err != nil {
			return nil, err
		}
	}

	s.initHandlers()

	// Set the renderer on the Echo instance so it can be used for page rendering.
	s.E.Renderer = s.Renderer

	// Configure and use session middleware
	store := sessions.NewCookieStore([]byte(s.Cfg.GetSessionSecret()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	s.E.Use(session.Middleware(store))

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

	// Register all application routes, including those from modules.
	s.RegisterRoutes()

	return s, nil
}

// initHandlers initializes all handler structs using the Server's dependencies.
func (s *Server) initHandlers() {
	s.homeHandler = handlers.NewHomeHandler()
	s.dataHandler = data.NewHandler(s.dataHub)
	s.authHandler = handlers.NewAuthHandler(s.UserStore, s.Emailer, s.Cfg.GetAppBaseURL())
	s.dashboardHandler = handlers.NewDashboardHandler()
	s.aboutHandler = &handlers.AboutHandler{}
}

// Start runs the HTTP server with graceful shutdown.
func (s *Server) Start() {
	addr := s.Cfg.GetServerAddr()

	// Start server in a goroutine so that it doesn't block.
	// Also start the hubs, which are background services of the server.
	go s.htmlHub.Run()
	// Start the new WebsocketBridge runner
	if s.wsBridge != nil {
		go s.wsBridge.Run()
	}
	// Start the new V2 WebsocketBridge runner
	if s.newBridge != nil {
		go s.newBridge.Run()
	}
	go s.dataHub.Run()

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
