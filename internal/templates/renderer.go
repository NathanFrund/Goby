package templates

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"path/filepath"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"
)

// Renderer is a custom html/template renderer for Echo framework
type Renderer struct {
	// We use a map to store templates, with each page having its own isolated template set.
	templates map[string]*template.Template
}

// NewRenderer creates a new Renderer instance
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

	return &Renderer{templates: templates}
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
		// If data is nil, create an empty map to avoid panics, especially for pages
		// that don't pass any data but still need the flash message context.
		data = map[string]interface{}{"Flashes": view.GetFlashes(c)}
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
