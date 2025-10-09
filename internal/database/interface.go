package database

import (
	"context"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// Client defines the interface for database operations.
type Client interface {
	// Executor provides the query execution interface
	Query(ctx context.Context, query string, params map[string]any, result interface{}) error
	QueryOne(ctx context.Context, query string, params map[string]any, result interface{}) error
	Execute(ctx context.Context, query string, params map[string]any) error

	// Create creates a new record in the specified table.
	Create(ctx context.Context, table string, data interface{}) (interface{}, error)

	// Select selects a record by ID from the specified table.
	Select(ctx context.Context, id string) (interface{}, error)

	// Update updates a record by ID in the specified table.
	Update(ctx context.Context, id string, data interface{}) (interface{}, error)

	// Delete deletes a record by ID from the specified table.
	Delete(ctx context.Context, id string) error

	// Close closes the database connection.
	Close() error
}

// NewClient creates a new database client that wraps the SurrealDB connection.
func NewClient(db *surrealdb.DB) Client {
	return &surrealClient{
		db:       db,
		executor: NewExecutor(db),
	}
}

type surrealClient struct {
	db       *surrealdb.DB
	executor Executor
}

// Executor implementation
func (c *surrealClient) Query(ctx context.Context, query string, params map[string]any, result interface{}) error {
	return c.executor.Query(ctx, query, params, result)
}

func (c *surrealClient) QueryOne(ctx context.Context, query string, params map[string]any, result interface{}) error {
	return c.executor.QueryOne(ctx, query, params, result)
}

func (c *surrealClient) Execute(ctx context.Context, query string, params map[string]any) error {
	return c.executor.Execute(ctx, query, params)
}

// CRUD operations
func (c *surrealClient) Create(ctx context.Context, table string, data interface{}) (interface{}, error) {
    query := fmt.Sprintf("CREATE %s CONTENT $data", table)
    var result interface{}
    err := c.executor.QueryOne(ctx, query, map[string]interface{}{"data": data}, &result)
    return result, err
}

func (c *surrealClient) Select(ctx context.Context, id string) (interface{}, error) {
    query := fmt.Sprintf("SELECT * FROM %s", id)
    var result interface{}
    err := c.executor.QueryOne(ctx, query, nil, &result)
    return result, err
}

func (c *surrealClient) Update(ctx context.Context, id string, data interface{}) (interface{}, error) {
    query := fmt.Sprintf("UPDATE %s MERGE $data", id)
    var result interface{}
    err := c.executor.QueryOne(ctx, query, map[string]interface{}{"data": data}, &result)
    return result, err
}

func (c *surrealClient) Delete(ctx context.Context, id string) error {
    query := fmt.Sprintf("DELETE %s", id)
    return c.executor.Execute(ctx, query, nil)
}

func (c *surrealClient) Close() error {
    // Pass context.Background() since we don't have a context here
    // The actual Close method might not use the context, but we need to satisfy the interface
    return c.db.Close(context.Background())
}
