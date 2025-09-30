package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"

	// --- TEMPL IMPORTS ---
	// Need the packages where Base and Home components live
	"github.com/nfrund/goby/web/src/templates/layouts"
	"github.com/nfrund/goby/web/src/templates/pages"
	"github.com/nfrund/goby/web/src/templates/partials" // Required by the layouts.Base component signature
)

// HomeHandler handles requests for the home page.
type HomeHandler struct{}

// NewHomeHandler creates a new HomeHandler.
func NewHomeHandler() *HomeHandler {
	return &HomeHandler{}
}

// HomeGet handles the GET request for the home page.
func (h *HomeHandler) HomeGet(c echo.Context) error {
	// Check if the user is authenticated by looking for the auth cookie.
	cookie, err := c.Cookie("auth_token")
	isAuthenticated := err == nil && cookie.Value != ""

	// 1. Instantiate the inner page content component (pages.Home).
	pageContent := pages.Home(isAuthenticated)

	// 2. Wrap the inner page in the Base layout component.
	// We use a placeholder for FlashData since we don't have the logic to retrieve it yet.
	flashData := partials.FlashData{}

	finalComponent := layouts.Base("Home", flashData, pageContent)

	// 3. Render the final component directly to the response writer.
	// This replaces the old c.Render(status, name, data) call.

	// FIX: Use the standard string "text/html" as a reliable workaround for the
	// potentially missing constant in the specific echo version being used.
	c.Response().Header().Set(echo.HeaderContentType, "text/html")
	c.Response().WriteHeader(http.StatusOK)

	// Use the component's Render method to stream the HTML output.
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
}
