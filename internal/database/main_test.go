package database

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/require"
	"github.com/surrealdb/surrealdb.go"
)

// setupTestDB creates a test database connection and returns a cleanup function.
// This is a shared helper for all tests in the database package.
func setupTestDB(t *testing.T) (*surrealdb.DB, func()) {
	t.Helper()

	cfg := testutils.ConfigForTests(t)

	ctx := context.Background()
	db, err := NewDB(ctx, cfg)
	require.NoError(t, err, "failed to connect to test database")

	// Return connection and a cleanup function to be deferred by the caller.
	return db, func() {
		// Clean up all user records after tests run to ensure a clean slate.
		_, _ = surrealdb.Query[any](context.Background(), db, "DELETE user", nil)
		db.Close(context.Background())
	}
}
