package chat

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/chat/templates/components"
	"github.com/nfrund/goby/internal/modules/chat/templates/pages"
	"github.com/nfrund/goby/internal/presence"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/rendering"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler manages the HTTP requests for the chat module.
type Handler struct {
	publisher       pubsub.Publisher
	presenceService *presence.Service
	renderer        rendering.Renderer
}

// NewHandler creates a new chat handler.
func NewHandler(pub pubsub.Publisher, presenceService *presence.Service, renderer rendering.Renderer) *Handler {
	return &Handler{
		publisher:       pub,
		presenceService: presenceService,
		renderer:        renderer,
	}
}

// ChatGet renders the main chat page using the application's rendering pipeline.
func (h *Handler) ChatGet(c echo.Context) error {
	pageContent := pages.ChatPage()
	// The renderer expects a templ.Component, so we wrap it. We pass .FlashData to get the embedded struct.
	finalComponent := templ.Component(layouts.Base("Chat", view.GetFlashData(c).Messages, pageContent))
	return c.Render(http.StatusOK, "", finalComponent)
}

// MessagePost handles the form submission for a new chat message.
func (h *Handler) MessagePost(c echo.Context) error {
	user := c.Get(middleware.UserContextKey).(*domain.User)
	content := c.FormValue("content")

	if content == "" {
		return c.NoContent(http.StatusBadRequest)
	}

	// Create a structured message with the new topic format
	msg := struct {
		Content string `json:"content"`
		User    string `json:"user"`
	}{
		Content: content,
		User:    user.Email,
	}

	payload, err := json.Marshal(msg)
	if err != nil {
		slog.Error("Failed to marshal chat message", "error", err)
		return c.NoContent(http.StatusInternalServerError)
	}

	// Publish to the chat.messages topic using the typed topic
	h.publisher.Publish(c.Request().Context(), pubsub.Message{
		Topic:   Messages.Name(),
		UserID:  user.Email,
		Payload: payload,
	})

	return c.NoContent(http.StatusOK)
}

// PresenceGet renders the presence component as HTML fragment for HTMX
func (h *Handler) PresenceGet(c echo.Context) error {
	// Fetch current online users from presence service
	onlineUsers := h.presenceService.GetOnlineUsers()
	
	slog.Info("Rendering presence for HTMX request", 
		"user_count", len(onlineUsers),
		"users", onlineUsers)
	
	// Render the presence component
	component := components.OnlineUsers(onlineUsers)
	renderedHTML, err := h.renderer.RenderComponent(c.Request().Context(), component)
	if err != nil {
		slog.Error("Failed to render presence component", "error", err)
		return c.HTML(http.StatusInternalServerError, `<div class="text-red-500">Error loading presence</div>`)
	}
	
	return c.HTMLBlob(http.StatusOK, renderedHTML)
}
