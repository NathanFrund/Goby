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
	return c.Render(http.StatusOK, "home.html", nil)
}
