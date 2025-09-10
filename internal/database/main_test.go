package database

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/logging"
	"github.com/stretchr/testify/require"
	"github.com/surrealdb/surrealdb.go"
)

// TestMain is a special function that runs before any other tests in this package.
// It's used here to load the test-specific environment variables from `.env.test`.
func TestMain(m *testing.M) {
	// Attempt to load the .env.test file from the project root.
	// This is safe to fail if running in a CI environment where vars are set directly.
	if err := godotenv.Load("../../.env.test"); err != nil {
		// No need to log here, as the config will fall back to env vars.
	}
	logging.New() // Initialize logger for tests
	os.Exit(m.Run())
}

// setupTestDB creates a test database connection and returns a cleanup function.
// This is a shared helper for all tests in the database package.
func setupTestDB(t *testing.T) (*surrealdb.DB, func()) {
	t.Helper()

	// We use the same config logic as the main application for consistency.
	cfg := config.New()

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
