package chat

import (
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

	// The payload is the raw form data, which the subscriber expects.
	payload := []byte(`{"content":"` + content + `"}`)
	msg := pubsub.Message{Topic: "chat.messages.new", UserID: user.Email, Payload: payload}
	h.publisher.Publish(c.Request().Context(), msg)

	return c.NoContent(http.StatusOK)
}
