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
	"github.com/samber/do/v2"
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
// creating and connecting all the application's components using dependency injection.
func buildServer(appCtx context.Context, cfg config.Provider) (srv *server.Server, cleanup func(), err error) {
	// Set static asset loading strategy if specified
	if AppStatic != "" {
		os.Setenv("APP_STATIC", AppStatic)
	}

	// Create the DI container
	injector := do.New()

	// Provide the configuration and app context
	do.ProvideValue(injector, cfg)
	do.ProvideValue(injector, appCtx)

	// Provide the service registry (framework-agnostic - doesn't know about do/v2)
	do.Provide(injector, func(i do.Injector) (*registry.Registry, error) {
		cfg := do.MustInvoke[config.Provider](i)
		return registry.New(cfg), nil
	})

	// Provide core services
	do.Provide(injector, provideDatabaseConnection)
	do.Provide(injector, provideEmailService)
	// Provide pubsub as both Publisher and Subscriber (WatermillBridge implements both)
	do.Provide(injector, providePubSub)
	do.Provide(injector, provideSubscriber)
	do.Provide(injector, provideTopicManager)
	do.Provide(injector, provideRenderer)
	// Provide renderer as echo.Renderer as well (UniversalRenderer implements both)
	do.Provide(injector, provideEchoRenderer)
	do.Provide(injector, providePresenceService)
	do.Provide(injector, provideScriptEngine)
	do.Provide(injector, provideEcho)
	do.Provide(injector, provideStorage)

	// Provide database clients and stores
	do.Provide(injector, provideUserStore)
	do.Provide(injector, provideFileStore)

	// Provide WebSocket bridges (after pubsub and topic manager)
	do.ProvideNamed(injector, "html", provideHTMLBridge)
	do.ProvideNamed(injector, "data", provideDataBridge)

	// Provide handlers
	do.Provide(injector, provideFileHandler)
	do.Provide(injector, provideDashboardHandler)
	do.Provide(injector, providePresenceHandler)

	// Provide module dependencies
	do.Provide(injector, provideModuleDependencies)

	// Provide the server (depends on everything above)
	do.Provide(injector, provideServer)

	// Register topics (must be done before bridges start)
	if err := wsTopics.RegisterTopics(); err != nil {
		return nil, nil, fmt.Errorf("failed to register WebSocket topics: %w", err)
	}
	if err := presence.RegisterTopics(); err != nil {
		return nil, nil, fmt.Errorf("failed to register presence topics: %w", err)
	}

	// Get services from DI container and initialize them
	reg, err := do.Invoke[*registry.Registry](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get registry: %w", err)
	}

	// Database connection needs explicit initialization
	dbConn, err := do.Invoke[*database.Connection](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := dbConn.Connect(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	dbConn.StartMonitoring()

	// Register the core connection manager in the registry (registry receives plain value)
	reg.Set((*database.Connection)(nil), dbConn)

	// Get presence service and register in registry (registry is agnostic)
	presenceService, err := do.Invoke[*presence.Service](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get presence service: %w", err)
	}
	reg.Set((*presence.Service)(nil), presenceService)
	slog.Info("Presence service initialized")

	// Get script engine (provideScriptEngine already handles registry registration)
	scriptEngine, err := do.Invoke[script.ScriptEngine](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get script engine: %w", err)
	}
	slog.Info("Script engine initialized")

	// Start WebSocket bridges (they need explicit startup)
	htmlBridge, err := do.InvokeNamed[*websocket.Bridge](injector, "html")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get HTML bridge: %w", err)
	}
	if err := htmlBridge.Start(appCtx); err != nil {
		return nil, nil, fmt.Errorf("failed to start HTML WebSocket bridge: %w", err)
	}

	dataBridge, err := do.InvokeNamed[*websocket.Bridge](injector, "data")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get data bridge: %w", err)
	}
	if err := dataBridge.Start(appCtx); err != nil {
		return nil, nil, fmt.Errorf("failed to start Data WebSocket bridge: %w", err)
	}

	// Get the server
	srv, err = do.Invoke[*server.Server](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create server: %w", err)
	}

	// Initialize modules
	moduleDeps, err := do.Invoke[app.Dependencies](injector)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get module dependencies: %w", err)
	}
	modules := app.NewModules(moduleDeps)
	srv.InitModules(appCtx, modules, reg)
	srv.RegisterRoutes()

	// Define cleanup function
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

		// 3. Shut down bridges
		slog.Info("Shutting down WebSocket bridges...")
		htmlBridge.Shutdown(context.Background())
		dataBridge.Shutdown(context.Background())

		// 4. Shut down script engine
		slog.Info("Shutting down script engine...")
		if err := scriptEngine.Shutdown(context.Background()); err != nil {
			errs = errors.Join(errs, err)
		}

		// 5. Shut down database
		slog.Info("Closing database connection...")
		if err := dbConn.Close(context.Background()); err != nil {
			errs = errors.Join(errs, err)
		}

		// 6. Shut down pubsub
		slog.Info("Shutting down Pub/Sub system...")
		ps := do.MustInvoke[pubsub.Publisher](injector)
		if pubSubBridge, ok := ps.(interface{ Close() error }); ok {
			if err := pubSubBridge.Close(); err != nil {
				errs = errors.Join(errs, err)
			}
		}

		// 7. Shutdown the container
		_ = injector.Shutdown()

		if errs != nil {
			slog.Error("Errors during shutdown", "errors", errs)
		}

		slog.Info("Shutdown process complete.")
	}

	return srv, cleanup, nil
}

// Provider functions for dependency injection
// Note: Provider[T] signature is func(Injector) (T, error)
// Dependencies are resolved using do.Invoke or do.MustInvoke within providers

func provideDatabaseConnection(i do.Injector) (*database.Connection, error) {
	cfg := do.MustInvoke[config.Provider](i)
	return database.NewConnection(cfg), nil
}

func provideEmailService(i do.Injector) (domain.EmailSender, error) {
	cfg := do.MustInvoke[config.Provider](i)
	return email.NewEmailService(cfg)
}

func providePubSub(i do.Injector) (pubsub.Publisher, error) {
	return pubsub.NewWatermillBridge(), nil
}

func provideSubscriber(i do.Injector) (pubsub.Subscriber, error) {
	// WatermillBridge implements both Publisher and Subscriber
	ps := do.MustInvoke[pubsub.Publisher](i)
	return ps.(pubsub.Subscriber), nil
}

func provideTopicManager(i do.Injector) (*topicmgr.Manager, error) {
	return topicmgr.Default(), nil
}

func provideRenderer(i do.Injector) (rendering.Renderer, error) {
	return rendering.NewUniversalRenderer(), nil
}

func provideEchoRenderer(i do.Injector) (echo.Renderer, error) {
	// UniversalRenderer implements both rendering.Renderer and echo.Renderer
	r := do.MustInvoke[rendering.Renderer](i)
	return r.(echo.Renderer), nil
}

func providePresenceService(i do.Injector) (*presence.Service, error) {
	ps := do.MustInvoke[pubsub.Publisher](i)
	sub := do.MustInvoke[pubsub.Subscriber](i)
	topicMgr := do.MustInvoke[*topicmgr.Manager](i)
	return presence.NewService(ps, sub, topicMgr), nil
}

func provideScriptEngine(i do.Injector) (script.ScriptEngine, error) {
	reg := do.MustInvoke[*registry.Registry](i)
	cfg := do.MustInvoke[config.Provider](i)
	// Register the script service in the registry first
	// The registry is agnostic - it just receives the service as a value
	scriptEngine, err := script.RegisterService(reg, cfg)
	if err != nil {
		return nil, err
	}
	return scriptEngine, nil
}

func provideEcho(i do.Injector) (*echo.Echo, error) {
	return echo.New(), nil
}

func provideStorage(i do.Injector) (storage.Store, error) {
	cfg := do.MustInvoke[config.Provider](i)
	if cfg.GetStorageBackend() == "mem" {
		slog.Info("Using in-memory file storage")
		return storage.NewAferoStore(afero.NewMemMapFs()), nil
	}
	slog.Info("Using OS file storage", "path", cfg.GetStoragePath())
	return storage.NewAferoStore(afero.NewBasePathFs(afero.NewOsFs(), cfg.GetStoragePath())), nil
}

func provideUserStore(i do.Injector) (domain.UserRepository, error) {
	dbConn := do.MustInvoke[*database.Connection](i)
	userDBClient, err := database.NewClient[domain.User](dbConn)
	if err != nil {
		return nil, err
	}
	return database.NewUserStore(userDBClient, dbConn), nil
}

func provideFileStore(i do.Injector) (*database.FileStore, error) {
	dbConn := do.MustInvoke[*database.Connection](i)
	fileClient, err := database.NewClient[domain.File](dbConn)
	if err != nil {
		return nil, err
	}
	return database.NewFileStore(fileClient), nil
}

func provideHTMLBridge(i do.Injector) (*websocket.Bridge, error) {
	ps := do.MustInvoke[pubsub.Publisher](i)
	sub := do.MustInvoke[pubsub.Subscriber](i)
	topicMgr := do.MustInvoke[*topicmgr.Manager](i)
	return websocket.NewBridge("html", websocket.BridgeDependencies{
		Publisher:    ps,
		Subscriber:   sub,
		TopicManager: topicMgr,
		ReadyTopic:   wsTopics.TopicClientReady,
	}), nil
}

func provideDataBridge(i do.Injector) (*websocket.Bridge, error) {
	ps := do.MustInvoke[pubsub.Publisher](i)
	sub := do.MustInvoke[pubsub.Subscriber](i)
	topicMgr := do.MustInvoke[*topicmgr.Manager](i)
	return websocket.NewBridge("data", websocket.BridgeDependencies{
		Publisher:    ps,
		Subscriber:   sub,
		TopicManager: topicMgr,
		ReadyTopic:   wsTopics.TopicClientReady,
	}), nil
}

func provideFileHandler(i do.Injector) (*handlers.FileHandler, error) {
	fileStorage := do.MustInvoke[storage.Store](i)
	fileRepo := do.MustInvoke[*database.FileStore](i)
	cfg := do.MustInvoke[config.Provider](i)
	return handlers.NewFileHandler(
		fileStorage,
		fileRepo,
		cfg.GetMaxFileSize(),
		cfg.GetAllowedMimeTypes(),
	), nil
}

func provideDashboardHandler(i do.Injector) (*handlers.DashboardHandler, error) {
	fileRepo := do.MustInvoke[*database.FileStore](i)
	return handlers.NewDashboardHandler(fileRepo), nil
}

func providePresenceHandler(i do.Injector) (*handlers.PresenceHandler, error) {
	presenceService := do.MustInvoke[*presence.Service](i)
	return handlers.NewPresenceHandler(presenceService), nil
}

// provideModuleDependencies creates the app.Dependencies struct for module initialization
func provideModuleDependencies(i do.Injector) (app.Dependencies, error) {
	ps := do.MustInvoke[pubsub.Publisher](i)
	sub := do.MustInvoke[pubsub.Subscriber](i)
	renderer := do.MustInvoke[rendering.Renderer](i)
	topicMgr := do.MustInvoke[*topicmgr.Manager](i)
	presenceService := do.MustInvoke[*presence.Service](i)
	scriptEngine := do.MustInvoke[script.ScriptEngine](i)
	return app.Dependencies{
		Publisher:       ps,
		Subscriber:      sub,
		Renderer:        renderer,
		TopicMgr:        topicMgr,
		PresenceService: presenceService,
		ScriptEngine:    scriptEngine,
	}, nil
}

func provideServer(i do.Injector) (*server.Server, error) {
	cfg := do.MustInvoke[config.Provider](i)
	emailer := do.MustInvoke[domain.EmailSender](i)
	userStore := do.MustInvoke[domain.UserRepository](i)
	echoRenderer := do.MustInvoke[echo.Renderer](i)
	ps := do.MustInvoke[pubsub.Publisher](i)
	echo := do.MustInvoke[*echo.Echo](i)
	htmlBridge := do.MustInvokeNamed[*websocket.Bridge](i, "html")
	dataBridge := do.MustInvokeNamed[*websocket.Bridge](i, "data")
	fileHandler := do.MustInvoke[*handlers.FileHandler](i)
	dashboardHandler := do.MustInvoke[*handlers.DashboardHandler](i)
	presenceHandler := do.MustInvoke[*handlers.PresenceHandler](i)
	scriptEngine := do.MustInvoke[script.ScriptEngine](i)
	return server.New(server.Dependencies{
		Config:           cfg,
		Emailer:          emailer,
		UserStore:        userStore,
		Renderer:         echoRenderer,
		Publisher:        ps,
		Echo:             echo,
		HTMLBridge:       htmlBridge,
		DataBridge:       dataBridge,
		FileHandler:      fileHandler,
		DashboardHandler: dashboardHandler,
		PresenceHandler:  presenceHandler,
		ScriptEngine:     scriptEngine,
	})
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
