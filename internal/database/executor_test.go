package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	surreal "github.com/surrealdb/surrealdb.go"
)

// TestUser is a local test struct that embeds models.User
// and adds a Password field for testing purposes
type TestUser struct {
	models.User
	Password string `json:"password,omitempty"`
}

func TestExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, cleanup := setupExecutorTestDB(t)
	defer cleanup()

	// Clean up any existing test users first
	_, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email CONTAINS 'test' AND email CONTAINS 'example.com'", nil)

	// Insert test users directly using Query with TestUser
	user1 := map[string]any{
		"email":    "test1@example.com",
		"name":     "Test User 1",
		"password": "password1",
	}
	_, err := surreal.Query[[]TestUser](ctx, db, "CREATE user SET name = $name, email = $email, password = $password", user1)
	require.NoError(t, err, "failed to create first test user")

	user2 := map[string]any{
		"email":    "test2@example.com",
		"name":     "Test User 2",
		"password": "password2",
	}
	_, err = surreal.Query[[]TestUser](ctx, db, "CREATE user SET name = $name, email = $email, password = $password", user2)
	require.NoError(t, err, "failed to create second test user")

	t.Run("Query - returns multiple results", func(t *testing.T) {
		// Test - query using a simpler approach that we know works
		query := "SELECT * FROM user WHERE email = $email1 OR email = $email2"
		params := map[string]any{
			"email1": "test1@example.com",
			"email2": "test2@example.com",
		}
		users, err := Query[TestUser](ctx, db, query, params)

		// Verify
		assert.NoError(t, err)
		assert.Len(t, users, 2, "should return exactly 2 test users")
	})
	t.Run("QueryOne - returns single result", func(t *testing.T) {
		// Test
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user WHERE email = $email",
			map[string]any{"email": "test1@example.com"})

		// Verify
		assert.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "Test User 1", user.Name)
	})

	t.Run("QueryOne - returns nil when not found", func(t *testing.T) {
		// Test
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user WHERE email = $email",
			map[string]any{"email": "nonexistent@example.com"})
		// Verify
		assert.NoError(t, err)
		assert.Nil(t, user)
	})
	t.Run("Execute - runs mutation queries", func(t *testing.T) {
		// Setup
		// Create a test user first
		testEmail := "testupdate@example.com"

		// Clean up any existing test user first
		_, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email = $email",
			map[string]any{"email": testEmail})

		// Create a test user with all required fields
		_, err := surreal.Query[any](ctx, db,
			"CREATE user SET name = $name, email = $email, password = 'testpass'",
			map[string]any{
				"name":  "Original Name",
				"email": testEmail,
			})
		require.NoError(t, err)

		// Test update
		_, err = Execute(ctx, db, "UPDATE user SET name = $name WHERE email = $email",
			map[string]any{
				"name":  "Updated Name",
				"email": testEmail,
			})
		// Verify
		assert.NoError(t, err)

		// Verify update
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user WHERE email = $email",
			map[string]any{"email": testEmail})
		assert.NoError(t, err)
		require.NotNil(t, user, "user should be found after update")
		assert.Equal(t, "Updated Name", user.Name, "user name should be updated")
	})
	t.Run("Query - handles errors", func(t *testing.T) {
		// Test with invalid query
		_, err := Query[TestUser](ctx, db, "INVALID QUERY", nil)
		assert.Error(t, err)
	})
	t.Run("QueryOne - automatically adds LIMIT 1", func(t *testing.T) {
		// This test verifies the hasLimitClause function works
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user",
			nil)

		// Should not error even though we didn't specify LIMIT
		assert.NoError(t, err)
		assert.NotNil(t, user) // At least one user should exist from previous tests
	})

	// Cleanup
	_, _ = surreal.Query[any](ctx, db, "DELETE user", nil)
}
func setupExecutorTestDB(t *testing.T) (*surreal.DB, func()) {
	t.Helper()
	// Initialize database connection
	ctx := context.Background()
	db, err := surreal.FromEndpointURLString(ctx, os.Getenv("SURREAL_URL"))
	require.NoError(t, err, "failed to connect to database")
	// Sign in
	_, err = db.SignIn(ctx, map[string]any{
		"user": os.Getenv("SURREAL_USER"),
		"pass": os.Getenv("SURREAL_PASS"),
	})
	require.NoError(t, err, "failed to sign in")
	// Use test namespace and database
	err = db.Use(ctx, os.Getenv("SURREAL_NS"), os.Getenv("SURREAL_DB"))
	require.NoError(t, err, "failed to use namespace/database")
	// Return connection and cleanup function
	return db, func() {
		// Clean up test data
		_, _ = surreal.Query[any](ctx, db, "DELETE user", nil)
		db.Close(ctx)
	}
}
