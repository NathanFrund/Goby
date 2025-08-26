package database

import (
	"context"
	"os"
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

	// Clean up any existing test users before starting
	_, err := surrealdb.Query[[]models.User](ctx, db, "DELETE user WHERE email = $email", map[string]interface{}{
		"email": "test@example.com",
	})
	require.NoError(t, err, "failed to clean up test users")

	// Test data
	testUser := &models.User{
		Email:    "test@example.com",
		Name:     "Test User",
	}

	t.Run("success - finds existing user", func(t *testing.T) {
		// Insert test user using Query to be consistent with FindUserByEmail
		query := "CREATE user SET email = $email, name = $name, password = $password"
		params := map[string]interface{}{
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

// setupTestDB creates a test database connection and returns a cleanup function
func setupTestDB(t *testing.T) (*surrealdb.DB, func()) {
	t.Helper()

	// Initialize database connection
	ctx := context.Background()
	db, err := surrealdb.FromEndpointURLString(ctx, os.Getenv("SURREAL_URL"))
	require.NoError(t, err, "failed to connect to database")

	// Sign in
	_, err = db.SignIn(ctx, map[string]interface{}{
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
		_, _ = surrealdb.Query[[]models.User](ctx, db, "DELETE user", nil)
		db.Close(ctx)
	}
}
