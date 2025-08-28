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

	// Create the store we are testing
	store := NewUserStore(db)

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
		foundUser, err := store.FindUserByEmail(ctx, testUser.Email)

		// Assert
		require.NoError(t, err)
		require.NotNil(t, foundUser)
		assert.Equal(t, createdUser.ID, foundUser.ID)
		assert.Equal(t, testUser.Email, foundUser.Email)
		assert.Equal(t, testUser.Name, foundUser.Name)
	})

	t.Run("error - user not found", func(t *testing.T) {
		user, err := store.FindUserByEmail(ctx, "nonexistent@example.com")

		assert.Nil(t, user)
		assert.NoError(t, err) // No error expected for non-existent user, returns nil user
	})

	t.Run("empty email returns no error", func(t *testing.T) {
		user, err := store.FindUserByEmail(ctx, "")

		assert.Nil(t, user)
		assert.NoError(t, err) // No error expected for empty email
	})

	t.Run("database connection error", func(t *testing.T) {
		// Create a new context with cancellation to simulate a connection error
		cancelCtx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel the context immediately

		// Test with canceled context
		user, err := store.FindUserByEmail(cancelCtx, "test@example.com")
		assert.Nil(t, user)
		assert.Error(t, err, "should return error with canceled context")
	})
}

func TestCreateUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test database and store, following the pattern from TestFindUserByEmail
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	store := NewUserStore(db)

	t.Run("success - creates a new user", func(t *testing.T) {
		// Arrange: Define the user data we want to create.
		newUser := &models.User{
			Email: "create-test@example.com",
			Name:  "Create Test User",
		}
		password := "a-strong-password-123"

		// Use t.Cleanup to ensure the test user is deleted after this sub-test runs,
		// keeping our tests isolated from each other.
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": newUser.Email})
		})

		// Act: Call the method we are testing.
		createdUser, err := store.CreateUser(ctx, newUser, password)

		// Assert: Verify the outcome.
		require.NoError(t, err)
		require.NotNil(t, createdUser)

		// Check that the returned user has the correct data and a database-generated ID.
		assert.NotEmpty(t, createdUser.ID, "ID should be set by the database")
		assert.Equal(t, newUser.Email, createdUser.Email)
		assert.Equal(t, newUser.Name, createdUser.Name)
	})

	t.Run("error - email already exists", func(t *testing.T) {
		// Arrange: Create an initial user that we will try to duplicate.
		initialUser := &models.User{
			Email: "duplicate-test@example.com",
			Name:  "Initial User",
		}
		password := "some-password"

		_, err := store.CreateUser(ctx, initialUser, password)
		require.NoError(t, err, "failed to create the initial user for the test")

		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": initialUser.Email})
		})

		// Act: Attempt to create another user with the same email.
		duplicateUser, err := store.CreateUser(ctx, initialUser, "another-password")

		// Assert: Verify that we get an error and a nil user.
		assert.Error(t, err, "expected an error when creating a user with a duplicate email")
		assert.Nil(t, duplicateUser, "the returned user should be nil on error")
	})
}
