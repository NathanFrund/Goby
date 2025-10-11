package v2

import (
	"context"
	"fmt"
	"time"

	"github.com/nfrund/goby/internal/config"
	"github.com/surrealdb/surrealdb.go"
)

type client[T any] struct {
	db             *surrealdb.DB
	executor       QueryExecutor[T]
	queryTimeout   time.Duration
	executeTimeout time.Duration
}

// NewClient creates a new type-safe database client
func NewClient[T any](db *surrealdb.DB, cfg config.Provider, opts ...ClientOption[T]) (Client[T], error) {
	if db == nil {
		return nil, NewDBError(ErrInvalidInput, "db cannot be nil")
	}
	if cfg == nil {
		return nil, NewDBError(ErrInvalidInput, "config provider cannot be nil")
	}

	// Validate configuration values
	queryTimeout := cfg.GetDBQueryTimeout()
	if queryTimeout <= 0 {
		return nil, NewDBError(ErrInvalidInput, "DB_QUERY_TIMEOUT must be a positive duration")
	}
	executeTimeout := cfg.GetDBExecuteTimeout()
	if executeTimeout <= 0 {
		return nil, NewDBError(ErrInvalidInput, "DB_EXECUTE_TIMEOUT must be a positive duration")
	}

	c := &client[T]{
		db:             db,
		executor:       NewSurrealExecutor[T](db),
		queryTimeout:   cfg.GetDBQueryTimeout(),
		executeTimeout: cfg.GetDBExecuteTimeout(),
	}

	// Apply options
	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Query implements the Client interface
func (c *client[T]) Query(ctx context.Context, query string, params map[string]any) ([]T, error) {
	ctx, cancel := getTimeoutFromContext(ctx, c.queryTimeout, ContextKeyQueryTimeout)
	defer cancel()
	return c.executor.Query(ctx, query, params)
}

// QueryOne implements the Client interface
func (c *client[T]) QueryOne(ctx context.Context, query string, params map[string]any) (*T, error) {
	ctx, cancel := getTimeoutFromContext(ctx, c.queryTimeout, ContextKeyQueryTimeout)
	defer cancel()
	return c.executor.QueryOne(ctx, query, params)
}

// Execute implements the Client interface
func (c *client[T]) Execute(ctx context.Context, query string, params map[string]any) error {
	ctx, cancel := getTimeoutFromContext(ctx, c.executeTimeout, ContextKeyExecuteTimeout)
	defer cancel()
	return c.executor.Execute(ctx, query, params)
}

// DB implements the Client interface
func (c *client[T]) DB() *surrealdb.DB {
	return c.db
}

// Create implements the Client interface
func (c *client[T]) Create(ctx context.Context, table string, data any) (*T, error) {
	if table == "" {
		return nil, NewDBError(ErrInvalidInput, "table cannot be empty")
	}
	if data == nil {
		return nil, NewDBError(ErrInvalidInput, "data cannot be nil")
	}

	ctx, cancel := getTimeoutFromContext(ctx, c.executeTimeout, ContextKeyExecuteTimeout)
	defer cancel()

	// Use a raw query for create
	query := "CREATE type::table($table) CONTENT $data"
	result, err := c.QueryOne(ctx, query, map[string]any{"table": table, "data": data})
	if err != nil {
		return nil, NewDBError(err, "create operation failed")
	}
	return result, nil
}

// Select implements the Client interface
func (c *client[T]) Select(ctx context.Context, id string) (*T, error) {
	if id == "" {
		return nil, NewDBError(ErrInvalidInput, "id cannot be empty")
	}

	ctx, cancel := getTimeoutFromContext(ctx, c.queryTimeout, ContextKeyQueryTimeout)
	defer cancel()

	// Use a raw query for select
	query := fmt.Sprintf("SELECT * FROM %s", id)
	result, err := c.QueryOne(ctx, query, nil)
	if err != nil {
		return nil, NewDBError(err, "select operation failed")
	}
	if result == nil {
		return nil, NewDBError(ErrNotFound, "record not found")
	}
	return result, nil
}

// Update implements the Client interface
func (c *client[T]) Update(ctx context.Context, id string, data any) (*T, error) {
	if id == "" {
		return nil, NewDBError(ErrInvalidInput, "id cannot be empty")
	}
	if data == nil {
		return nil, NewDBError(ErrInvalidInput, "data cannot be nil")
	}

	ctx, cancel := getTimeoutFromContext(ctx, c.executeTimeout, ContextKeyExecuteTimeout)
	defer cancel()

	// Use a raw query for update
	query := "UPDATE type::thing($id) MERGE $data"
	result, err := c.QueryOne(ctx, query, map[string]any{"id": id, "data": data})
	if err != nil {
		return nil, NewDBError(err, "update operation failed")
	}
	if result == nil {
		return nil, NewDBError(ErrNotFound, "record not found")
	}
	return result, nil
}

// Delete implements the Client interface
func (c *client[T]) Delete(ctx context.Context, id string) error {
	if id == "" {
		return NewDBError(ErrInvalidInput, "id cannot be empty")
	}

	ctx, cancel := getTimeoutFromContext(ctx, c.executeTimeout, ContextKeyExecuteTimeout)
	defer cancel()

	// Use a raw query for delete
	query := "DELETE type::thing($id)"
	return c.Execute(ctx, query, map[string]any{"id": id})
}

// Close implements the Client interface
func (c *client[T]) Close() error {
	return c.db.Close(context.Background())
}
