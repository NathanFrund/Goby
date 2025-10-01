package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/view/dto/dashboard" // DTO
	"github.com/nfrund/goby/web/src/templates/layouts"   // Layout
	"github.com/nfrund/goby/web/src/templates/pages"     // Page Component
	"github.com/nfrund/goby/web/src/templates/partials"  // Partials
)

// DashboardHandler handles requests for the user dashboard.
type DashboardHandler struct{}

// NewDashboardHandler creates a new DashboardHandler without dependencies.
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// DashboardGet retrieves the authenticated user from the context and renders the Dashboard page.
func (h *DashboardHandler) DashboardGet(c echo.Context) error {
	// 1. Retrieve the authenticated user from the context (using existing middleware logic).
	userVal := c.Get(middleware.UserContextKey)

	if userVal == nil {
		c.Logger().Warn("unauthenticated access attempt on protected dashboard")
		// Redirect to login if unauthenticated (assuming your auth logic handles this)
		return c.Redirect(http.StatusFound, "/auth/login")
	}

	user, ok := userVal.(*domain.User)
	if !ok {
		c.Logger().Error("failed to assert context user to *domain.User")
		return c.String(http.StatusInternalServerError, "Authentication context error.")
	}

	// 2. Map the complex domain.User object to the simple view.DashboardData DTO.
	var idString string
	// Check if the SurrealDB ID is present and convert it to a string for display.
	if user.ID != nil {
		idString = user.ID.String()
	}

	data := dashboard.Data{
		ID:    idString,
		Email: user.Email,
	}

	// 3. Instantiate and render the final Templ component.

	pageContent := pages.Dashboard(data)
	flashData := partials.FlashData{}

	finalComponent := layouts.Base("Dashboard", flashData, pageContent)

	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	c.Response().WriteHeader(http.StatusOK)

	// Use the component's Render method to stream the HTML output.
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
}
