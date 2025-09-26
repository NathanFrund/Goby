package messenger

import (
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
)

// ChatUI renders the chat interface
func (h *handler) ChatUI(c echo.Context) error {
	user, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	messages, err := h.store.ListMessages(50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load messages")
	}

	// Reverse the messages to show newest at the bottom
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return c.Render(http.StatusOK, "pages/messenger/messenger.html", map[string]interface{}{
		"Messages": messages,
		"User":     user,
	})
}

// CreateMessage handles the creation of a new message
func (h *handler) CreateMessage(c echo.Context) error {
	user, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request")
	}

	// Use email as username since that's what we have in the User struct
	username := strings.Split(user.Email, "@")[0] // Use the part before @ as username

	msg := &Message{
		Content:   req.Content,
		UserID:    user.ID.String(), // Convert RecordID to string
		Username:  username,
		CreatedAt: time.Now(),
	}

	if err := h.store.CreateMessage(msg); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to save message")
	}

	// The message will be broadcast via the live query in the WebSocket handler
	return c.NoContent(http.StatusCreated)
}

// ListMessages returns a list of recent messages
func (h *handler) ListMessages(c echo.Context) error {
	messages, err := h.store.ListMessages(50)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to load messages")
	}
	return c.JSON(http.StatusOK, messages)
}

// WebSocket handles WebSocket connections for real-time updates
func (h *handler) WebSocket(c echo.Context) error {
	_, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok {
		return echo.NewHTTPError(http.StatusUnauthorized, "User not authenticated")
	}

	// For now, we'll just return a not implemented error
	// as the WebSocket implementation needs to be updated
	// to work with the hub package's channel-based API
	return echo.NewHTTPError(http.StatusNotImplemented, "WebSocket support is not yet implemented")
}
