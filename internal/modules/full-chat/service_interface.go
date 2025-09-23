package fullchat

import "context"

// Service defines the interface for the full-chat service.
type Service interface {
	// SendMessage sends a new chat message.
	SendMessage(ctx context.Context, text string) (*Message, error)
	
	// GetMessages retrieves a list of recent messages.
	// The limit parameter specifies the maximum number of messages to return.
	GetMessages(ctx context.Context, limit int) ([]*Message, error)
}
