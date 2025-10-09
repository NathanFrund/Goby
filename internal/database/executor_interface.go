package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/surrealdb/surrealdb.go"
)

// Executor defines the interface for executing database queries
type Executor interface {
	// Query executes a query and returns multiple results
	Query(ctx context.Context, query string, params map[string]any, result interface{}) error
	
	// QueryOne executes a query and returns a single result
	QueryOne(ctx context.Context, query string, params map[string]any, result interface{}) error
	
	// Execute runs a query that doesn't return rows
	Execute(ctx context.Context, query string, params map[string]any) error
}

// NewExecutor creates a new executor instance
func NewExecutor(db *surrealdb.DB) Executor {
	return &executor{db: db}
}

type executor struct {
	db *surrealdb.DB
}

func (e *executor) Query(ctx context.Context, query string, params map[string]any, result interface{}) error {
	queryResults, err := surrealdb.Query[[]interface{}](ctx, e.db, query, params)
	if err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	if len(*queryResults) == 0 {
		return nil
	}
	// Convert the result to the target type
	data, err := json.Marshal((*queryResults)[0].Result)
	if err != nil {
		return fmt.Errorf("failed to marshal query result: %w", err)
	}
	return json.Unmarshal(data, result)
}

func (e *executor) QueryOne(ctx context.Context, query string, params map[string]any, result interface{}) error {
	// For QueryOne, we'll use the same implementation as Query but expect a single result
	var results []interface{}
	if err := e.Query(ctx, query, params, &results); err != nil {
		return err
	}
	if len(results) == 0 {
		return nil
	}
	// Convert the first result to the target type
	data, err := json.Marshal(results[0])
	if err != nil {
		return fmt.Errorf("failed to marshal query result: %w", err)
	}
	return json.Unmarshal(data, result)
}

func (e *executor) Execute(ctx context.Context, query string, params map[string]any) error {
	if _, err := surrealdb.Query[any](ctx, e.db, query, params); err != nil {
		return fmt.Errorf("query execution failed: %w", err)
	}
	return nil
}
