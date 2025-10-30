package database

import (
	"context"
	"log/slog"
	"strings"

	"github.com/surrealdb/surrealdb.go"
)

// surrealExecutor implements the QueryExecutor interface for SurrealDB
type surrealExecutor[T any] struct {
	conn DBConnection
}

// NewSurrealExecutor creates a new SurrealDB query executor
func NewSurrealExecutor[T any](conn DBConnection) QueryExecutor[T] {
	return &surrealExecutor[T]{conn: conn}
}

// Query executes a query and returns multiple results
func (e *surrealExecutor[T]) Query(ctx context.Context, query string, params map[string]any) ([]T, error) {
	if query == "" {
		return nil, NewDBError(ErrInvalidInput, "query cannot be empty")
	}

	// Start logging with context. This assumes a logger is available in the context.
	// For now, we'll use the global slog logger for demonstration.
	slog.DebugContext(ctx, "Executing database query",
		"query", query,
		"params", params,
	)

	var finalResults []T
	err := e.conn.WithConnection(ctx, func(db *surrealdb.DB) error {
		// Execute the query with context using the package-level Query function
		results, err := surrealdb.Query[[]T](ctx, db, query, params)
		if err != nil {
			return NewDBError(err, "query execution failed")
		}

		// Check if we got any results
		if results == nil || len(*results) == 0 {
			finalResults = nil // Ensure nil slice if no results
			return nil
		}

		finalResults = (*results)[0].Result
		return nil
	})
	return finalResults, err
}

// QueryOne executes a query and returns a single result
func (e *surrealExecutor[T]) QueryOne(ctx context.Context, query string, params map[string]any) (*T, error) {
	// Ensure we're only getting one result for SELECT queries.
	// CREATE/UPDATE/DELETE statements don't support LIMIT.
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") && !hasLimitClause(query) {
		query += " LIMIT 1"
	}

	// Reuse the Query helper to avoid duplicating logic for handling results.
	results, err := e.Query(ctx, query, params)
	if err != nil {
		return nil, err // Error is already wrapped by the Query function
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

// Execute runs a query that doesn't return any rows
func (e *surrealExecutor[T]) Execute(ctx context.Context, query string, params map[string]any) error {
	if query == "" {
		return NewDBError(ErrInvalidInput, "query cannot be empty")
	}

	// Log the execution command.
	slog.DebugContext(ctx, "Executing database command",
		"query", query,
		"params", params,
	)

	return e.conn.WithConnection(ctx, func(db *surrealdb.DB) error {
		// Execute the query using the package-level Query function
		_, err := surrealdb.Query[any](ctx, db, query, params)
		if err != nil {
			return NewDBError(err, "query execution failed")
		}
		return nil
	})
}

// hasLimitClause checks if the query already has a LIMIT clause
func hasLimitClause(query string) bool {
	// Simple check for LIMIT keyword (case insensitive)
	return strings.Contains(strings.ToUpper(" "+query+" "), " LIMIT ")
}
