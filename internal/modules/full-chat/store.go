package fullchat

import (
	"context"
	"fmt"
	"time"

	"github.com/nfrund/goby/internal/database"
	"github.com/surrealdb/surrealdb.go"
)

// Message represents a chat message in the system.
type Message struct {
	ID        string    `json:"id,omitempty"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store handles database operations for the chat module.
type Store struct {
	db     *surrealdb.DB
	ns     string
	dbName string
}

// NewStore creates a new Store instance.
func NewStore(db *surrealdb.DB, ns, dbName string) *Store {
	return &Store{
		db:     db,
		ns:     ns,
		dbName: dbName,
	}
}

// Create saves a new message to the database.
func (s *Store) Create(ctx context.Context, text string) (*Message, error) {
	// Ensure we're using the correct namespace and database
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// Use the database executor to create the message
	query := "CREATE message SET text = $text, createdAt = time::now() RETURN AFTER"
	params := map[string]interface{}{
		"text": text,
	}

	// Execute the query and fetch the created message
	created, err := database.QueryOne[Message](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create and fetch message: %w", err)
	}
	if created == nil {
		return nil, fmt.Errorf("message was not created or could not be fetched")
	}

	return created, nil
}

// GetMessages retrieves a list of recent messages.
func (s *Store) GetMessages(ctx context.Context, limit int) ([]*Message, error) {
	// Ensure we're using the correct namespace and database
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// Use the database executor to query messages
	query := "SELECT * FROM message ORDER BY createdAt DESC LIMIT $limit"
	params := map[string]interface{}{
		"limit": limit,
	}

	// Execute the query
	result, err := database.Query[Message](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	// Convert to slice of pointers
	messages := make([]*Message, len(result))
	for i := range result {
		messages[i] = &result[i]
	}

	// Reverse the order to get oldest first for UI display
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}
