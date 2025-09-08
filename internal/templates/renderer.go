package templates

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
)

// Renderer is a custom html/template renderer for Echo framework
type Renderer struct {
	// We use a map to store templates, with each page having its own isolated template set.
	templates map[string]*template.Template
}

// NewRenderer creates a new Renderer instance
func NewRenderer(path string) *Renderer {
	templates := make(map[string]*template.Template)

	// Find all layout files, which will be shared across all pages.
	layouts, err := filepath.Glob(filepath.Join(path, "layouts", "*.html"))
	if err != nil {
		log.Fatalf("could not glob layouts: %v", err)
	}

	// Find all include/partial files, also shared.
	includes, err := filepath.Glob(filepath.Join(path, "includes", "*.html"))
	if err != nil {
		log.Fatalf("could not glob includes: %v", err)
	}

	// Walk the "pages" directory. For each page, create a new template set
	// that includes the page itself, the layouts, and the includes.
	// This isolates each page's `{{define}}` blocks.
	err = filepath.WalkDir(filepath.Join(path, "pages"), func(pagePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() && strings.HasSuffix(d.Name(), ".html") {
			filesToParse := append([]string{pagePath}, layouts...)
			filesToParse = append(filesToParse, includes...)
			templates[d.Name()] = template.Must(template.ParseFiles(filesToParse...))
		}
		return nil
	})

	if err != nil {
		log.Fatalf("could not walk pages directory: %v", err)
	}

	return &Renderer{templates: templates}
}

// Render renders a template document
func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}
	return tmpl.ExecuteTemplate(w, name, data)
}
