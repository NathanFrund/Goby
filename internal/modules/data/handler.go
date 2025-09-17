package data

import (
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/middleware"
)

// Handler holds dependencies for the data module's WebSocket handlers.
type Handler struct {
	hub *hub.Hub
}

// NewHandler creates a new data handler with its dependencies.
func NewHandler(h *hub.Hub) *Handler {
	return &Handler{hub: h}
}

// ServeWS handles WebSocket connection requests for the data channel.
func (h *Handler) ServeWS(c echo.Context) error {
	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		InsecureSkipVerify: true, // In production, check origin.
	})
	if err != nil {
		slog.Error("Failed to upgrade data WebSocket", "error", err)
		return err
	}

	// Get the authenticated user from the context, with proper error handling.
	user, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok || user == nil {
		slog.Error("Could not get user from context for data WebSocket connection")
		return c.String(http.StatusUnauthorized, "User not authenticated")
	}
	sub := &hub.Subscriber{
		UserID: user.ID.String(),
		Send:   make(chan []byte, 256),
	}
	client := &Client{conn: conn, hub: h.hub, subscriber: sub}
	h.hub.Register <- client.subscriber

	go client.writePump()
	go client.readPump()

	return nil
}
