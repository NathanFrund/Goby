package chat

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/modules/chat/templates/pages"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/web/src/templates/layouts"
)

// Handler manages the HTTP requests for the chat module.
type Handler struct{}

// NewHandler creates a new chat handler.
func NewHandler() *Handler {
	return &Handler{}
}

// ChatGet renders the main chat page using the application's rendering pipeline.
func (h *Handler) ChatGet(c echo.Context) error {
	pageContent := pages.ChatPage()
	finalComponent := layouts.Base("Chat", view.GetFlashData(c), pageContent)
	return c.Render(http.StatusOK, "", finalComponent)
}
