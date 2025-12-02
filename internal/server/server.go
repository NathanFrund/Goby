package server

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"runtime/debug"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/handlers"
	appmiddleware "github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/script"
	"github.com/nfrund/goby/internal/websocket"
	"github.com/nfrund/goby/web"
)

// Server holds the dependencies for the HTTP server.
type Server struct {
	E               *echo.Echo
	Cfg             config.Provider
	Emailer         domain.EmailSender
	UserStore       domain.UserRepository
	Renderer        rendering.Renderer
	FileHandler     *handlers.FileHandler
	PresenceHandler *handlers.PresenceHandler
	HTMLBridge      *websocket.Bridge
	DataBridge      *websocket.Bridge
	ScriptEngine    script.ScriptEngine

	modules []module.Module
	PubSub  pubsub.Publisher
}

// Dependencies holds all the services that the Server requires to operate.
// This struct is used for constructor injection to make dependencies explicit.
type Dependencies struct {
	Config          config.Provider
	Emailer         domain.EmailSender
	UserStore       domain.UserRepository
	Renderer        echo.Renderer // The renderer for the Echo framework
	Publisher       pubsub.Publisher
	Echo            *echo.Echo
	HTMLBridge      *websocket.Bridge
	DataBridge      *websocket.Bridge
	FileHandler     *handlers.FileHandler
	PresenceHandler *handlers.PresenceHandler
	ScriptEngine    script.ScriptEngine
}

func setupErrorHandling(e *echo.Echo) {
	// 1. Recover Middleware: CRITICAL for Panics
	// 1. Recover Middleware: CRITICAL for Panics.
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
			// Get the request-scoped logger.
			logger := appmiddleware.FromContext(c.Request().Context())

			// If it's not an Echo HTTPError, it's an unexpected internal error.
			logger.Error("Internal Server Error (Unhandled)",
				"error", err.Error(),
				"method", c.Request().Method,
				"path", c.Path(),
				"remote_ip", c.RealIP(),
				"stack_trace", string(debug.Stack()),
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
		errResp := handlers.ErrorResponse{
			Code:    http.StatusText(he.Code), // A simple default code
			Message: fmt.Sprintf("%v", he.Message),
		}
		c.JSON(he.Code, errResp)
	}
}

// New creates a new Server instance by applying functional options.
func New(deps Dependencies) (*Server, error) {
	// The echo instance is now created in main.go and passed in as a dependency.
	// This allows us to configure it before the server is created.
	e := deps.Echo
	setupErrorHandling(e)

	// Register the custom validator.
	e.Validator = handlers.NewValidator()
	e.Renderer = deps.Renderer

	// The server needs the more specific rendering.Renderer for internal use.
	// We perform a safe type assertion to ensure the provided renderer supports it.
	appRenderer, ok := deps.Renderer.(rendering.Renderer)
	if !ok {
		return nil, fmt.Errorf("the provided echo.Renderer does not implement the required rendering.Renderer interface")
	}

	s := &Server{
		E:               e,
		Cfg:             deps.Config,
		Emailer:         deps.Emailer,
		Renderer:        appRenderer,
		PubSub:          deps.Publisher,
		UserStore:       deps.UserStore,
		HTMLBridge:      deps.HTMLBridge,
		DataBridge:      deps.DataBridge,
		FileHandler:     deps.FileHandler,
		PresenceHandler: deps.PresenceHandler,
		ScriptEngine:    deps.ScriptEngine,
	}

	// Configure and use session middleware
	store := sessions.NewCookieStore([]byte(s.Cfg.GetSessionSecret()))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
	}
	s.E.Use(session.Middleware(store))

	// Add the RequestID middleware to assign a unique ID to every request.
	s.E.Use(middleware.RequestID())

	// Add our custom logger middleware to inject a request-scoped logger.
	s.E.Use(appmiddleware.Logger)

	// Add security headers middleware for production hardening.
	s.E.Use(middleware.Secure())

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

// InitModules runs the two-phase startup for all registered application modules.
//
// The process is as follows:
//  1. Register Phase: Each module registers its own services into the provided
//     registry. This allows modules to make their services available to others.
//  2. Boot Phase: Each module performs its startup logic, such as starting
//     background workers and registering HTTP routes. During this phase, a module
//     can safely resolve services that were registered by other modules in the first phase.
func (s *Server) InitModules(ctx context.Context, modules []module.Module, reg *registry.Registry) {
	s.modules = modules

	// --- Phase 0: Register Client Actions ---
	// Allow modules to register their client-callable WebSocket actions.
	// This is done before other phases to ensure the bridges are configured
	// before any module starts its background services.
	for _, mod := range modules {
		if registrar, ok := mod.(module.ClientActionRegistrar); ok {
			registrar.RegisterClientActions(s.HTMLBridge, s.DataBridge)
		}
	}

	// --- Phase 1: Register Module-Provided Services ---
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
		if err := mod.Boot(ctx, group, reg); err != nil {
			slog.Error("Failed to boot module", "module", mod.Name(), "error", err)
		}
	}
}

// GetScriptEngine returns the script engine for use by modules
func (s *Server) GetScriptEngine() script.ScriptEngine {
	return s.ScriptEngine
}

// Start runs the HTTP server with graceful shutdown.
func (s *Server) Start(ctx context.Context) {
	addr := s.Cfg.GetServerAddr()

	// Start server in a goroutine so that it doesn't block.
	go func() {
		slog.Info("Starting server", "address", addr)
		if err := s.E.Start(addr); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	// Block until the application context is canceled.
	<-ctx.Done()
	slog.Info("Server shutting down...")
}
