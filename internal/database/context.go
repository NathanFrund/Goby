package database

import (
	"context"
	"time"
)

// ContextKey is a custom type for context keys to avoid collisions.
type ContextKey string

const (
	// ContextKeyQueryTimeout allows overriding the default timeout for read queries.
	ContextKeyQueryTimeout ContextKey = "db_query_timeout"
	// ContextKeyExecuteTimeout allows overriding the default timeout for write operations.
	ContextKeyExecuteTimeout ContextKey = "db_execute_timeout"
)

// getTimeoutFromContext is a helper that retrieves a timeout duration from the context
// or returns a default value. It also returns a new context with the timeout applied
// and its corresponding cancellation function.
func getTimeoutFromContext(ctx context.Context, defaultTimeout time.Duration, key ContextKey) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := defaultTimeout
	if v, ok := ctx.Value(key).(time.Duration); ok && v > 0 {
		timeout = v
	}
	return context.WithTimeout(ctx, timeout)
}
