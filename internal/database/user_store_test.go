package database

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/surrealdb/surrealdb.go"
)

func TestFindUserByEmail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Test data
	testUser := &models.User{
		Email: "test@example.com",
		Name:  "Test User",
	}

	t.Run("success - finds existing user", func(t *testing.T) {
		// Setup: create a user for this specific test case.
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testUser.Email})
		})

		// Insert test user using Query to be consistent with FindUserByEmail
		query := "CREATE user SET email = $email, name = $name, password = $password"
		params := map[string]any{
			"email":    testUser.Email,
			"name":     testUser.Name,
			"password": "testpassword123",
		}

		// Execute the query
		results, err := surrealdb.Query[[]models.User](ctx, db, query, params)
		require.NoError(t, err, "failed to create test user")
		require.NotEmpty(t, results, "expected results from create query")
		require.NotEmpty(t, *results, "expected at least one result set")
		require.NotEmpty(t, (*results)[0].Result, "expected created user in result")

		createdUser := &(*results)[0].Result[0]

		// Test
		foundUser, err := FindUserByEmail(ctx, db, testUser.Email)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, foundUser)
		assert.Equal(t, createdUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Email, foundUser.Email)
		assert.Equal(t, testUser.Name, foundUser.Name)
	})

	t.Run("error - user not found", func(t *testing.T) {
		user, err := FindUserByEmail(ctx, db, "nonexistent@example.com")

		assert.Nil(t, user)
		assert.NoError(t, err) // No error expected for non-existent user, returns nil user
	})

	t.Run("empty email returns no error", func(t *testing.T) {
		user, err := FindUserByEmail(ctx, db, "")

		assert.Nil(t, user)
		assert.NoError(t, err) // No error expected for empty email
	})

	t.Run("database connection error", func(t *testing.T) {
		// Create a new context with cancellation to simulate connection error
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		// Test with canceled context to simulate connection error
		user, err := FindUserByEmail(cancelCtx, db, "test@example.com")
		assert.Nil(t, user)
		assert.Error(t, err, "should return error with canceled context")
	})
}
