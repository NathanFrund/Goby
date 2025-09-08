package templates

import (
	"html/template"
	"io"
	"path/filepath"

	"github.com/labstack/echo/v4"
)

type Template struct {
	templates map[string]*template.Template
}

func New() *Template {
	return &Template{
		templates: make(map[string]*template.Template),
	}
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		// Parse template if not in cache
		path := filepath.Join("web/templates", name+".html")
		var err error
		tmpl, err = template.ParseFiles(
			"web/templates/base.html",
			path,
		)
		if err != nil {
			return err
		}
		t.templates[name] = tmpl
	}

	// Add global template data
	templateData, ok := data.(map[string]interface{})
	if !ok {
		templateData = make(map[string]interface{})
	}

	// Add CSRF token if available
	if csrf := c.Get("csrf"); csrf != nil {
		templateData["CSRFToken"] = csrf
	}

	return tmpl.ExecuteTemplate(w, "base.html", templateData)
}
