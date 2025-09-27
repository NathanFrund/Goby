package templateregistry

import (
	"fmt"
	"io"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"
)

// Renderer implements the echo.Renderer interface using our template registry.
type Renderer struct {
	registry *Registry
}

// NewRenderer creates a new renderer instance.
func NewRenderer(registry *Registry) *Renderer {
	return &Renderer{registry: registry}
}

// Render looks up a template by name in the registry and executes it.
func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := r.registry.Get(name)
	if !ok {
		return fmt.Errorf("template not found: %s", name)
	}

	// Prepare data for the template, injecting flash messages.
	// This logic is adapted from the old renderer to ensure compatibility.
	if dataMap, ok := data.(map[string]interface{}); ok {
		if c != nil {
			dataMap["Flashes"] = view.GetFlashes(c)
		}
	} else if data == nil && c != nil {
		data = map[string]interface{}{"Flashes": view.GetFlashes(c)}
	}

	return tmpl.Execute(w, data)
}
