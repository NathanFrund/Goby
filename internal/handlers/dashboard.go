package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/models"
)

// DashboardHandler handles requests for the user dashboard.
type DashboardHandler struct{}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler() *DashboardHandler {
	return &DashboardHandler{}
}

// DashboardGet shows the user's dashboard page.
func (h *DashboardHandler) DashboardGet(c echo.Context) error {
	// The Auth middleware has already run and placed the user in the context.
	// We can safely retrieve it.
	user := c.Get(middleware.UserContextKey).(*models.User)

	return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
		"Page": "Dashboard",
		"User": user,
	})
}
