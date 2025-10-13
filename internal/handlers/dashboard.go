package handlers

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/internal/view/dto/dashboard" // DTO
	"github.com/nfrund/goby/web/src/templates/layouts"   // Layout
	"github.com/nfrund/goby/web/src/templates/pages"     // Page Component
	// Partials
)

// DashboardHandler handles requests for the user dashboard.
type DashboardHandler struct {
	fileRepo domain.FileRepository
}

// NewDashboardHandler creates a new DashboardHandler.
func NewDashboardHandler(fileRepo domain.FileRepository) *DashboardHandler {
	return &DashboardHandler{fileRepo: fileRepo}
}

// Get renders the user dashboard page.
func (h *DashboardHandler) Get(c echo.Context) error {
	userVal := c.Get(middleware.UserContextKey)
	if userVal == nil {
		c.Logger().Warn("unauthenticated access attempt on protected dashboard")
		return c.Redirect(http.StatusFound, "/auth/login")
	}

	user, ok := userVal.(*domain.User)
	if !ok || user.ID == nil {
		c.Logger().Error("failed to assert context user to *domain.User")
		return c.String(http.StatusInternalServerError, "Authentication context error.")
	}

	// Find the most recent file uploaded by the user to use as a profile picture.
	var profilePicURL string
	if h.fileRepo != nil {
		latestFile, err := h.fileRepo.FindLatestByUser(c.Request().Context(), user.ID)
		if err != nil {
			// Log the error but don't block rendering the page.
			c.Logger().Warn("could not retrieve latest file for user", "user_id", user.ID.String(), "error", err)
		}
		if latestFile != nil && latestFile.ID != nil {
			profilePicURL = fmt.Sprintf("/app/files/%s/download", latestFile.ID.String())
		}
	}

	data := dashboard.Data{
		ID:                user.ID.String(),
		Email:             user.Email,
		ProfilePictureURL: profilePicURL,
	}

	pageContent := pages.Dashboard(data)
	flashData := view.GetFlashData(c) // Use view helper to get flash data

	finalComponent := layouts.Base("Dashboard", flashData.Messages, pageContent)
	return c.Render(http.StatusOK, "", finalComponent)
}
