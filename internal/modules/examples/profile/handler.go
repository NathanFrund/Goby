package profile

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/examples/profile/view"
	gview "github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler handles requests for the user dashboard.
type Handler struct {
	fileRepo domain.FileRepository
}

// NewHandler creates a new DashboardHandler.
func NewHandler(fileRepo domain.FileRepository) *Handler {
	return &Handler{
		fileRepo: fileRepo,
	}
}

// Get renders the user dashboard page.
func (h *Handler) Get(c echo.Context) error {
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

	data := view.Data{
		ID:                user.ID.String(),
		Email:             user.Email,
		ProfilePictureURL: profilePicURL,
	}

	pageContent := view.Profile(data)
	flashData := gview.GetFlashData(c) // Use view helper to get flash data

	finalComponent := layouts.Base("Profile", flashData.Messages, pageContent)
	return c.Render(http.StatusOK, "", finalComponent)
}
