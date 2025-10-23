package script

import (
	"context"
)

// ScriptEngine provides the main interface for script execution
type ScriptEngine interface {
	// Execute runs a script with the given context and returns results
	Execute(ctx context.Context, req ExecutionRequest) (*ScriptOutput, error)

	// GetScript retrieves a script by module and name
	GetScript(moduleName, scriptName string) (*Script, error)

	// ExtractDefaultScripts writes embedded scripts to filesystem
	ExtractDefaultScripts(targetDir string) error

	// Shutdown gracefully stops the engine and cleans up resources
	Shutdown(ctx context.Context) error
}

// ScriptRegistry manages script discovery, loading, and hot-reloading
type ScriptRegistry interface {
	// LoadScripts discovers and loads all available scripts
	LoadScripts() error

	// GetScript retrieves a script by module and name
	GetScript(moduleName, scriptName string) (*Script, error)

	// ReloadScript reloads a specific script from disk
	ReloadScript(moduleName, scriptName string) error

	// ListScripts returns all available scripts organized by module
	ListScripts() map[string][]string

	// StartWatcher begins monitoring external script files for changes
	StartWatcher(ctx context.Context) error
}

// EngineFactory creates language-specific script engines
type EngineFactory interface {
	// CreateEngine returns an engine for the specified language
	CreateEngine(language ScriptLanguage) (LanguageEngine, error)

	// SupportedLanguages returns all supported script languages
	SupportedLanguages() []ScriptLanguage
}

// LanguageEngine executes scripts in a specific language
type LanguageEngine interface {
	// Compile prepares a script for execution
	Compile(script *Script) (*CompiledScript, error)

	// Execute runs a compiled script with context
	Execute(ctx context.Context, compiled *CompiledScript, input *ScriptInput) (*ScriptOutput, error)

	// SetSecurityLimits configures resource and security constraints
	SetSecurityLimits(limits SecurityLimits) error
}
