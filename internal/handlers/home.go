package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
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
	// A more robust implementation would validate the token, but this is sufficient for UI purposes.
	cookie, err := c.Cookie("auth_token")
	isAuthenticated := err == nil && cookie.Value != ""

	return c.Render(http.StatusOK, "home.html", map[string]interface{}{
		"IsAuthenticated": isAuthenticated,
	})
}
