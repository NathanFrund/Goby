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
	layouts, err := filepath.Glob(filepath.Join(path, "base.html"))
	if err != nil {
		log.Fatalf("could not glob base template: %v", err)
	}
	partials, err := filepath.Glob(filepath.Join(path, "partials", "*.html"))
	if err != nil {
		log.Fatalf("could not glob partials: %v", err)
	}

	// Find all page templates
	pages, err := filepath.Glob(filepath.Join(path, "pages", "*.html"))
	if err != nil {
		log.Fatalf("could not glob page templates: %v", err)
	}

	// For each page, parse it with the base and partials
	for _, page := range pages {
		files := append(layouts, partials...)
		files = append(files, page)
		templates[filepath.Base(page)] = template.Must(template.ParseFiles(files...))
	}

	return &Renderer{templates: templates}
}

// Render renders a template document
func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// Add flash messages to the data map for every render.
	dataMap, _ := data.(map[string]interface{})
	if dataMap == nil {
		dataMap = make(map[string]interface{})
	}
	dataMap["Flashes"] = view.GetFlashes(c)

	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}
	return tmpl.ExecuteTemplate(w, "base.html", dataMap)
}
