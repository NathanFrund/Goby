package topics

import (
	"fmt"
	"io"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/registry"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Initialize sets up minimal dependencies to register all topics
// This function extracts the initialization logic from the standalone topics CLI
// and makes it reusable for the goby-cli topics commands.
func Initialize() error {
	// Suppress all logging output to make CLI less chatty
	log.SetOutput(io.Discard)
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))

	// Also suppress watermill logging
	os.Setenv("WATERMILL_LOG_LEVEL", "ERROR")

	// Load .env file if it exists
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error loading .env file: %w", err)
	}

	// Create minimal configuration
	cfg := config.New()

	// Create a minimal registry
	reg := registry.New(cfg)

	// Create minimal module dependencies
	moduleDeps := app.Dependencies{
		Publisher:       nil,
		Subscriber:      nil,
		Renderer:        nil,
		TopicMgr:        topicmgr.Default(),
		PresenceService: nil,
		// Other fields will be zero values
	}

	// Initialize modules to register their topics
	modules := app.NewModules(moduleDeps)

	// Register module topics by calling their Register methods
	for _, mod := range modules {
		if err := mod.Register(reg); err != nil {
			// Ignore registry-related errors since we only care about topic registration
			if !strings.Contains(err.Error(), "nil pointer") {
				return fmt.Errorf("failed to register module %s: %w", mod.Name(), err)
			}
		}
	}

	// All modules should now register their topics in their Register() method
	// This provides a clean, consistent way for the CLI to discover all topics
	// without needing to know about specific modules or their internal structure

	return nil
}
