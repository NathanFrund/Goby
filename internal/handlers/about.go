package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
	"github.com/nfrund/goby/web/src/templates/pages"
)

// AboutGet is a handler function that renders the about page.
func AboutGet(c echo.Context) error {
	// 1. Get the Gomponents content (which returns a gomponents.Node).
	gomponentsContent := pages.AboutContent()

	// 2. Wrap the Gomponents content to make it compatible with the Templ layout.
	pageContent := view.AdaptGomponentToTempl(gomponentsContent)

	// 3. Retrieve flash data from the session.
	flashData := view.GetFlashData(c)

	// 4. Wrap the inner page in the Base layout, passing the flash data.
	finalComponent := layouts.Base("About", flashData.Messages, pageContent)

	// 5. Render the final component.
	return c.Render(http.StatusOK, "", finalComponent)
}
