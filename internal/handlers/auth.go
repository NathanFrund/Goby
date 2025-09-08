package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	// In the future, we would add dependencies here, like a user service.
	// userService *user.Service
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler() *AuthHandler {
	return &AuthHandler{}
}

// RegisterGet handles the request to show the registration page.
func (h *AuthHandler) RegisterGet(c echo.Context) error {
	// This handler's only job is to render the registration page template.
	// The template name "pages/register" corresponds to the file
	// "web/src/templates/pages/register.html".
	return c.Render(http.StatusOK, "pages/register", nil)
}

// RegisterPost handles the form submission for creating a new user.
func (h *AuthHandler) RegisterPost(c echo.Context) error {
	// This is a placeholder for now. We will add logic to create a user here.
	email := c.FormValue("email")
	return c.String(http.StatusOK, "Account creation request received for: "+email)
}
