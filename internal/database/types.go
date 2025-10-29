package database

import (
	"context"
	"time"

	"github.com/surrealdb/surrealdb.go"
)

// DBConnection defines the interface for a managed database connection.
// It abstracts the underlying database driver and handles connection logic,
// allowing repositories to perform driver-specific operations without being
// tied to a concrete implementation.
type DBConnection interface {
	DB() (*surrealdb.DB, error)
	WithConnection(ctx context.Context, fn func(*surrealdb.DB) error) error
	Close(ctx context.Context) error
	IsHealthy() bool
	StartMonitoring()
	Connect(ctx context.Context) error
	GetDBNs() string
	GetDBDb() string
	GetDBQueryTimeout() time.Duration
	GetDBExecuteTimeout() time.Duration
}

// Client defines the main database client interface with type-safe methods.
// It provides a generic interface for database operations on a specific type T.
type Client[T any] interface {
	// Create inserts a new record into the specified table with the given data.
	// The data can be either a struct or a map of fields to values.
	// Returns the created record with all fields populated, including any server-generated fields.
	Create(ctx context.Context, table string, data any) (*T, error)

	// Select retrieves a record by its full ID (e.g., "user:123").
	// Returns the record with all fields populated.
	// Returns ErrNotFound if no record exists with the given ID.
	Select(ctx context.Context, id string) (*T, error)

	// Update modifies an existing record with the given ID using the provided data.
	// The data can be either a struct (which will be merged with the existing record)
	// or a map of fields to update.
	// Returns the updated record with all fields populated.
	Update(ctx context.Context, id string, data any) (*T, error)

	// Delete removes a record with the given ID.
	// Returns ErrNotFound if no record exists with the given ID.
	Delete(ctx context.Context, id string) error

	// Query executes a raw query and returns multiple results.
	// The query can include parameters using the $param syntax.
	// Returns a slice of type T containing the query results.
	Query(ctx context.Context, query string, params map[string]any) ([]T, error)

	// QueryOne executes a raw query and returns a single result.
	// Returns (nil, nil) if no results are found.
	// Returns an error if the query returns more than one result.
	QueryOne(ctx context.Context, query string, params map[string]any) (*T, error)

	// Execute runs a query that doesn't return any rows (e.g., INSERT, UPDATE, DELETE).
	// Use this for operations where you don't need to process the returned data.
	Execute(ctx context.Context, query string, params map[string]any) error

	// Close releases any resources associated with the client.
	// Always call this when the client is no longer needed.
	Close() error
}

// QueryExecutor handles the execution of database queries.
// This interface is used internally by the Client implementation.
type QueryExecutor[T any] interface {
	// Query executes a query and returns multiple results.
	// The query can include parameters using the $param syntax.
	// Returns a slice of type T containing the query results.
	Query(ctx context.Context, query string, params map[string]any) ([]T, error)

	// QueryOne executes a query and returns a single result.
	// Returns (nil, nil) if no results are found.
	// Returns an error if the query returns more than one result.
	QueryOne(ctx context.Context, query string, params map[string]any) (*T, error)

	// Execute runs a query that doesn't return any rows.
	// Use this for operations like INSERT, UPDATE, or DELETE.
	Execute(ctx context.Context, query string, params map[string]any) error
}

// ClientOption defines a function that configures a Client.
// This allows for flexible client configuration using functional options.
type ClientOption[T any] func(*client[T])

// WithExecutor configures the client to use a custom QueryExecutor.
// This is useful for testing or for adding middleware to the executor.
func WithExecutor[T any](executor QueryExecutor[T]) ClientOption[T] {
	return func(c *client[T]) {
		c.executor = executor
	}
}
