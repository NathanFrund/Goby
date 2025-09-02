package database

import (
	"context"
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
	Password *string `json:"password,omitempty"`
}

func TestExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, cleanup := setupTestDB(t)
	defer cleanup()

	t.Run("Query - returns multiple results", func(t *testing.T) {
		// Setup: Create specific users for this test and ensure they are cleaned up.
		t.Cleanup(func() {
			_, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email = 'test1@example.com' OR email = 'test2@example.com'", nil)
		})

		user1 := map[string]any{
			"email":    "test1@example.com",
			"name":     "Test User 1",
			"password": "password1",
		}
		_, err := surreal.Query[[]TestUser](ctx, db, "CREATE user SET name = $name, email = $email, password = $password", user1)
		require.NoError(t, err)

		user2 := map[string]any{
			"email":    "test2@example.com",
			"name":     "Test User 2",
			"password": "password2",
		}
		_, err = surreal.Query[[]TestUser](ctx, db, "CREATE user SET name = $name, email = $email, password = $password", user2)
		require.NoError(t, err)

		// Test
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
		// Setup
		email := "queryone@example.com"
		t.Cleanup(func() {
			_, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": email})
		})
		_, err := surreal.Query[[]TestUser](ctx, db, "CREATE user SET name = 'QueryOne User', email = $email, password = 'testpassword'", map[string]any{"email": email})
		require.NoError(t, err)

		// Test
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user WHERE email = $email",
			map[string]any{"email": email})

		// Verify
		assert.NoError(t, err)
		require.NotNil(t, user)
		require.NotNil(t, user.Name, "user name should not be nil")
		assert.Equal(t, "QueryOne User", *user.Name)
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
		testEmail := "testupdate@example.com"
		t.Cleanup(func() {
			_, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// Create a test user with all required fields
		_, err := surreal.Query[any](ctx, db,
			"CREATE user SET name = $name, email = $email, password = 'initialpassword'",
			map[string]any{
				"name":  "Original Name",
				"email": testEmail,
			})
		require.NoError(t, err)

		// Test update
		err = Execute(ctx, db, "UPDATE user SET name = $name WHERE email = $email",
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
		require.NotNil(t, user.Name, "user name should not be nil")
		assert.Equal(t, "Updated Name", *user.Name, "user name should be updated")
	})
	t.Run("Query - handles errors", func(t *testing.T) {
		// Test with invalid query
		_, err := Query[TestUser](ctx, db, "INVALID QUERY", nil)
		assert.Error(t, err)
	})
	t.Run("QueryOne - automatically adds LIMIT 1", func(t *testing.T) {
		// Setup
		t.Cleanup(func() { _, _ = surreal.Query[any](ctx, db, "DELETE user WHERE email = 'limit-test@example.com'", nil) })
		_, err := surreal.Query[[]TestUser](ctx, db, "CREATE user SET email = 'limit-test@example.com', name = 'Limit Test', password = 'limitpassword'", nil)
		require.NoError(t, err)

		// Test: Call QueryOne on a query that could return multiple rows
		user, err := QueryOne[TestUser](ctx, db,
			"SELECT * FROM user",
			nil)

		// Verify it doesn't fail and returns one user.
		assert.NoError(t, err)
		assert.NotNil(t, user, "should still return one user")
	})
}
