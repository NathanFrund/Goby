package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"
)

// Renderer is a custom html/template renderer for Echo framework
type Renderer struct {
	// We use a map to store templates, with each page having its own isolated template set.
	templates map[string]*template.Template
	// Keep track of base layout and standalone templates so we can compose new pages from other dirs.
	layouts    []string
	standalone []string
}

// NewRenderer creates a new Renderer instance from disk paths
func NewRenderer(path string) *Renderer {
	templates := make(map[string]*template.Template)

	// Find all base and partial templates
	layouts, err := filepath.Glob(filepath.Join(path, "layouts", "*.html"))
	if err != nil {
		log.Fatalf("could not glob base template: %v", err)
	}
	partials, err := filepath.Glob(filepath.Join(path, "partials", "*.html"))
	if err != nil {
		log.Fatalf("could not glob partials: %v", err)
	}
	components, err := filepath.Glob(filepath.Join(path, "components", "*.html"))
	if err != nil {
		log.Fatalf("could not glob components: %v", err)
	}
	standaloneTemplates := append(partials, components...)

	// Find all page templates
	pages, err := filepath.Glob(filepath.Join(path, "pages", "*.html"))
	if err != nil {
		log.Fatalf("could not glob page templates: %v", err)
	}

	// For each page, parse it with the base and partials
	for _, page := range pages {
		files := append(layouts, standaloneTemplates...)
		files = append(files, page)
		templates[filepath.Base(page)] = template.Must(template.ParseFiles(files...))
	}

	// Also, parse each partial as a standalone template. This allows them to be
	// rendered individually, which is useful for htmx partial updates.
	for _, standalone := range standaloneTemplates {
		templates[filepath.Base(standalone)] = template.Must(template.ParseFiles(standalone))
	}

	return &Renderer{templates: templates, layouts: layouts, standalone: standaloneTemplates}
}

// NewRendererFromFS creates a new Renderer instance using an embedded filesystem root
// The root should be the directory that contains "layouts", "partials", "components", and "pages" subdirectories.
func NewRendererFromFS(root fs.FS, rootPath string) *Renderer {
	templates := make(map[string]*template.Template)

	// Find all base and partial templates
	layouts, err := fs.Glob(root, filepath.Join(rootPath, "layouts", "*.html"))
	if err != nil {
		log.Fatalf("could not glob base template (fs): %v", err)
	}
	partials, err := fs.Glob(root, filepath.Join(rootPath, "partials", "*.html"))
	if err != nil {
		log.Fatalf("could not glob partials (fs): %v", err)
	}
	components, err := fs.Glob(root, filepath.Join(rootPath, "components", "*.html"))
	if err != nil {
		log.Fatalf("could not glob components (fs): %v", err)
	}
	standaloneTemplates := append(partials, components...)

	// Find all page templates
	pages, err := fs.Glob(root, filepath.Join(rootPath, "pages", "*.html"))
	if err != nil {
		log.Fatalf("could not glob page templates (fs): %v", err)
	}

	// For each page, parse it with the base and partials using ParseFS
	for _, page := range pages {
		patterns := append([]string{}, layouts...)
		patterns = append(patterns, standaloneTemplates...)
		patterns = append(patterns, page)
		templates[filepath.Base(page)] = template.Must(template.ParseFS(root, patterns...))
	}

	// Also parse each standalone partial/component as its own template set.
	for _, standalone := range standaloneTemplates {
		templates[filepath.Base(standalone)] = template.Must(template.ParseFS(root, standalone))
	}

	// Save only the patterns for layouts/standalones relative to rootPath; they will be used when composing
	return &Renderer{templates: templates, layouts: layouts, standalone: standaloneTemplates}
}

// AddPagesFrom parses page templates found under the given directory (expects a "pages/*.html" subtree)
// and registers them into the renderer. If namespace is non-empty, the template names are stored as
// "namespace/<basename>" to avoid collisions. Pages are composed with the renderer's base layouts and partials.
func (t *Renderer) AddPagesFrom(path string, namespace string) error {
	pages, err := filepath.Glob(filepath.Join(path, "pages", "*.html"))
	if err != nil {
		return fmt.Errorf("could not glob page templates from %s: %w", path, err)
	}
	for _, page := range pages {
		files := append([]string{}, t.layouts...)
		files = append(files, t.standalone...)
		files = append(files, page)
		set := template.Must(template.ParseFiles(files...))
		name := filepath.Base(page)
		if namespace != "" {
			name = filepath.Join(namespace, name)
		}
		t.templates[name] = set
	}
	return nil
}

// AddPagesFromFS is the FS equivalent of AddPagesFrom. The root should be the directory that contains a "pages" subdirectory.
func (t *Renderer) AddPagesFromFS(root fs.FS, rootPath string, namespace string) error {
	pages, err := fs.Glob(root, filepath.Join(rootPath, "pages", "*.html"))
	if err != nil {
		return fmt.Errorf("could not glob page templates from fs at %s: %w", rootPath, err)
	}
	for _, page := range pages {
		patterns := append([]string{}, t.layouts...)
		patterns = append(patterns, t.standalone...)
		patterns = append(patterns, page)
		set := template.Must(template.ParseFS(root, patterns...))
		name := filepath.Base(page)
		if namespace != "" {
			name = filepath.Join(namespace, name)
		}
		t.templates[name] = set
	}
	return nil
}

// AddStandaloneFrom parses standalone templates (partials/components) found directly under the given directory
// (expects "*.html") and registers them. If namespace is non-empty, names are stored as "namespace/<basename>".
// Note: standalone templates are parsed as-is without composing with base layouts, so they can be rendered directly.
func (t *Renderer) AddStandaloneFrom(path string, namespace string) error {
	items, err := filepath.Glob(filepath.Join(path, "*.html"))
	if err != nil {
		return fmt.Errorf("could not glob standalone templates from %s: %w", path, err)
	}
	for _, item := range items {
		set := template.Must(template.ParseFiles(item))
		name := filepath.Base(item)
		if namespace != "" {
			name = filepath.Join(namespace, name)
		}
		t.templates[name] = set
	}
	return nil
}

// AddStandaloneFromFS is the FS equivalent of AddStandaloneFrom. The dir should be relative to the provided FS root.
func (t *Renderer) AddStandaloneFromFS(root fs.FS, dir string, namespace string) error {
	items, err := fs.Glob(root, filepath.Join(dir, "*.html"))
	if err != nil {
		return fmt.Errorf("could not glob standalone templates from fs at %s: %w", dir, err)
	}
	for _, item := range items {
		set := template.Must(template.ParseFS(root, item))
		name := filepath.Base(item)
		if namespace != "" {
			name = filepath.Join(namespace, name)
		}
		t.templates[name] = set
	}
	return nil
}

// Render renders a template document
func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// For full page renders, we expect data to be a map so we can add flash messages.
	// For partial/component renders (like from a WebSocket), data might be a struct.
	// We only modify the data if it's a map.
	if dataMap, ok := data.(map[string]interface{}); ok {
		if c != nil {
			dataMap["Flashes"] = view.GetFlashes(c)
		}
	} else if data == nil {
		if c != nil {
			data = map[string]interface{}{"Flashes": view.GetFlashes(c)}
		} else {
			data = map[string]interface{}{"Flashes": nil}
		}
	}

	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	// If the template set does not contain a "base.html" definition, it's a
	// standalone partial/component. Execute it directly.
	if tmpl.Lookup("base.html") == nil {
		return tmpl.Execute(w, data) // This is a partial, execute it directly.
	}
	// Otherwise, it's a full page, so execute it within the base layout.
	return tmpl.ExecuteTemplate(w, "base.html", data) // This is a full page, execute the layout.
}
