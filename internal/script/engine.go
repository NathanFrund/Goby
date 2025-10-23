package script

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/nfrund/goby/internal/config"
)

// Engine implements the ScriptEngine interface
type Engine struct {
	registry       ScriptRegistry
	factory        EngineFactory
	config         config.Provider
	securityLimits SecurityLimits
	errorReporter  *ErrorReporter
}

// Dependencies holds all the services that the Engine requires to operate
type Dependencies struct {
	Config config.Provider
}

// NewEngine creates a new script engine with the given dependencies
func NewEngine(deps Dependencies) *Engine {
	return &Engine{
		registry:       NewRegistry(),
		factory:        NewFactory(),
		config:         deps.Config,
		securityLimits: GetDefaultSecurityLimits(),
		errorReporter:  NewErrorReporter(),
	}
}

// Initialize sets up the script engine and loads embedded scripts
func (e *Engine) Initialize(ctx context.Context) error {
	slog.Info("Initializing script engine")

	// Load all embedded scripts
	if err := e.registry.LoadScripts(); err != nil {
		return fmt.Errorf("failed to load scripts: %w", err)
	}

	// Start file system watcher for hot-reloading
	if err := e.registry.StartWatcher(ctx); err != nil {
		slog.Error("Failed to start file system watcher", "error", err)
		// Don't fail initialization if watcher fails to start
	}

	// Log loaded scripts
	scripts := e.registry.ListScripts()
	totalScripts := 0
	for module, scriptList := range scripts {
		totalScripts += len(scriptList)
		slog.Debug("Loaded scripts for module", "module", module, "count", len(scriptList))
	}

	slog.Info("Script engine initialized", "total_scripts", totalScripts, "modules", len(scripts))
	return nil
}

// RegisterEmbeddedProvider registers a provider for embedded scripts
func (e *Engine) RegisterEmbeddedProvider(provider EmbeddedScriptProvider) {
	if registry, ok := e.registry.(*Registry); ok {
		registry.RegisterEmbeddedProvider(provider)

		// Immediately load scripts from this provider
		if err := registry.LoadScriptsFromProvider(provider); err != nil {
			slog.Error("Failed to load scripts from provider",
				"module", provider.GetModuleName(),
				"error", err)
		} else {
			slog.Debug("Loaded embedded scripts from provider",
				"module", provider.GetModuleName())
		}
	}
}

// Execute runs a script with the given context and returns results
func (e *Engine) Execute(ctx context.Context, req ExecutionRequest) (*ScriptOutput, error) {
	// Get the script
	script, err := e.GetScript(req.ModuleName, req.ScriptName)
	if err != nil {
		if scriptErr, ok := err.(*ScriptError); ok {
			e.errorReporter.ReportError(ctx, scriptErr, nil)
		}
		return nil, err
	}

	// Create the appropriate language engine
	langEngine, err := e.factory.CreateEngine(script.Language)
	if err != nil {
		scriptErr := NewScriptError(
			ErrorTypeExecution,
			req.ModuleName,
			req.ScriptName,
			"failed to create language engine",
			err,
		)
		e.errorReporter.ReportError(ctx, scriptErr, nil)
		return nil, scriptErr
	}

	// Apply security limits
	limits := e.securityLimits
	if req.SecurityLimits.MaxExecutionTime > 0 {
		limits = req.SecurityLimits
	}
	if err := langEngine.SetSecurityLimits(limits); err != nil {
		scriptErr := NewScriptError(
			ErrorTypeExecution,
			req.ModuleName,
			req.ScriptName,
			"failed to set security limits",
			err,
		)
		e.errorReporter.ReportError(ctx, scriptErr, nil)
		return nil, scriptErr
	}

	// Compile the script
	compiled, err := langEngine.Compile(script)
	if err != nil {
		if scriptErr, ok := err.(*ScriptError); ok {
			e.errorReporter.ReportError(ctx, scriptErr, nil)
		}
		return nil, err
	}

	// Execute the script
	output, err := langEngine.Execute(ctx, compiled, req.Input)
	if err != nil {
		if scriptErr, ok := err.(*ScriptError); ok {
			e.errorReporter.ReportError(ctx, scriptErr, nil)
		}
		return nil, err
	}

	// Log execution success with performance metrics
	LogExecution(slog.LevelDebug, "Script executed successfully", req.ModuleName, req.ScriptName,
		slog.String("language", string(script.Language)),
		slog.Duration("execution_time", output.Metrics.ExecutionTime),
	)

	// Log performance metrics separately for monitoring
	LogPerformance(req.ModuleName, req.ScriptName, output.Metrics)

	return output, nil
}

// GetScript retrieves a script by module and name
func (e *Engine) GetScript(moduleName, scriptName string) (*Script, error) {
	return e.registry.GetScript(moduleName, scriptName)
}

// ExtractDefaultScripts writes embedded scripts to filesystem
func (e *Engine) ExtractDefaultScripts(targetDir string) error {
	slog.Info("Extracting default scripts", "target_dir", targetDir)

	scripts := e.registry.ListScripts()
	extractedCount := 0

	for moduleName, scriptNames := range scripts {
		moduleDir := filepath.Join(targetDir, moduleName)

		// Create module directory
		if err := os.MkdirAll(moduleDir, 0755); err != nil {
			return fmt.Errorf("failed to create module directory %s: %w", moduleDir, err)
		}

		for _, scriptName := range scriptNames {
			script, err := e.registry.GetScript(moduleName, scriptName)
			if err != nil {
				slog.Warn("Failed to get script for extraction", "module", moduleName, "script", scriptName, "error", err)
				continue
			}

			// Only extract embedded scripts
			if script.Source != SourceEmbedded {
				continue
			}

			// Determine file extension based on language
			var filename string
			switch script.Language {
			case LanguageTengo:
				filename = scriptName + ".tengo"
			case LanguageZygomys:
				filename = scriptName + ".zygomys"
			default:
				filename = scriptName
			}

			filePath := filepath.Join(moduleDir, filename)

			// Check if file already exists
			if _, err := os.Stat(filePath); err == nil {
				slog.Debug("Skipping existing file", "file", filePath)
				continue
			}

			// Write script content to file
			if err := os.WriteFile(filePath, []byte(script.Content), 0644); err != nil {
				return fmt.Errorf("failed to write script file %s: %w", filePath, err)
			}

			extractedCount++
			slog.Debug("Extracted script", "file", filePath, "language", script.Language)
		}
	}

	slog.Info("Script extraction completed", "extracted_count", extractedCount, "target_dir", targetDir)
	return nil
}

// Shutdown gracefully stops the engine and cleans up resources
func (e *Engine) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down script engine")

	// Stop the file system watcher
	if registry, ok := e.registry.(*Registry); ok {
		registry.StopWatcher()
	}

	slog.Info("Script engine shutdown complete")
	return nil
}

// GetScriptMetadata returns metadata about all loaded scripts
func (e *Engine) GetScriptMetadata() map[string]map[string]ScriptMetadata {
	if registry, ok := e.registry.(*Registry); ok {
		return registry.GetScriptMetadata()
	}
	return make(map[string]map[string]ScriptMetadata)
}

// GetSupportedLanguages returns all supported script languages
func (e *Engine) GetSupportedLanguages() []ScriptLanguage {
	return e.factory.SupportedLanguages()
}

// SetSecurityLimits updates the default security limits for the engine
func (e *Engine) SetSecurityLimits(limits SecurityLimits) {
	e.securityLimits = limits
	slog.Debug("Updated default security limits",
		"max_execution_time", limits.MaxExecutionTime,
		"max_memory_bytes", limits.MaxMemoryBytes,
	)
}

// GetErrorSummary returns aggregated error statistics
func (e *Engine) GetErrorSummary() *ErrorSummary {
	return e.errorReporter.GetErrorSummary()
}

// ClearErrorHistory clears error tracking history
func (e *Engine) ClearErrorHistory() {
	e.errorReporter.ClearErrorHistory()
}

// SetRecoveryPolicy updates the error recovery policy
func (e *Engine) SetRecoveryPolicy(policy RecoveryPolicy) {
	e.errorReporter.SetRecoveryPolicy(policy)
}
