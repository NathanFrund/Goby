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

// Handler holds dependencies for the chat module's HTTP handlers.
type Handler struct {
	hub      *hub.Hub
	renderer echo.Renderer
}

// NewHandler creates a new chat handler with its dependencies.
func NewHandler(h *hub.Hub, r echo.Renderer) *Handler {
	return &Handler{hub: h, renderer: r}
}

// ChatGet serves the main chat page.
func (h *Handler) ChatGet(c echo.Context) error {
	return c.Render(http.StatusOK, "chat.html", nil)
}

// ServeWS handles WebSocket connection requests for the chat.
func (h *Handler) ServeWS(c echo.Context) error {
	slog.Info("ServeWS: Received request to upgrade to WebSocket")
	conn, err := websocket.Accept(c.Response(), c.Request(), &websocket.AcceptOptions{
		// In a production environment, you should check the origin to prevent CSRF.
		// For this template, we'll allow any origin.
		InsecureSkipVerify: true,
	})
	if err != nil {
		slog.Error("Failed to upgrade WebSocket connection", "error", err)
		return c.String(http.StatusInternalServerError, "Failed to upgrade to WebSocket")
	}

	slog.Info("ServeWS: WebSocket connection upgraded successfully. Creating client.")

	// Get the authenticated user from the context.
	user, ok := c.Get(middleware.UserContextKey).(*domain.User)
	if !ok {
		slog.Error("Could not get user from context for WebSocket connection")
		return c.String(http.StatusUnauthorized, "User not authenticated")
	}

	sub := &hub.Subscriber{Send: make(chan any, 256)}
	client := &Client{conn: conn, hub: h.hub, subscriber: sub, User: user, renderer: h.renderer}
	h.hub.Register <- client.subscriber

	go client.writePump()
	go client.readPump()

	return nil
}
