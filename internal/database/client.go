package database

import (
	"context"

	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// Client defines the interface for database operations. It abstracts the underlying
// database implementation, allowing for easier testing and potential future database migrations.
type Client interface {
	// Query executes a raw query and unmarshals multiple results into a slice of type T.
	Query(ctx context.Context, query string, params map[string]any) ([]any, error)
	// QueryOne executes a raw query and unmarshals a single result into a pointer of type T.
	QueryOne(ctx context.Context, query string, params map[string]any) (any, error)
	// Execute runs a query that doesn't return rows (e.g., INSERT, UPDATE, DELETE).
	Execute(ctx context.Context, query string, params map[string]any) error

	// Create creates a new record in the specified table.
	Create(ctx context.Context, table string, data any) (any, error)
	// Select retrieves a record by its full ID (e.g., "user:123").
	Select(ctx context.Context, id string) (any, error)
	// Update merges data into an existing record by its full ID.
	Update(ctx context.Context, id string, data any) (any, error)
	// Delete removes a record by its full ID.
	Delete(ctx context.Context, id string) error

	// DB returns the raw underlying database connection, useful for specific driver features.
	DB() *surrealdb.DB
	// Close closes the database connection.
	Close() error
}

// NewClient creates a new database client that wraps the SurrealDB connection.
func NewClient(db *surrealdb.DB) Client {
	return &surrealClient{db: db}
}

type surrealClient struct {
	db *surrealdb.DB
}

func (c *surrealClient) Query(ctx context.Context, query string, params map[string]any) ([]any, error) {
	return Query[any](ctx, c.db, query, params)
}

func (c *surrealClient) QueryOne(ctx context.Context, query string, params map[string]any) (any, error) {
	return QueryOne[any](ctx, c.db, query, params)
}

func (c *surrealClient) Execute(ctx context.Context, query string, params map[string]any) error {
	return Execute(ctx, c.db, query, params)
}

func (c *surrealClient) Create(ctx context.Context, table string, data any) (any, error) {
	query := fmt.Sprintf("CREATE %s CONTENT $data", table)
	return QueryOne[any](ctx, c.db, query, map[string]any{"data": data})
}

func (c *surrealClient) Select(ctx context.Context, id string) (any, error) {
	query := fmt.Sprintf("SELECT * FROM %s", id)
	return QueryOne[any](ctx, c.db, query, nil)
}

func (c *surrealClient) Update(ctx context.Context, id string, data any) (any, error) {
	query := fmt.Sprintf("UPDATE %s MERGE $data", id)
	return QueryOne[any](ctx, c.db, query, map[string]any{"data": data})
}

func (c *surrealClient) Delete(ctx context.Context, id string) error {
	query := fmt.Sprintf("DELETE %s", id)
	return Execute(ctx, c.db, query, nil)
}

func (c *surrealClient) DB() *surrealdb.DB {
	return c.db
}

func (c *surrealClient) Close() error {
	return c.db.Close(context.Background())
}
