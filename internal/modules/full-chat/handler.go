package fullchat

import "github.com/labstack/echo/v4"

// MessageHandler defines the interface for handling HTTP requests related to messages.
type MessageHandler interface {
	// ChatUI renders the chat interface
	ChatUI(c echo.Context) error
	
	// WebSocketHandler handles WebSocket connections for real-time chat
	WebSocketHandler(c echo.Context) error
	
	// CreateMessage handles the creation of a new message
	CreateMessage(c echo.Context) error
	
	// ListMessages retrieves a list of recent messages
	ListMessages(c echo.Context) error
}
