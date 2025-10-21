package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/presence"
)

// PresenceHandler handles presence-related HTTP requests
type PresenceHandler struct {
	presenceService *presence.Service
}

// NewPresenceHandler creates a new presence handler
func NewPresenceHandler(presenceService *presence.Service) *PresenceHandler {
	return &PresenceHandler{
		presenceService: presenceService,
	}
}

// GetPresence returns the current online users as JSON
func (h *PresenceHandler) GetPresence(c echo.Context) error {
	c.Logger().Info("Presence JSON endpoint called")
	
	if h.presenceService == nil {
		c.Logger().Error("Presence service is nil")
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "presence service not available",
		})
	}

	c.Logger().Info("About to call GetOnlineUsers")
	onlineUsers := h.presenceService.GetOnlineUsers()
	c.Logger().Info("GetOnlineUsers returned", "count", len(onlineUsers))
	
	response := map[string]interface{}{
		"online_users": onlineUsers,
		"count":        len(onlineUsers),
	}
	
	c.Logger().Info("About to return JSON response")
	return c.JSON(http.StatusOK, response)
}

// GetPresenceHTML returns the presence list as HTML fragment for HTMX
func (h *PresenceHandler) GetPresenceHTML(c echo.Context) error {
	c.Logger().Info("=== PRESENCE HTML ENDPOINT CALLED ===")
	c.Logger().Info("Request path", "path", c.Request().URL.Path)
	c.Logger().Info("Request method", "method", c.Request().Method)
	
	if h.presenceService == nil {
		c.Logger().Error("Presence service is nil")
		return c.HTML(http.StatusServiceUnavailable, `<div class="text-red-500">Presence service unavailable</div>`)
	}

	c.Logger().Info("About to get online users")
	onlineUsers := h.presenceService.GetOnlineUsers()
	c.Logger().Info("Retrieved online users", "count", len(onlineUsers), "users", onlineUsers)
	
	// Render the presence component
	c.Logger().Info("About to render component")
	component := components.OnlineUsers(onlineUsers)
	
	c.Logger().Info("About to return rendered component")
	return c.Render(http.StatusOK, "", component)
}

// GetUserPresence returns the presence status for a specific user
func (h *PresenceHandler) GetUserPresence(c echo.Context) error {
	if h.presenceService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "presence service not available",
		})
	}

	userID := c.Param("userID")
	if userID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "userID parameter required",
		})
	}

	presence, exists := h.presenceService.GetPresence(userID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "user not found or offline",
		})
	}

	return c.JSON(http.StatusOK, presence)
}

// DebugAddUser manually adds a user for testing (remove in production)
func (h *PresenceHandler) DebugAddUser(c echo.Context) error {
	c.Logger().Info("Debug endpoint called")
	
	if h.presenceService == nil {
		c.Logger().Error("Presence service is nil in debug endpoint")
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"error": "presence service not available",
		})
	}

	c.Logger().Info("Presence service is available in debug endpoint")
	
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Presence service is available",
		"status":  "ok",
	})
}

// HealthCheck returns the health status of the presence service
func (h *PresenceHandler) HealthCheck(c echo.Context) error {
	c.Logger().Info("Health check endpoint called")
	
	if h.presenceService == nil {
		return c.JSON(http.StatusServiceUnavailable, map[string]string{
			"status": "error",
			"error":  "presence service not available",
		})
	}
	
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"message": "presence service is available",
	})
}

// TestHTML is a simple test endpoint to check if HTML rendering works
func (h *PresenceHandler) TestHTML(c echo.Context) error {
	c.Logger().Info("=== TEST HTML ENDPOINT CALLED ===")
	return c.HTML(http.StatusOK, `<div>Test HTML Response</div>`)
}