package handlers

import (
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"

	"github.com/nfrund/goby/web/src/templates/layouts"
	"github.com/nfrund/goby/web/src/templates/pages"
	"github.com/nfrund/goby/web/src/templates/partials"
)

// AboutHandler handles requests for the about page.
type AboutHandler struct{}

// HandleGet renders the About page using Gomponents content wrapped by the Templ base layout.
func (h *AboutHandler) HandleGet(c echo.Context) error {
	// 1. Get the Gomponents content (which returns a gomponents.Node).
	gomponentsContent := pages.AboutContent()

	// 2. Wrap the Gomponents content using view.Adapt().
	// This uses the GomponentAdapter to satisfy the templ.Component interface.
	pageContent := view.AdaptGomponentToTempl(gomponentsContent)

	// 3. Pass the wrapped content to the main Templ Base layout.
	// This ensures the Gomponents content is rendered within your unified Templ shell.
	page := layouts.Base("About Us", partials.FlashData{}, pageContent)

	// 4. Render the full page using Templ's Render method.
	return page.Render(c.Request().Context(), c.Response().Writer)
}
