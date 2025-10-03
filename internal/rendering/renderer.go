package rendering

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
)

// --- Universal Renderer Implementation ---

// Renderer defines the contract for rendering any supported component (templ, gomponents, etc.).
// It uses interface{} for the component input to support heterogeneous types.
type Renderer interface {
	// RenderComponent renders a component to a slice of bytes. Useful for HTMX fragments or WebSockets.
	RenderComponent(ctx context.Context, component interface{}) ([]byte, error)

	// RenderPage handles full-page rendering for Echo's context.Render() method.
	RenderPage(c echo.Context, status int, component interface{}) error
}

// UniversalRenderer is the concrete implementation that handles rendering for multiple component types.
type UniversalRenderer struct{}

// NewUniversalRenderer creates a new UniversalRenderer instance.
func NewUniversalRenderer() *UniversalRenderer {
	return &UniversalRenderer{}
}

// gomponentNode defines the structural interface for gomponents.Node,
// which typically only requires an io.Writer.
type gomponentNode interface {
	Render(w io.Writer) error
}

// render is the core logic that inspects the component type and calls the appropriate render method.
func (tr *UniversalRenderer) render(ctx context.Context, component interface{}, w io.Writer) error {
	switch c := component.(type) {
	case templ.Component:
		// Case 1: Handle templ components (requires context and writer)
		return c.Render(ctx, w)

	case gomponentNode:
		// Case 2: Handle types that implement Render(io.Writer) error (like gomponents.Node).
		// IMPORTANT: If you are using gomponents, ensure the actual package is imported in your project.
		return c.Render(w)

	default:
		return fmt.Errorf("unsupported component type: %T. Component must be templ.Component or implement Render(io.Writer) error (like gomponents.Node)", component)
	}
}

// RenderComponent implements the Renderer interface.
func (tr *UniversalRenderer) RenderComponent(ctx context.Context, component interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := tr.render(ctx, component, &buf); err != nil {
		return nil, fmt.Errorf("failed to render component to bytes: %w", err)
	}
	return buf.Bytes(), nil
}

// RenderPage implements the Renderer interface for full HTTP responses.
func (tr *UniversalRenderer) RenderPage(c echo.Context, status int, component interface{}) error {
	c.Response().Writer.WriteHeader(status)
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)

	// Use the context from the Echo request for rendering
	if err := tr.render(c.Request().Context(), component, c.Response().Writer); err != nil {
		c.Logger().Error("Failed to stream component to response writer:", err)
		return err
	}
	return nil
}

// Render implements the echo.Renderer interface for use with c.Render(status, name, component).
func (tr *UniversalRenderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// For component-based rendering, the component object is passed in the 'data' parameter.
	if c.Response().Header().Get(echo.HeaderContentType) == "" {
		c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTML)
	}

	// Use the core rendering logic
	return tr.render(c.Request().Context(), data, w)
}
