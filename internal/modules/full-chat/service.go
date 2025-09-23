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
	query := "CREATE message SET text = $text"
	params := map[string]interface{}{
		"text": text,
	}

	// Execute the query
	if err := database.Execute(ctx, s.db, query, params); err != nil {
		return nil, fmt.Errorf("failed to create message: %w", err)
	}

	// Now fetch the created message to get the ID
	query = "SELECT * FROM message WHERE text = $text ORDER BY id DESC LIMIT 1"
	createdMsg, err := database.QueryOne[Message](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created message: %w", err)
	}
	if createdMsg == nil {
		return nil, fmt.Errorf("message was not created")
	}

	return createdMsg, nil
}
