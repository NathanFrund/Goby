package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
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
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/script"
	"github.com/nfrund/goby/internal/server"
	"github.com/nfrund/goby/internal/storage"
	"github.com/nfrund/goby/internal/topicmgr"
	"github.com/nfrund/goby/internal/websocket"
	wsTopics "github.com/nfrund/goby/internal/websocket"
	"github.com/spf13/afero"
)

// AppStatic can be set at build time to force an asset loading strategy.
// Example: go build -ldflags "-X 'main.AppStatic=embed'"
var AppStatic string

func main() {
	// Parse command line flags
	var (
		extractScripts = flag.String("extract-scripts", "", "Extract embedded scripts to specified directory and exit")
		forceExtract   = flag.Bool("force-extract", false, "Overwrite existing files when extracting scripts")
	)
	flag.Parse()

	// 1. Initialize Configuration and Logging
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}
	cfg := config.New()
	logging.New()

	// 2. Handle script extraction if requested
	if *extractScripts != "" {
		if err := handleScriptExtraction(*extractScripts, *forceExtract, cfg); err != nil {
			slog.Error("Script extraction failed", "error", err)
			os.Exit(1)
		}
		return // Exit after extraction
	}

	// 3. Build and Start Server (normal operation)
	appCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	srv, cleanup, err := buildServer(appCtx, cfg)
	if err != nil {
		slog.Error("Failed to build server", "error", err)
		os.Exit(1)
	}
	defer cleanup()

	// 4. Start the server and its background processes
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

	// Create the topic manager
	topicManager := topicmgr.Default()

	// Register all WebSocket framework topics
	if err := wsTopics.RegisterTopics(); err != nil {
		return nil, nil, fmt.Errorf("failed to register WebSocket topics: %w", err)
	}

	// Register presence framework topics
	if err := presence.RegisterTopics(); err != nil {
		return nil, nil, fmt.Errorf("failed to register presence topics: %w", err)
	}

	// Create the dual WebSocket bridges
	htmlBridge := websocket.NewBridge("html", websocket.BridgeDependencies{
		Publisher:    ps,
		Subscriber:   ps,
		TopicManager: topicManager,
		ReadyTopic:   wsTopics.TopicClientReady,
	})

	dataBridge := websocket.NewBridge("data", websocket.BridgeDependencies{
		Publisher:    ps,
		Subscriber:   ps,
		TopicManager: topicManager,
		ReadyTopic:   wsTopics.TopicClientReady,
	})

	// Start the WebSocket bridges
	if err := htmlBridge.Start(appCtx); err != nil {
		return nil, nil, fmt.Errorf("failed to start HTML WebSocket bridge: %w", err)
	}
	closers = append(closers, func() error {
		slog.Info("Shutting down HTML WebSocket bridge...")
		htmlBridge.Shutdown(context.Background())
		return nil
	})

	if err := dataBridge.Start(appCtx); err != nil {
		return nil, nil, fmt.Errorf("failed to start Data WebSocket bridge: %w", err)
	}
	closers = append(closers, func() error {
		slog.Info("Shutting down Data WebSocket bridge...")
		dataBridge.Shutdown(context.Background())
		return nil
	})

	// Renderer (needed for presence service)
	renderer := rendering.NewUniversalRenderer()

	// Presence Service
	presenceService := presence.NewService(ps, ps, topicManager) // Using default options
	reg.Set((*presence.Service)(nil), presenceService)
	slog.Info("Presence service initialized")

	// Script Engine
	scriptEngine, err := script.RegisterService(reg, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to register script engine: %w", err)
	}
	closers = append(closers, func() error {
		slog.Info("Shutting down script engine...")
		return scriptEngine.Shutdown(context.Background())
	})
	slog.Info("Script engine initialized")

	// User Store (using the new v2 client)
	userDBClient, err := database.NewClient[domain.User](dbConn, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create user db client: %w", err)
	}
	userStore := database.NewUserStore(userDBClient, cfg)

	// Web Framework (renderer already created above)
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

	fileHandler := handlers.NewFileHandler(
		fileStorage,
		fileRepo,
		cfg.GetMaxFileSize(),
		cfg.GetAllowedMimeTypes(),
	)

	dashboardHandler := handlers.NewDashboardHandler(fileRepo)
	presenceHandler := handlers.NewPresenceHandler(presenceService)

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
		HTMLBridge:       htmlBridge,
		DataBridge:       dataBridge,
		DashboardHandler: dashboardHandler,
		PresenceHandler:  presenceHandler,
		FileHandler:      fileHandler,
		ScriptEngine:     scriptEngine,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	// 4. Initialize Application Modules
	// Core services are passed to the module container, which then wires up
	// all active application features.
	slog.Info("Initializing application modules...")
	moduleDeps := app.Dependencies{
		Publisher:       ps,
		Subscriber:      ps,
		Renderer:        renderer,
		TopicMgr:        topicManager,
		PresenceService: presenceService,
		ScriptEngine:    scriptEngine,
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

		// The order of shutdown is critical to prevent hangs.
		// 1. Shut down the HTTP server first. This stops accepting new connections
		//    and allows active requests to finish.
		slog.Info("Shutting down HTTP server...")
		if err := srv.E.Shutdown(shutdownCtx); err != nil {
			slog.Error("Errors during HTTP server shutdown", "error", err)
		}

		// 2. Shut down modules, which may have background workers.
		var errs error
		for _, mod := range modules {
			errs = errors.Join(errs, mod.Shutdown(shutdownCtx))
		}

		// 3. Shut down core services in reverse order of creation.
		//    The closers slice is already in the correct reverse order.
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

// handleScriptExtraction extracts embedded scripts to the filesystem
func handleScriptExtraction(targetDir string, force bool, cfg config.Provider) error {
	fmt.Printf("Goby Script Extractor\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("Target directory: %s\n", targetDir)
	fmt.Printf("Force overwrite: %v\n\n", force)

	// Check if target directory exists and handle force flag
	if err := prepareTargetDirectory(targetDir, force); err != nil {
		return err
	}

	// Create script engine
	scriptEngine := script.NewEngine(script.Dependencies{
		Config: cfg,
	})

	// Create minimal dependencies for modules to register their embedded scripts
	ps := pubsub.NewWatermillBridge()
	defer ps.Close()

	renderer := rendering.NewUniversalRenderer()
	topicManager := topicmgr.Default()
	presenceService := presence.NewService(ps, ps, topicManager)

	moduleDeps := app.Dependencies{
		Publisher:       ps,
		Subscriber:      ps,
		Renderer:        renderer,
		TopicMgr:        topicManager,
		PresenceService: presenceService,
		ScriptEngine:    scriptEngine,
	}

	// Initialize modules to register their embedded scripts
	slog.Info("Initializing modules to register embedded scripts...")
	_ = app.NewModules(moduleDeps)

	// The modules will register their embedded scripts during creation
	// No need to manually register them here since they do it in their constructors

	// Initialize the script engine to load embedded scripts
	ctx := context.Background()
	// Enable hot-reload by default, disable with HOT_RELOAD_SCRIPTS=false
	hotReloadEnabled := os.Getenv("HOT_RELOAD_SCRIPTS") != "false"
	if err := scriptEngine.Initialize(ctx, hotReloadEnabled); err != nil {
		return fmt.Errorf("failed to initialize script engine: %w", err)
	}

	// Extract the scripts
	slog.Info("Extracting embedded scripts...")
	if err := scriptEngine.ExtractDefaultScripts(targetDir); err != nil {
		return fmt.Errorf("failed to extract scripts: %w", err)
	}

	// Show summary and next steps
	showExtractionSummary(targetDir)

	fmt.Printf("\n✅ Script extraction completed successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Review the extracted scripts in: %s\n", targetDir)
	fmt.Printf("2. Modify scripts as needed for your use case\n")
	fmt.Printf("3. Start the Goby server normally - it will automatically detect and use your custom scripts\n")
	fmt.Printf("4. Scripts will be hot-reloaded when you save changes\n\n")

	return nil
}

// prepareTargetDirectory prepares the target directory for extraction
func prepareTargetDirectory(targetDir string, force bool) error {
	// Check if directory exists
	if _, err := os.Stat(targetDir); err == nil {
		if !force {
			fmt.Printf("⚠️  Target directory '%s' already exists.\n", targetDir)
			fmt.Printf("Use --force-extract to overwrite existing files, or choose a different directory.\n")
			return fmt.Errorf("target directory exists and --force-extract not specified")
		}
		fmt.Printf("📁 Target directory exists, will overwrite files due to --force-extract flag\n")
	} else if os.IsNotExist(err) {
		// Directory doesn't exist, create it
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create target directory: %w", err)
		}
		fmt.Printf("📁 Created target directory: %s\n", targetDir)
	} else {
		return fmt.Errorf("failed to check target directory: %w", err)
	}

	return nil
}

// showExtractionSummary displays a summary of extracted scripts
func showExtractionSummary(targetDir string) {
	fmt.Printf("\n📊 Extraction Summary:\n")
	fmt.Printf("===================\n")

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && (filepath.Ext(path) == ".tengo" || filepath.Ext(path) == ".zygomys" || filepath.Ext(path) == "") {
			relPath, _ := filepath.Rel(targetDir, path)
			fmt.Printf("📄 %s (%d bytes)\n", relPath, info.Size())
		}

		return nil
	})

	if err != nil {
		slog.Error("Failed to walk extracted files", "error", err)
	}
}
