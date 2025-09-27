package chat

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/middleware"
)

// Handler handles HTTP requests for the chat2 module.
type Handler struct {
	hub      *hub.Hub
	renderer echo.Renderer
}

// NewHandler creates a new chat2 handler.
func NewHandler(h *hub.Hub, r echo.Renderer) *Handler {
	return &Handler{
		hub:      h,
		renderer: r,
	}
}

// ChatGet handles GET /chat2 requests.
func (h *Handler) ChatGet(c echo.Context) error {
	return c.Render(http.StatusOK, "chat2/pages/chat.html", nil)
}

// ServeWS handles WebSocket connections for chat.
func (h *Handler) ServeWS(c echo.Context) error {
	// Get the authenticated user from context
	user, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok {
		slog.Error("Could not get user from context for WebSocket connection")
		return c.String(http.StatusUnauthorized, "User not authenticated")
	}

	// Upgrade to WebSocket
	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		InsecureSkipVerify: true, // In production, verify origin
	})
	if err != nil {
		slog.Error("Failed to upgrade WebSocket connection", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to upgrade to WebSocket")
	}

	// Create and register client
	sub := &hub.Subscriber{
		UserID: user.ID.String(),
		Send:   make(chan []byte, 256),
	}

	client := &Client{
		conn:       conn,
		hub:        h.hub,
		subscriber: sub,
		User:       user,
		renderer:   h.renderer,
	}

	h.hub.Register <- client.subscriber

	// Start client goroutines
	go client.writePump()
	go client.readPump()

	return nil
}
