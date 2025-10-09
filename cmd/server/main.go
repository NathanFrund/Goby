package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/logging"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/websocket"
)

// AppStatic can be set at build time to force an asset loading strategy.
// Example: go build -ldflags "-X 'main.AppStatic=embed'"
var AppStatic string

func main() {
	// 1. Initialize Configuration and Logging
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}
	cfg := config.New()
	logging.New()

	// 2. Build and Start Server
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv, cleanup, err := buildServer(appCtx, cfg)
	if err != nil {
		slog.Error("Failed to build server", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// 3. Start the server and its background processes
	srv.Start(appCtx)
}

// buildServer is the "Composition Root" of the application. It's responsible for
// creating and connecting all the application's components.
func buildServer(appCtx context.Context, cfg config.Provider) (srv *server.Server, cleanup func(), err error) {
	// Set static asset loading strategy if specified
	if AppStatic != "" {
		os.Setenv("APP_STATIC", AppStatic)
	}

	// 1. Create the service registry and a list for cleanup functions.
	// The registry is used for inter-module communication, not for core dependency injection.
	reg := registry.New(cfg)
	var closers []func() error

	// 2. Initialize Core Services (Database, Pub/Sub, etc.)
	// Each service is created, and its cleanup function is added to the closers list.
	slog.Info("Initializing core services...")

	// Database
	db, err := database.NewDB(context.Background(), cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	closers = append(closers, func() error {
		slog.Info("Closing database connection...")
		return db.Close(context.Background())
	})
	dbClient := database.NewClient(db)

	// Email
	emailer, err := email.NewEmailService(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create email service: %w", err)
	}

	// Pub/Sub and WebSocket Bridge
	ps := pubsub.NewWatermillBridge()
	closers = append(closers, func() error {
		slog.Info("Shutting down Pub/Sub system...")
		return ps.Close()
	})
	wsBridge := websocket.NewBridge(ps)

	// User Store (depends on the concrete *surrealdb.DB)
	userStore := database.NewSurrealUserStore(db, cfg.GetDBNs(), cfg.GetDBDb())

	// Renderer and Web Framework
	renderer := rendering.NewUniversalRenderer()
	e := echo.New()

	// 3. Assemble and Create the Main Server Instance
	// All core dependencies are explicitly passed to the server's constructor.
	slog.Info("Creating server instance...")
	srv, err = server.New(server.Dependencies{
		Config:    cfg,
		DB:        dbClient,
		Emailer:   emailer,
		UserStore: userStore,
		Renderer:  renderer,
		Publisher: ps,
		Echo:      e,
		Bridge:    wsBridge,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	// 4. Initialize Application Modules
	// Core services are passed to the module container, which then wires up
	// all active application features.
	slog.Info("Initializing application modules...")
	moduleDeps := app.Dependencies{
		Publisher:  ps,
		Subscriber: ps,
		Bridge:     wsBridge,
		Renderer:   renderer,
	}
	modules := app.NewModules(moduleDeps)
	srv.InitModules(appCtx, modules, reg)
	srv.RegisterRoutes()

	// Add the Echo server shutdown to the list of cleanup tasks.
	closers = append(closers, func() error {
		slog.Info("Shutting down HTTP server...")
		return srv.E.Shutdown(context.Background())
	})

	// 5. Define the master cleanup function.
	cleanup = func() {
		slog.Info("Shutting down application...")
		var errs error

		// Shut down modules
		for _, mod := range modules {
			errs = errors.Join(errs, mod.Shutdown(context.Background()))
		}

		for _, closeFn := range closers {
			errs = errors.Join(errs, closeFn())
		}
		if errs != nil {
			slog.Error("Errors during shutdown", "errors", errs)
		}
		slog.Info("Shutdown process complete.")
	}

	return srv, cleanup, nil
}
