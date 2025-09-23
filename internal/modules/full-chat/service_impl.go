package fullchat

import (
	"context"
	"fmt"
	"time"

	"github.com/nfrund/goby/internal/database"
	"github.com/surrealdb/surrealdb.go"
)

// service provides chat-related operations.
type service struct {
	store *Store
}

// NewService creates a new Service instance.
func NewService(db *surrealdb.DB, cfg *Config) Service {
	return &service{
		store: NewStore(db, cfg.SurrealNS, cfg.SurrealDB),
	}
}

// SendMessage sends a new chat message.
func (s *service) SendMessage(ctx context.Context, text string) (*Message, error) {
	msg, err := s.store.Create(ctx, text)
	if err != nil {
		return nil, err
	}
	
	// Set the creation time if not set
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}
	
	return msg, nil
}

// GetMessages retrieves a list of recent messages.
func (s *service) GetMessages(ctx context.Context, limit int) ([]*Message, error) {
	// Ensure we have a valid limit
	if limit <= 0 {
		limit = 50 // Default limit
	}

	// Use the database executor to query messages
	query := "SELECT * FROM message ORDER BY createdAt DESC LIMIT $limit"
	params := map[string]interface{}{
		"limit": limit,
	}

	// Execute the query
	result, err := database.Query[Message](ctx, s.store.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Convert to slice of pointers
	messages := make([]*Message, len(result))
	for i := range result {
		messages[i] = &result[i]
	}

	// Reverse the order to get oldest first
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
