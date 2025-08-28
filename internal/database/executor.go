package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/surrealdb/surrealdb.go"
)

// Query executes a raw SurrealQL query with parameters and returns multiple results.
// It's a generic function that can unmarshal results into any type T.
//
// Example:
//
//	query := "SELECT * FROM user WHERE active = $active"
//	users, err := Query[User](ctx, db, query, map[string]interface{}{"active": true})
func Query[T any](ctx context.Context, db *surrealdb.DB, query string, params map[string]any) ([]T, error) {
	queryResults, err := surrealdb.Query[[]T](ctx, db, query, params)
	if err != nil {
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	if len(*queryResults) == 0 {
		return nil, nil
	}
	return (*queryResults)[0].Result, nil
}

// QueryOne executes a query and returns a single result.
// If no results are found, it returns nil, nil.
//
// Example:
//
//	query := "SELECT * FROM user WHERE email = $email"
//	user, err := QueryOne[User](ctx, db, query, map[string]interface{}{"email": "test@example.com"})
func QueryOne[T any](ctx context.Context, db *surrealdb.DB, query string, params map[string]any) (*T, error) {
	// Ensure we're only getting one result for SELECT queries.
	// CREATE/UPDATE/DELETE statements don't support LIMIT.
	if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(query)), "SELECT") && !hasLimitClause(query) {
		query += " LIMIT 1"
	}

	// Reuse the Query helper to avoid duplicating logic for handling results.
	results, err := Query[T](ctx, db, query, params)
	if err != nil {
		return nil, err // Error is already wrapped by the Query function
	}
	if len(results) == 0 {
		return nil, nil
	}
	return &results[0], nil
}

// Execute runs a query that doesn't return rows (INSERT, UPDATE, DELETE, etc.)
// and returns the raw SurrealDB response.
//
// Example:
//
//	query := "UPDATE user SET name = $name WHERE id = $id"
//	_, err := Execute(ctx, db, query, map[string]interface{}{
//	    "id": "user:123",
//	    "name": "New Name",
//	})
func Execute(ctx context.Context, db *surrealdb.DB, query string, params map[string]any) error {
	// Use the underlying Query method but discard the results. We only care about the error.
	if _, err := surrealdb.Query[any](ctx, db, query, params); err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	return nil
}

// hasLimitClause checks if the query already has a LIMIT clause
func hasLimitClause(query string) bool {
	// Simple check for LIMIT keyword (case insensitive)
	query = " " + strings.ToUpper(query) + " "
	return strings.Contains(query, " LIMIT ")
}
