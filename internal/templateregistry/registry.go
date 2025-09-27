package templateregistry

import (
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Registry holds all parsed templates for the application.
type Registry struct {
	templates *template.Template
	mu        sync.RWMutex
}

// Initialize creates a new Registry and loads the base templates.
func Initialize(diskPath string, embedFS fs.FS) (*Registry, error) {
	reg := &Registry{
		templates: template.New("").Funcs(template.FuncMap{
			// Add global template functions here if needed.
		}),
	}

	var templateFS fs.FS
	templateMode := os.Getenv("APP_TEMPLATES")

	if templateMode == "embed" {
		slog.Info("Initializing templates from embedded FS")
		subFS, err := fs.Sub(embedFS, "src/templates")
		if err != nil {
			return nil, fmt.Errorf("failed to create sub-filesystem for embedded templates: %w", err)
		}
		templateFS = subFS
	} else {
		slog.Info("Initializing templates from disk", "path", diskPath)
		templateFS = os.DirFS(diskPath)
	}

	// Parse all base templates into the single template set.
	err := reg.AddFS(templateFS, "**/*.html")
	if err != nil {
		return nil, fmt.Errorf("failed to parse base templates: %w", err)
	}

	return reg, nil
}

// AddModule loads all templates from a module's filesystem, namespacing them.
func (r *Registry) AddModule(name string, moduleFS fs.FS) error {
	// We assume module templates are in a "templates" subdirectory.
	subFS, err := fs.Sub(moduleFS, "templates")
	if err != nil {
		// If "templates" dir doesn't exist, just skip.
		slog.Debug("no 'templates' directory found in module, skipping", "module", name)
		return nil
	}

	// To correctly namespace module templates, we create a new template set
	// that is associated with the main one, but has a unique name for each template.
	// We must lock during this entire process to prevent concurrent modifications.
	r.mu.Lock()
	defer r.mu.Unlock()

	// By using ParseFS, we efficiently parse all module templates.
	// The trick is that we need to rename them to include the module namespace.
	// We can achieve this by creating a new template associated with the main one.
	// Note: This approach assumes module templates don't rely on `{{block}}` from the main layout,
	// but are standalone components, which matches your current usage.
	return r.parseModuleFS(subFS, name)
}

// AddFS parses all templates matching the patterns from the given filesystem.
func (r *Registry) AddFS(templateFS fs.FS, patterns ...string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// ParseFS adds the parsed templates to the existing template set.
	tmpl, err := r.templates.ParseFS(templateFS, patterns...)
	if err != nil {
		return err
	}
	r.templates = tmpl
	return nil
}

// Get retrieves a specific template definition by its name.
func (r *Registry) Get(name string) (*template.Template, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// All templates are part of the same set, so we just look it up by name.
	tmpl := r.templates.Lookup(name)
	return tmpl, tmpl != nil
}

// parseModuleFS walks the module filesystem, reads each file, and parses it
// into the main template set with a namespaced name. This must be called
// within a write lock.
func (r *Registry) parseModuleFS(moduleFS fs.FS, namespace string) error {
	return fs.WalkDir(moduleFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".html") {
			return err
		}

		content, err := fs.ReadFile(moduleFS, path)
		if err != nil {
			return fmt.Errorf("could not read module template %s: %w", path, err)
		}

		// Create a namespaced name, e.g., "wargame/components/damage.html"
		templateName := filepath.ToSlash(filepath.Join(namespace, path))
		slog.Debug("Registering module template", "name", templateName)

		// Create a new template definition associated with the main set.
		_, err = r.templates.New(templateName).Parse(string(content))
		return err
	})
}
