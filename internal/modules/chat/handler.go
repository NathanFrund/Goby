package chat

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/a-h/templ"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/middleware"
	"github.com/nfrund/goby/internal/modules/chat/templates/pages"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler manages the HTTP requests for the chat module.
type Handler struct {
	publisher pubsub.Publisher
}

// NewHandler creates a new chat handler.
func NewHandler(pub pubsub.Publisher) *Handler {
	return &Handler{
		publisher: pub,
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
	// For now, return a simple message - the real-time updates will come via WebSocket
	// In a production system, you might want to fetch current presence from the API
	return c.HTML(http.StatusOK, `<div id="presence-container">
		<div class="bg-gray-50 p-4 rounded-lg">
			<h3 class="text-sm font-semibold text-gray-700 mb-2">Online Users</h3>
			<div class="text-sm text-gray-500">Loading...</div>
		</div>
	</div>`)
}
