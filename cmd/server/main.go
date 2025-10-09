package main

import (
	"context"
	"log"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/logging"
	"github.com/nfrund/goby/internal/module"
	"github.com/nfrund/goby/internal/modules/chat"
	"github.com/nfrund/goby/internal/modules/wargame"
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
	if AppStatic != "" {
		os.Setenv("APP_STATIC", AppStatic)
	}

	// 1. Load config and logger first.
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables.")
	}
	cfg := config.New()
	logging.New()

	// 2. Create the DI Registry.
	reg := registry.New(cfg)

	// 3. Initialize and register core services.
	surrealDB, err := database.NewDB(context.Background(), cfg)
	if err != nil {
		slog.Error("Failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer surrealDB.Close(context.Background())

	// Create and register the database client
	dbClient := database.NewClient(surrealDB)
	reg.Set((*database.Client)(nil), dbClient)
	reg.Set((*config.Provider)(nil), cfg)

	// Create and register the user repository.
	userStore := database.NewSurrealUserStore(surrealDB, cfg.GetDBNs(), cfg.GetDBDb())
	reg.Set((*domain.UserRepository)(nil), userStore)

	emailer, err := email.NewEmailService(cfg)
	if err != nil {
		slog.Error("Failed to initialize email service", "error", err)
		os.Exit(1)
	}
	// Register the email service with its interface type
	reg.Set((*domain.EmailSender)(nil), emailer)

	ps := pubsub.NewWatermillBridge()
	defer func() {
		slog.Info("Shutting down Pub/Sub system...")
		if err := ps.Close(); err != nil {
			slog.Error("Failed to close Pub/Sub system", "error", err)
		}
	}()
	// Register pubsub services with their interface types
	reg.Set((*pubsub.Publisher)(nil), ps)
	reg.Set((*pubsub.Subscriber)(nil), ps)

	wsBridge := websocket.NewBridge(ps)
	reg.Set((*websocket.Bridge)(nil), wsBridge)

	renderer := rendering.NewUniversalRenderer()
	// Register the renderer against both interfaces it implements.
	reg.Set((*rendering.Renderer)(nil), renderer)
	reg.Set((*echo.Renderer)(nil), renderer)

	// 4. Create the server instance.
	// The server now only needs the registry to get its dependencies.
	s, err := server.New(reg)
	if err != nil {
		slog.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	// 5. Define the list of active modules for the application.
	modules := []module.Module{
		wargame.New(),
		chat.New(),
	}

	// 6. Run the two-phase module initialization.
	s.InitModules(modules, reg)
	s.RegisterRoutes()

	// 7. Start the server and its background processes.
	s.Start()
}
