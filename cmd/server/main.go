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
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/nfrund/goby/internal/logging"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/storage"
	"github.com/nfrund/goby/internal/websocket"
	"github.com/spf13/afero"
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
	dbConn := database.NewConnection(cfg)
	err = dbConn.Connect(context.Background())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	dbConn.StartMonitoring()

	closers = append(closers, func() error {
		slog.Info("Closing database connection...")
		return dbConn.Close(context.Background())
	})
	// Register the core connection manager in the registry so modules can access it.
	reg.Set((*database.Connection)(nil), dbConn)

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

	// User Store (using the new v2 client)
	userDBClient, err := database.NewClient[domain.User](dbConn, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create user db client: %w", err)
	}
	userStore := database.NewUserStore(userDBClient, cfg)

	// Renderer and Web Framework
	renderer := rendering.NewUniversalRenderer()
	e := echo.New()

	// Storage and File Handler
	var fileStorage storage.Store
	if cfg.GetStorageBackend() == "mem" {
		slog.Info("Using in-memory file storage")
		fileStorage = storage.NewAferoStore(afero.NewMemMapFs())
	} else {
		slog.Info("Using OS file storage", "path", cfg.GetStoragePath())
		fileStorage = storage.NewAferoStore(afero.NewBasePathFs(afero.NewOsFs(), cfg.GetStoragePath()))
	}

	fileClient, err := database.NewClient[domain.File](dbConn, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create file db client: %w", err)
	}
	fileRepo := database.NewFileStore(fileClient)

	fileHandler := storage.NewFileHandler(
		fileStorage,
		fileRepo,
		cfg.GetMaxFileSize(),
		cfg.GetAllowedMimeTypes(),
	)

	dashboardHandler := handlers.NewDashboardHandler(fileRepo)

	// 3. Assemble and Create the Main Server Instance
	// All core dependencies are explicitly passed to the server's constructor.
	slog.Info("Creating server instance...")
	srv, err = server.New(server.Dependencies{
		Config:           cfg,
		Emailer:          emailer,
		UserStore:        userStore,
		Renderer:         renderer,
		Publisher:        ps,
		Echo:             e,
		Bridge:           wsBridge,
		DashboardHandler: dashboardHandler,
		FileHandler:      fileHandler,
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

	// 5. Define the master cleanup function.
	cleanup = func() {
		slog.Info("Shutting down application...")

		// Create a context with a timeout for the entire shutdown process.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		var errs error

		// Shut down modules
		for _, mod := range modules {
			errs = errors.Join(errs, mod.Shutdown(shutdownCtx))
		}

		// Shut down core services
		for _, closeFn := range closers {
			errs = errors.Join(errs, closeFn())
		}

		// Finally, shut down the HTTP server. This must be done after services
		// like the database are closed, but within the same timeout.
		slog.Info("Shutting down HTTP server...")
		errs = errors.Join(errs, srv.E.Shutdown(shutdownCtx))

		if errs != nil {
			slog.Error("Errors during shutdown", "errors", errs)
		}
		slog.Info("Shutdown process complete.")
	}

	return srv, cleanup, nil
}
