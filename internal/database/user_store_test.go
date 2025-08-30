package database

import (
	"context"
	"testing"

	"github.com/nfrund/goby/internal/config"
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
	cfg := config.New()

	// Create the store we are testing
	store := NewUserStore(db, cfg.DBNs, cfg.DBDb)

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

// testDBQuery is a helper to run a query and log the results for debugging
func TestSignUp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test database
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := config.New()

	// Create the store we are testing
	store := NewUserStore(db, cfg.DBNs, cfg.DBDb)

	t.Run("success - creates and signs up a new user", func(t *testing.T) {
		// Test data
		testEmail := "signup-test@example.com"
		testPassword := "securepassword123"
		testName := "Test SignUp User"

		// Cleanup after test
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// Create user using SignUp
		token, err := store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  testName,
		}, testPassword)

		// Verify results
		require.NoError(t, err, "SignUp should not return an error")
		assert.NotEmpty(t, token, "SignUp should return a non-empty token")

		// Verify user was created (only email is set during signup)
		user, err := store.FindUserByEmail(ctx, testEmail)
		require.NoError(t, err, "Should find the created user")
		assert.Equal(t, testEmail, user.Email)
		// Note: Name is not set during signup, only email and password are used
	})

	t.Run("error - duplicate email", func(t *testing.T) {
		// Test data
		testEmail := "duplicate-signup@example.com"
		testPassword := "securepassword123"

		// Create a user first
		_, err := store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  "First User",
		}, testPassword)
		require.NoError(t, err)

		// Try to create another user with the same email
		_, err = store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  "Duplicate User",
		}, "anotherpassword")

		// Verify error (actual error from SurrealDB)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The record access signup query failed")
	})
}

func testDBQuery(t *testing.T, ctx context.Context, db *surrealdb.DB, query string, params map[string]interface{}) {
	t.Helper()
	result, err := surrealdb.Query[any](ctx, db, query, params)
	if err != nil {
		t.Logf("Query failed: %v", err)
		return
	}
	t.Logf("Query result: %+v", result)
}

func TestCreateUser(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup test database and store, following the pattern from TestFindUserByEmail
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := config.New()

	// Log database connection details
	t.Logf("Connecting to database - URL: %s, NS: %s, DB: %s", cfg.DBUrl, cfg.DBNs, cfg.DBDb)

	store := NewUserStore(db, cfg.DBNs, cfg.DBDb)

	// Verify database connection and check permissions
	if err := db.Use(ctx, cfg.DBNs, cfg.DBDb); err != nil {
		t.Fatalf("Failed to use database: %v", err)
	}
	t.Log("Successfully connected to database")

	// Check database info and permissions
	info, err := db.Info(ctx)
	if err != nil {
		t.Logf("Failed to get database info: %v", err)
	} else {
		t.Logf("Database info: %+v", info)
	}

	// Check if we can query the user table
	testDBQuery(t, ctx, db, "INFO FOR TABLE user", nil)
	testDBQuery(t, ctx, db, "SELECT * FROM user LIMIT 10", nil)

	// Test creating a user directly with a query
	t.Run("direct query - create user", func(t *testing.T) {
		query := `
			CREATE user SET 
				email = $email, 
				name = $name, 
				password = crypto::argon2::generate($password)
		`
		params := map[string]interface{}{
			"email":    "direct-test@example.com",
			"name":     "Direct Test User",
			"password": "testpassword123",
		}

		result, err := surrealdb.Query[any](ctx, db, query, params)
		if err != nil {
			t.Fatalf("Direct query failed: %v", err)
		}
		t.Logf("Direct query result: %+v", result)

		// Cleanup
		_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", 
			map[string]any{"email": "direct-test@example.com"})
	})

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
			result, err := surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": newUser.Email})
			if err != nil {
				t.Logf("Cleanup failed to delete user: %v", err)
			} else {
				t.Logf("Cleanup deleted user: %+v", result)
			}
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
