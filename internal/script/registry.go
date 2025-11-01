package script

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Registry implements the ScriptRegistry interface
type Registry struct {
	mu                sync.RWMutex
	scripts           map[string]map[string]*Script // moduleName -> scriptName -> Script
	embeddedProviders map[string]EmbeddedScriptProvider
	scriptCache       map[string]*Script // cache by module/script key
	watcher           *fsnotify.Watcher
	watcherActive     bool
}

// EmbeddedScriptProvider defines the interface for modules to provide embedded scripts
type EmbeddedScriptProvider interface {
	// GetEmbeddedScripts returns a map of script name to script content
	GetEmbeddedScripts() map[string]string

	// GetModuleName returns the module name for these scripts
	GetModuleName() string
}

// NewRegistry creates a new script registry
func NewRegistry() *Registry {
	return &Registry{
		scripts:           make(map[string]map[string]*Script),
		embeddedProviders: make(map[string]EmbeddedScriptProvider),
		scriptCache:       make(map[string]*Script),
	}
}

// RegisterEmbeddedProvider registers a provider for embedded scripts
func (r *Registry) RegisterEmbeddedProvider(provider EmbeddedScriptProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	moduleName := provider.GetModuleName()
	r.embeddedProviders[moduleName] = provider

	slog.Debug("Registered embedded script provider", "module", moduleName)
}

// LoadScriptsFromProvider loads scripts from a specific provider immediately
func (r *Registry) LoadScriptsFromProvider(provider EmbeddedScriptProvider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	moduleName := provider.GetModuleName()
	return r.loadEmbeddedScriptsFromProvider(moduleName, provider)
}

// LoadScripts discovers and loads all available scripts
func (r *Registry) LoadScripts() error {
	// Load embedded scripts first (with lock)
	r.mu.Lock()
	for moduleName, provider := range r.embeddedProviders {
		if err := r.loadEmbeddedScriptsFromProvider(moduleName, provider); err != nil {
			r.mu.Unlock()
			slog.Error("Failed to load embedded scripts", "module", moduleName, "error", err)
			return fmt.Errorf("failed to load embedded scripts for module %s: %w", moduleName, err)
		}
	}
	r.mu.Unlock()

	slog.Info("Loaded scripts from embedded providers", "modules", len(r.embeddedProviders))

	// Load external scripts (this method handles its own locking)
	if err := r.LoadExternalScripts(); err != nil {
		slog.Error("Failed to load external scripts", "error", err)
		// Don't return error - external scripts are optional
	}

	return nil
}

// GetScript retrieves a script by module and name
func (r *Registry) GetScript(moduleName, scriptName string) (*Script, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check cache first
	cacheKey := r.getCacheKey(moduleName, scriptName)
	if cached, exists := r.scriptCache[cacheKey]; exists {
		return cached, nil
	}

	// Look in module scripts
	if moduleScripts, exists := r.scripts[moduleName]; exists {
		if script, exists := moduleScripts[scriptName]; exists {
			// Cache the script
			r.scriptCache[cacheKey] = script
			return script, nil
		}
	}

	return nil, NewScriptError(
		ErrorTypeNotFound,
		moduleName,
		scriptName,
		fmt.Sprintf("script not found: %s/%s", moduleName, scriptName),
		nil,
	)
}

// ReloadScript reloads a specific script from disk
func (r *Registry) ReloadScript(moduleName, scriptName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Invalidate cache first
	cacheKey := r.getCacheKey(moduleName, scriptName)
	delete(r.scriptCache, cacheKey)

	// Try to load external script
	externalScript, err := r.loadExternalScript(moduleName, scriptName)
	if err != nil {
		slog.Debug("Failed to load external script, keeping embedded version",
			"module", moduleName, "script", scriptName, "error", err)
		return nil // Don't return error, just keep embedded version
	}

	if externalScript != nil {
		// Update the script in registry
		if r.scripts[moduleName] == nil {
			r.scripts[moduleName] = make(map[string]*Script)
		}
		r.scripts[moduleName][scriptName] = externalScript

		slog.Info("Reloaded external script",
			"module", moduleName, "script", scriptName, "language", externalScript.Language)
	}

	return nil
}

// ListScripts returns all available scripts organized by module
func (r *Registry) ListScripts() map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]string)

	for moduleName, moduleScripts := range r.scripts {
		scriptNames := make([]string, 0, len(moduleScripts))
		for scriptName := range moduleScripts {
			scriptNames = append(scriptNames, scriptName)
		}
		result[moduleName] = scriptNames
	}

	return result
}

// StartWatcher begins monitoring external script files for changes
func (r *Registry) StartWatcher(ctx context.Context, enableHotReload bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if hot-reload is disabled
	if !enableHotReload {
		slog.Info("Hot-reload disabled, skipping file system watcher setup")
		return nil
	}

	// Check if watcher is already active
	if r.watcherActive {
		slog.Debug("Script watcher already active")
		return nil
	}

	// Check if scripts directory exists
	scriptsDir := "scripts"
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		slog.Debug("Scripts directory does not exist, skipping watcher setup", "path", scriptsDir)
		return nil
	}

	// Create file system watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create file system watcher: %w", err)
	}

	r.watcher = watcher
	r.watcherActive = true

	// Add scripts directory and all subdirectories to watcher
	err = filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only watch directories
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				slog.Error("Failed to add directory to watcher", "path", path, "error", err)
				return err
			}
			slog.Debug("Added directory to watcher", "path", path)
		}

		return nil
	})

	if err != nil {
		watcher.Close()
		r.watcher = nil
		r.watcherActive = false
		return fmt.Errorf("failed to add directories to watcher: %w", err)
	}

	// Start watching in a goroutine
	go r.watchFiles(ctx)

	slog.Debug("Started file system watcher for script hot-reloading", "directory", scriptsDir)
	return nil
}

// watchFiles handles file system events
func (r *Registry) watchFiles(ctx context.Context) {
	defer func() {
		r.mu.Lock()
		if r.watcher != nil {
			r.watcher.Close()
			r.watcher = nil
		}
		r.watcherActive = false
		r.mu.Unlock()
		slog.Info("File system watcher stopped")
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("File system watcher context cancelled")
			return

		case event, ok := <-r.watcher.Events:
			if !ok {
				slog.Debug("File system watcher events channel closed")
				return
			}

			r.handleFileEvent(event)

		case err, ok := <-r.watcher.Errors:
			if !ok {
				slog.Debug("File system watcher errors channel closed")
				return
			}

			slog.Error("File system watcher error", "error", err)
		}
	}
}

// handleFileEvent processes individual file system events
func (r *Registry) handleFileEvent(event fsnotify.Event) {
	// Only handle script files
	if !r.isScriptFile(event.Name) {
		return
	}

	// Parse module and script name from path
	moduleName, scriptName, err := r.parseScriptPath(event.Name)
	if err != nil {
		slog.Debug("Failed to parse script path", "path", event.Name, "error", err)
		return
	}

	slog.Debug("File system event", "event", event.Op.String(), "path", event.Name, "module", moduleName, "script", scriptName)

	switch {
	case event.Op&fsnotify.Write == fsnotify.Write:
		// File was modified
		r.handleScriptModified(moduleName, scriptName, event.Name)

	case event.Op&fsnotify.Create == fsnotify.Create:
		// File was created
		r.handleScriptCreated(moduleName, scriptName, event.Name)

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		// File was deleted
		r.handleScriptDeleted(moduleName, scriptName, event.Name)

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		// File was renamed (treat as deletion)
		r.handleScriptDeleted(moduleName, scriptName, event.Name)
	}
}

// handleScriptModified handles script file modifications
func (r *Registry) handleScriptModified(moduleName, scriptName, filePath string) {
	slog.Info("Script file modified, reloading", "module", moduleName, "script", scriptName, "path", filePath)

	if err := r.ReloadScript(moduleName, scriptName); err != nil {
		slog.Error("Failed to reload modified script", "module", moduleName, "script", scriptName, "error", err)
	} else {
		slog.Info("Successfully reloaded modified script", "module", moduleName, "script", scriptName)
	}
}

// handleScriptCreated handles new script file creation
func (r *Registry) handleScriptCreated(moduleName, scriptName, filePath string) {
	slog.Info("New script file created, loading", "module", moduleName, "script", scriptName, "path", filePath)

	if err := r.ReloadScript(moduleName, scriptName); err != nil {
		slog.Error("Failed to load new script", "module", moduleName, "script", scriptName, "error", err)
	} else {
		slog.Info("Successfully loaded new script", "module", moduleName, "script", scriptName)
	}
}

// handleScriptDeleted handles script file deletion
func (r *Registry) handleScriptDeleted(moduleName, scriptName, filePath string) {
	slog.Info("Script file deleted, reverting to embedded version", "module", moduleName, "script", scriptName, "path", filePath)

	r.mu.Lock()
	defer r.mu.Unlock()

	// Remove from cache
	cacheKey := r.getCacheKey(moduleName, scriptName)
	delete(r.scriptCache, cacheKey)

	// Check if we have an embedded version to fall back to
	if moduleScripts, exists := r.scripts[moduleName]; exists {
		if script, exists := moduleScripts[scriptName]; exists && script.Source == SourceExternal {
			// Remove the external script entry
			delete(moduleScripts, scriptName)

			// Try to restore embedded version
			if provider, exists := r.embeddedProviders[moduleName]; exists {
				embeddedScripts := provider.GetEmbeddedScripts()
				if embeddedContent, exists := embeddedScripts[scriptName]; exists {
					// Restore embedded script
					language := r.detectLanguage(scriptName, embeddedContent)
					embeddedScript := &Script{
						ModuleName:       moduleName,
						Name:             scriptName,
						Language:         language,
						Content:          embeddedContent,
						Source:           SourceEmbedded,
						LastModified:     time.Now(),
						Checksum:         r.generateChecksum(embeddedContent),
						OriginalLanguage: language,
					}

					moduleScripts[scriptName] = embeddedScript
					slog.Info("Restored embedded script after external deletion", "module", moduleName, "script", scriptName)
				}
			}
		}
	}
}

// isScriptFile checks if a file is a script file based on extension
func (r *Registry) isScriptFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	return ext == ".tengo" || ext == ".zygomys" || ext == ""
}

// parseScriptPath extracts module name and script name from file path
func (r *Registry) parseScriptPath(filePath string) (moduleName, scriptName string, err error) {
	// Convert to relative path from scripts directory
	scriptsDir := "scripts"
	relPath, err := filepath.Rel(scriptsDir, filePath)
	if err != nil {
		return "", "", err
	}

	// Check if the path is actually within the scripts directory
	if strings.HasPrefix(relPath, "..") || filepath.IsAbs(relPath) {
		return "", "", fmt.Errorf("path %s is not within the scripts directory", filePath)
	}

	// Split path components
	pathParts := strings.Split(relPath, string(filepath.Separator))
	if len(pathParts) < 2 {
		return "", "", fmt.Errorf("invalid script path structure: %s", filePath)
	}

	moduleName = pathParts[0]
	filename := pathParts[len(pathParts)-1]

	// Remove extension to get script name
	scriptName = strings.TrimSuffix(filename, filepath.Ext(filename))
	if scriptName == "" {
		return "", "", fmt.Errorf("empty script name from path: %s", filePath)
	}

	return moduleName, scriptName, nil
}

// StopWatcher stops the file system watcher
func (r *Registry) StopWatcher() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.watcher != nil {
		r.watcher.Close()
		r.watcher = nil
		r.watcherActive = false
		slog.Info("File system watcher stopped")
	}
}

// loadExternalScript attempts to load a script from the external scripts directory
func (r *Registry) loadExternalScript(moduleName, scriptName string) (*Script, error) {
	// Try different file extensions for the script
	extensions := []string{".tengo", ".zygomys", ""}

	for _, ext := range extensions {
		filename := scriptName + ext
		scriptPath := filepath.Join("scripts", moduleName, filename)

		// Check if file exists
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			continue
		}

		// Read the file
		content, err := os.ReadFile(scriptPath)
		if err != nil {
			slog.Debug("Failed to read external script file", "path", scriptPath, "error", err)
			continue
		}

		// Determine language from extension
		language := r.detectLanguage(filename, string(content))

		// Get file info for modification time
		fileInfo, err := os.Stat(scriptPath)
		if err != nil {
			slog.Debug("Failed to get file info", "path", scriptPath, "error", err)
			continue
		}

		// Get original language from embedded script if it exists
		originalLanguage := language
		if moduleScripts, exists := r.scripts[moduleName]; exists {
			if embeddedScript, exists := moduleScripts[scriptName]; exists && embeddedScript.Source == SourceEmbedded {
				originalLanguage = embeddedScript.Language
			}
		}

		script := &Script{
			ModuleName:       moduleName,
			Name:             scriptName,
			Language:         language,
			Content:          string(content),
			Source:           SourceExternal,
			LastModified:     fileInfo.ModTime(),
			Checksum:         r.generateChecksum(string(content)),
			OriginalLanguage: originalLanguage,
		}

		slog.Info("Loaded external script",
			"module", moduleName,
			"script", scriptName,
			"path", scriptPath,
			"language", language,
			"size", len(content))

		return script, nil
	}

	return nil, fmt.Errorf("external script not found: %s/%s", moduleName, scriptName)
}

// LoadExternalScripts scans for and loads external scripts from the filesystem
func (r *Registry) LoadExternalScripts() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	scriptsDir := "scripts"

	// Check if scripts directory exists
	if _, err := os.Stat(scriptsDir); os.IsNotExist(err) {
		// Get current working directory for debugging
		if cwd, err := os.Getwd(); err == nil {
			slog.Info("Scripts directory does not exist", "path", scriptsDir, "cwd", cwd, "full_path", filepath.Join(cwd, scriptsDir))
		} else {
			slog.Debug("Scripts directory does not exist", "path", scriptsDir)
		}
		return nil // Not an error, just no external scripts
	}

	// Walk through the scripts directory
	err := filepath.Walk(scriptsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Parse the path to get module and script name
		relPath, err := filepath.Rel(scriptsDir, path)
		if err != nil {
			return err
		}

		pathParts := strings.Split(relPath, string(filepath.Separator))
		if len(pathParts) < 2 {
			return nil // Skip files not in module subdirectories
		}

		moduleName := pathParts[0]
		filename := pathParts[len(pathParts)-1]

		// Extract script name (remove extension)
		scriptName := strings.TrimSuffix(filename, filepath.Ext(filename))
		if scriptName == "" {
			return nil
		}

		// Read the file content
		content, err := os.ReadFile(path)
		if err != nil {
			slog.Error("Failed to read external script", "path", path, "error", err)
			return nil // Continue with other files
		}

		// Determine language
		language := r.detectLanguage(filename, string(content))

		// Get original language from embedded script if it exists
		originalLanguage := language
		if moduleScripts, exists := r.scripts[moduleName]; exists {
			if embeddedScript, exists := moduleScripts[scriptName]; exists && embeddedScript.Source == SourceEmbedded {
				originalLanguage = embeddedScript.Language
			}
		}

		// Create script object
		externalScript := &Script{
			ModuleName:       moduleName,
			Name:             scriptName,
			Language:         language,
			Content:          string(content),
			Source:           SourceExternal,
			LastModified:     info.ModTime(),
			Checksum:         r.generateChecksum(string(content)),
			OriginalLanguage: originalLanguage,
		}

		// Add to registry (this will override embedded scripts)
		if r.scripts[moduleName] == nil {
			r.scripts[moduleName] = make(map[string]*Script)
		}
		r.scripts[moduleName][scriptName] = externalScript

		slog.Debug("Loaded external script",
			"module", moduleName,
			"script", scriptName,
			"language", language,
			"original_language", originalLanguage,
			"size", len(content))

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to load external scripts: %w", err)
	}

	return nil
}

// loadEmbeddedScriptsFromProvider loads scripts from a specific provider
func (r *Registry) loadEmbeddedScriptsFromProvider(moduleName string, provider EmbeddedScriptProvider) error {
	embeddedScripts := provider.GetEmbeddedScripts()

	if r.scripts[moduleName] == nil {
		r.scripts[moduleName] = make(map[string]*Script)
	}

	for scriptName, content := range embeddedScripts {
		// Check if external script already exists - don't overwrite it
		if existingScript, exists := r.scripts[moduleName][scriptName]; exists && existingScript.Source == SourceExternal {
			slog.Debug("Skipping embedded script - external version already loaded",
				"module", moduleName,
				"script", scriptName,
				"external_size", len(existingScript.Content),
				"embedded_size", len(content),
			)
			continue
		}

		// Determine language from script name/content
		language := r.detectLanguage(scriptName, content)

		script := &Script{
			ModuleName:       moduleName,
			Name:             scriptName,
			Language:         language,
			Content:          content,
			Source:           SourceEmbedded,
			LastModified:     time.Now(), // For embedded scripts, use load time
			Checksum:         r.generateChecksum(content),
			OriginalLanguage: language, // Same as language for embedded scripts
		}

		r.scripts[moduleName][scriptName] = script

		slog.Debug("Loaded embedded script",
			"module", moduleName,
			"script", scriptName,
			"language", language,
			"size", len(content),
		)
	}

	return nil
}

// detectLanguage determines the script language from filename or content
func (r *Registry) detectLanguage(scriptName, content string) ScriptLanguage {
	// Check file extension first
	ext := filepath.Ext(scriptName)
	switch ext {
	case ".tengo":
		return LanguageTengo
	case ".zygomys":
		return LanguageZygomys
	}

	// If no extension, try to detect from content
	// Look for Zygomys-specific patterns first (more specific)
	if strings.Contains(content, "defn ") || strings.Contains(content, "(defn ") {
		return LanguageZygomys // Lisp function definition
	}

	// Look for other Zygomys patterns
	if strings.Contains(content, "(let ") || strings.Contains(content, "(if ") {
		return LanguageZygomys // Lisp-like syntax
	}

	// Default to Tengo for embedded scripts and unknown content
	return LanguageTengo
}

// generateChecksum creates a checksum for script content
func (r *Registry) generateChecksum(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// getCacheKey creates a cache key for a module/script combination
func (r *Registry) getCacheKey(moduleName, scriptName string) string {
	return fmt.Sprintf("%s/%s", moduleName, scriptName)
}

// GetScriptMetadata returns metadata about all loaded scripts
func (r *Registry) GetScriptMetadata() map[string]map[string]ScriptMetadata {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]map[string]ScriptMetadata)

	for moduleName, moduleScripts := range r.scripts {
		result[moduleName] = make(map[string]ScriptMetadata)
		for scriptName, script := range moduleScripts {
			result[moduleName][scriptName] = ScriptMetadata{
				Name:         script.Name,
				Language:     script.Language,
				Source:       script.Source,
				LastModified: script.LastModified,
				Checksum:     script.Checksum,
				Size:         len(script.Content),
			}
		}
	}

	return result
}

// ScriptMetadata contains metadata about a script without the content
type ScriptMetadata struct {
	Name         string
	Language     ScriptLanguage
	Source       ScriptSource
	LastModified time.Time
	Checksum     string
	Size         int
}
