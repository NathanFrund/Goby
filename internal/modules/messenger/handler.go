package messenger

import (
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/registry"
)

// Handler defines the interface for handling HTTP requests for the messenger.
type Handler interface {
	ChatUI(echo.Context) error
	CreateMessage(echo.Context) error
	ListMessages(echo.Context) error
	WebSocket(echo.Context) error
}

// handler implements the Handler interface
type handler struct {
	store  *Store
	sl     registry.ServiceLocator
}

// NewHandler creates a new handler instance
func NewHandler(store *Store, sl registry.ServiceLocator) Handler {
	return &handler{
		store: store,
		sl:    sl,
	}
}
