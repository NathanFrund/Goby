package messenger

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/registry"
	"github.com/surrealdb/surrealdb.go"
)

// Message represents a chat message
type Message struct {
	ID        string    `json:"id,omitempty"`
	Content   string    `json:"content"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
}

// Store handles database operations for the messenger module
type Store struct {
	db *surrealdb.DB
}

// This assignment makes the compiler think getMessageByID is being used
var _ = (*Store).getMessageByID

// NewStore creates a new store instance with a service locator
func NewStore(sl registry.ServiceLocator) *Store {
	dbIface, ok := sl.Get(string(registry.DBConnectionKey))
	if !ok || dbIface == nil {
		slog.Error("Failed to get database connection from service locator")
		return &Store{db: nil}
	}

	db, ok := dbIface.(*surrealdb.DB)
	if !ok {
		slog.Error("Database connection is not a SurrealDB client", "type", dbIface)
		return &Store{db: nil}
	}

	return &Store{
		db: db,
	}
}

// CreateMessage creates a new message in the database
func (s *Store) CreateMessage(msg *Message) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Set the creation time if not already set
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now()
	}

	// Create the message in the database using raw query
	query := `
		CREATE messages CONTENT {
			content: $content,
			user_id: $user_id,
			username: $username,
			createdAt: $created_at
		}
	`

	params := map[string]interface{}{
		"content":    msg.Content,
		"user_id":    msg.UserID,
		"username":   msg.Username,
		"created_at": msg.CreatedAt.Format(time.RFC3339),
	}

	// Execute the query
	return database.Execute(context.Background(), s.db, query, params)
}

// ListMessages retrieves messages from the database
func (s *Store) ListMessages(limit int) ([]*Message, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Query messages from the database
	query := `SELECT * FROM messages ORDER BY createdAt DESC LIMIT $limit`
	params := map[string]interface{}{
		"limit": limit,
	}

	// Execute the query
	return database.Query[*Message](context.Background(), s.db, query, params)
}

// DeleteMessage deletes a message by its ID
func (s *Store) DeleteMessage(id string) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Make sure the ID is in the correct format
	if !strings.HasPrefix(id, "messages:") {
		id = "messages:" + id
	}

	// Delete the message using a raw query
	query := `DELETE $id`
	params := map[string]interface{}{
		"id": id,
	}

	return database.Execute(context.Background(), s.db, query, params)
}

// getMessageByID retrieves a message by its ID.
// TODO: This will be used for fetching individual messages when users click on message links.
// The current implementation is kept for future use when implementing message permalinks.
func (s *Store) getMessageByID(id string) (*Message, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	// Make sure the ID is in the correct format
	if !strings.HasPrefix(id, "messages:") {
		id = "messages:" + id
	}

	// Query the message by ID
	query := `SELECT * FROM $id`
	params := map[string]interface{}{
		"id": id,
	}

	// Execute the query
	messages, err := database.Query[*Message](context.Background(), s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to query message: %w", err)
	}
	if len(messages) == 0 {
		return nil, fmt.Errorf("message not found")
	}
	return messages[0], nil
}

// WatchMessages starts watching for new messages using SurrealDB live queries
func (s *Store) WatchMessages(callback func(*Message)) error {
	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	// Create a live query for new messages
	query := `LIVE SELECT * FROM messages`

	// Execute the live query using the raw surrealdb package since it's a special case
	results, err := surrealdb.Query[[]map[string]interface{}](context.Background(), s.db, query, map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to create live query: %w", err)
	}

	// The first result contains the live query ID
	if len(*results) == 0 || len((*results)[0].Result) == 0 {
		return fmt.Errorf("failed to get live query ID")
	}

	liveID, ok := (*results)[0].Result[0]["id"].(string)
	if !ok {
		return fmt.Errorf("invalid live query ID format")
	}

	// Listen for live query events in a goroutine
	go func() {
		for {
			// Poll for new events (this is a simplified approach)
			// In a real application, you might want to use a more efficient method
			// like WebSockets or server-sent events
			time.Sleep(1 * time.Second)

			// Check for new events
			events, err := surrealdb.Query[[]map[string]interface{}](context.Background(), s.db, "SELECT * FROM $live_id", map[string]interface{}{
				"live_id": liveID,
			})

			if err != nil {
				slog.Error("Error polling live query", "error", err)
				continue
			}

			// Process events
			for _, result := range *events {
				for _, event := range result.Result {
					// Check if this is a create/update event
					if action, ok := event["action"].(string); ok && (action == "CREATE" || action == "UPDATE") {
						// Extract the message data
						if data, ok := event["result"].(map[string]interface{}); ok {
							// Parse the created time from the database
							var createdAt time.Time
							if createdAtStr, ok := data["createdAt"].(string); ok {
								if t, err := time.Parse(time.RFC3339, createdAtStr); err == nil {
									createdAt = t
								}
							}

							// Create the message object
							msg := &Message{
								ID:        getString(data, "id"),
								Content:   getString(data, "content"),
								UserID:    getString(data, "user_id"),
								Username:  getString(data, "username"),
								CreatedAt: createdAt,
							}

							// Call the callback with the new/updated message
							callback(msg)
						}
					}
				}
			}
		}
	}()

	return nil
}

// Helper function to safely get string values from a map
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}
