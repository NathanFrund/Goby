package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/nfrund/goby/internal/view"
	// --- TEMPL IMPORTS ---
	// Need the packages where Base and Home components live
	"github.com/nfrund/goby/web/src/templates/layouts"
	"github.com/nfrund/goby/web/src/templates/pages"
)

// HomeGet is a handler function that renders the home page.
func HomeGet(c echo.Context) error {
	// Check if the user is authenticated by looking for the auth cookie.
	cookie, err := c.Cookie("auth_token")
	isAuthenticated := err == nil && cookie.Value != ""

	// 1. Instantiate the inner page content component (pages.Home).
	pageContent := pages.Home(isAuthenticated)

	// 2. Retrieve flash data from the session. This consumes the messages.
	flashData := view.GetFlashData(c)

	// 3. Wrap the inner page in the Base layout, passing the flash data.
	finalComponent := layouts.Base("Home", flashData.Messages, pageContent)

	// The 'name' parameter is ignored by our renderer, but the component is passed as 'data'.
	return c.Render(http.StatusOK, "", finalComponent)
}
