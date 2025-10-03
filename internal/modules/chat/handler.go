package chat

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/hub"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/modules/chat/templates/pages"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler holds dependencies for the chat module's HTTP handlers.
type Handler struct {
	hub      *hub.Hub
	renderer rendering.Renderer
}

// NewHandler creates a new chat handler with its dependencies.
func NewHandler(h *hub.Hub, r rendering.Renderer) *Handler {
	return &Handler{hub: h, renderer: r}
}

// ChatGet serves the main chat page.
func (h *Handler) ChatGet(c echo.Context) error {
	// The renderer is available on the handler struct.
	// We can use it to render the ChatPage component directly.
	if r, ok := h.renderer.(*rendering.UniversalRenderer); ok {
		// Follow the handler-level composition pattern used elsewhere.
		// 1. Get the flash data.
		flashes := view.GetFlashData(c)
		// 2. Create the inner page content.
		pageContent := pages.ChatPage()
		// 3. Wrap the content in the base layout and render.
		return r.RenderPage(c, http.StatusOK, layouts.Base("Goby Chat", flashes, pageContent))
	}
	// Fallback for safety, though in practice the type assertion should always succeed.
	return c.String(http.StatusInternalServerError, "Renderer not configured correctly")
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

	sub := &hub.Subscriber{
		UserID: user.ID.String(),
		Send:   make(chan []byte, 256),
	}
	client := &Client{conn: conn, hub: h.hub, subscriber: sub, User: user, renderer: h.renderer}
	h.hub.Register <- client.subscriber

	// --- Send a welcome message directly to the new user ---
	go func(renderer rendering.Renderer) {
		// Pass the renderer into the goroutine to ensure safe concurrent access.
		welcomeComponent := components.WelcomeMessage("Welcome to the chat, " + user.Email + "!")
		renderedHTML, err := renderer.RenderComponent(context.Background(), welcomeComponent)
		if err != nil {
			slog.Error("Failed to render welcome message", "error", err)
		} else {

			directMessage := &hub.DirectMessage{
				UserID:  user.ID.String(),
				Payload: renderedHTML,
			}
			h.hub.Direct <- directMessage
		}
	}(h.renderer)

	go client.writePump()
	go client.readPump()

	return nil
}
