// internal/script/extractor/extractor.go
package extractor

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/app"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/script"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Extractor handles script extraction operations
type Extractor struct {
	cfg       config.Provider
	force     bool
	hotReload bool
}

// NewExtractor creates a new Extractor instance
func NewExtractor(cfg config.Provider, force bool) *Extractor {
	// Enable hot-reload by default, disable with HOT_RELOAD_SCRIPTS=false
	hotReload := os.Getenv("HOT_RELOAD_SCRIPTS") != "false"
	return &Extractor{
		cfg:       cfg,
		force:     force,
		hotReload: hotReload,
	}
}

// ExtractScripts extracts embedded scripts to the target directory
func (e *Extractor) ExtractScripts(targetDir string) error {
	fmt.Printf("Goby Script Extractor\n")
	fmt.Printf("=====================\n\n")
	fmt.Printf("Target directory: %s\n", targetDir)
	fmt.Printf("Force overwrite: %v\n\n", e.force)

	// Check if target directory exists and handle force flag
	if err := e.prepareTargetDirectory(targetDir); err != nil {
		return err
	}

	// Initialize script engine and dependencies
	scriptEngine, cleanup, err := e.initializeScriptEngine()
	if err != nil {
		return err
	}
	defer cleanup()

	// Extract the scripts
	slog.Info("Extracting embedded scripts...")
	if err := scriptEngine.ExtractDefaultScripts(targetDir); err != nil {
		return fmt.Errorf("failed to extract scripts: %w", err)
	}

	// Show summary and next steps
	e.showExtractionSummary(targetDir)

	fmt.Printf("\n✅ Script extraction completed successfully!\n")
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("1. Review the extracted scripts in: %s\n", targetDir)
	fmt.Printf("2. Modify scripts as needed for your use case\n")
	fmt.Printf("3. Start the Goby server normally - it will automatically detect and use your custom scripts\n")
	fmt.Printf("4. Scripts will be hot-reloaded when you save changes\n\n")

	return nil
}

func (e *Extractor) initializeScriptEngine() (script.ScriptEngine, func(), error) {
	// Create script engine with minimal dependencies
	scriptEngine := script.NewEngine(script.Dependencies{
		Config: e.cfg,
	})

	// Create a minimal pubsub implementation that won't initialize Watermill
	ps := &noopPubSub{}
	cleanup := func() {}

	// Create minimal dependencies for module initialization
	renderer := &noopRenderer{}
	topicManager := topicmgr.Default()

	// Create module dependencies
	moduleDeps := app.Dependencies{
		Publisher:       ps,
		Subscriber:      ps,
		Renderer:        renderer,
		TopicMgr:        topicManager,
		PresenceService: presence.NewService(ps, ps, topicManager),
		// ScriptEngine will be set after initialization
	}

	// Initialize modules to register their embedded scripts
	_ = app.NewModules(moduleDeps)

	// Initialize the script engine to load embedded scripts
	ctx := context.Background()
	if err := scriptEngine.Initialize(ctx, e.hotReload); err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("failed to initialize script engine: %w", err)
	}

	return scriptEngine, cleanup, nil
}

// prepareTargetDirectory ensures the target directory exists and is ready for extraction
func (e *Extractor) prepareTargetDirectory(targetDir string) error {
	// Check if directory exists
	if _, err := os.Stat(targetDir); err == nil {
		if !e.force {
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
func (e *Extractor) showExtractionSummary(targetDir string) {
	fmt.Printf("\n📊 Extraction Summary:\n")
	fmt.Printf("===================\n")

	err := filepath.Walk(targetDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(targetDir, path)
			fmt.Printf("📝 %s\n", relPath)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  Warning: Failed to generate full extraction summary: %v\n", err)
	}
}

// noopPubSub is a minimal pubsub implementation that does nothing
// and doesn't initialize Watermill
type noopPubSub struct{}

func (n *noopPubSub) Publish(ctx context.Context, msg pubsub.Message) error { 
	return nil 
}

func (n *noopPubSub) Subscribe(ctx context.Context, topic string, handler pubsub.Handler) error {
	return nil
}

func (n *noopPubSub) Close() error { 
	return nil 
}

// noopRenderer is a minimal renderer implementation
type noopRenderer struct{}

func (n *noopRenderer) Render(template string, data interface{}) (string, error) {
	return "", nil
}

func (n *noopRenderer) RenderComponent(ctx context.Context, data interface{}) ([]byte, error) {
	return []byte(""), nil
}

func (n *noopRenderer) RenderPage(ctx echo.Context, code int, data interface{}) error {
	return nil
}

// noopPresenceService is a minimal presence service implementation
type noopPresenceService struct{}

func (n *noopPresenceService) UpdatePresence(ctx context.Context, userID string, status presence.Status) error {
	return nil
}

func (n *noopPresenceService) GetPresence(ctx context.Context, userID string) (presence.Status, error) {
	return presence.StatusOffline, nil
}

func (n *noopPresenceService) ListOnlineUsers(ctx context.Context) ([]string, error) {
	return nil, nil
}
