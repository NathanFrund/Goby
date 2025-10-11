package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDB is a test helper that creates a connection to the test database
// and returns the connection along with a cleanup function.
func setupTestDB(t *testing.T) (*Connection, config.Provider, func()) {
	cfg := testutils.ConfigForTests(t)

	// Use the new Connection manager for tests
	conn := NewConnection(cfg)
	err := conn.Connect(context.Background())
	require.NoError(t, err, "Failed to connect to test database with new connection manager")
	conn.StartMonitoring()

	cleanup := func() {
		conn.Close(context.Background())
	}
	return conn, cfg, cleanup
}

func TestClient(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, cfg, cleanup := setupTestDB(t)
	defer cleanup()

	client, err := NewClient[TestUser](conn, cfg)
	require.NoError(t, err)

	t.Run("Create and Select", func(t *testing.T) {
		// Setup
		name := "Create User"
		email := "create@example.com"
		password := "password"
		userToCreate := TestUser{
			User: domain.User{
				Name:  &name, // domain.User fields are pointers
				Email: email, // domain.User.Email is a string
			},
			Password: password,
		}

		// Test Create
		createdUser, err := client.Create(ctx, "user", userToCreate)
		require.NoError(t, err)
		require.NotNil(t, createdUser)
		require.NotEmpty(t, createdUser.ID, "Created user should have an ID")

		// Cleanup
		defer client.Delete(ctx, createdUser.ID.String())

		// Test Select
		selectedUser, err := client.Select(ctx, createdUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, selectedUser)
		assert.Equal(t, createdUser.ID, selectedUser.ID)
		assert.Equal(t, *userToCreate.Name, *selectedUser.Name) // Dereference pointers for comparison
		assert.Equal(t, userToCreate.Email, selectedUser.Email)
		// The client is typed to TestUser, so the password field should be populated.
		assert.Equal(t, userToCreate.Password, selectedUser.Password)
	})

	t.Run("Update", func(t *testing.T) {
		// Setup
		name := "Update User"
		email := "update@example.com"
		password := "password"
		userToCreate := TestUser{
			User: domain.User{
				Name:  &name,
				Email: email,
			},
			Password: password,
		}
		createdUser, err := client.Create(ctx, "user", userToCreate)
		require.NoError(t, err)
		defer client.Delete(ctx, createdUser.ID.String())

		// Test Update
		updatedName := "Updated Name"
		updateData := map[string]any{
			"name": updatedName,
		}
		updatedUser, err := client.Update(ctx, createdUser.ID.String(), updateData)
		require.NoError(t, err)
		require.NotNil(t, updatedUser)
		assert.Equal(t, updatedName, *updatedUser.Name)

		// Verify with Select
		verifiedUser, err := client.Select(ctx, createdUser.ID.String())
		require.NoError(t, err)
		require.NotNil(t, verifiedUser)
		assert.Equal(t, updatedName, *verifiedUser.Name)
	})

	t.Run("Delete", func(t *testing.T) {
		// Setup
		name := "Delete User"
		email := "delete@example.com"
		password := "password"
		userToCreate := TestUser{
			User: domain.User{
				Name:  &name,
				Email: email,
			},
			Password: password,
		}
		createdUser, err := client.Create(ctx, "user", userToCreate)
		require.NoError(t, err)

		// Test Delete
		err = client.Delete(ctx, createdUser.ID.String())
		require.NoError(t, err)

		// Verify
		deletedUser, err := client.Select(ctx, createdUser.ID.String())
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Nil(t, deletedUser)
	})

	t.Run("QueryOne - returns nil when not found", func(t *testing.T) {
		// Test
		user, err := client.QueryOne(ctx,
			"SELECT * FROM user WHERE email = $email",
			map[string]any{"email": "nonexistent@example.com"})
		// Verify
		assert.NoError(t, err)
		assert.Nil(t, user)
	})

	t.Run("Execute - runs mutation queries", func(t *testing.T) {
		// Setup
		email := "execute@example.com"
		createdUser, err := client.Create(ctx, "user", map[string]any{"name": "Execute User", "email": email, "password": "password"})
		require.NoError(t, err)
		defer client.Delete(ctx, createdUser.ID.String())

		// Test update via Execute
		err = client.Execute(ctx, "UPDATE user SET name = $name WHERE email = $email",
			map[string]any{"name": "Updated Execute User", "email": email})
		assert.NoError(t, err)

		// Verify update
		user, err := client.QueryOne(ctx, "SELECT * FROM user WHERE email = $email", map[string]any{"email": email})
		assert.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, "Updated Execute User", *user.Name)
	})
}

func TestClient_Timeouts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	conn, cfg, cleanup := setupTestDB(t)
	defer cleanup()

	// Use a generic client for this test
	client, err := NewClient[any](conn, cfg)
	require.NoError(t, err)

	// Retrieve the configured execution timeout to use as the test boundary.
	executeTimeout := cfg.GetDBExecuteTimeout()
	require.Greater(t, executeTimeout, time.Duration(0), "DB_EXECUTE_TIMEOUT must be configured for this test")

	t.Run("execute succeeds when faster than timeout", func(t *testing.T) {
		// 1. Define the timeout for the context, equal to the configured limit.
		ctx, cancel := context.WithTimeout(context.Background(), executeTimeout)
		defer cancel()

		// 2. Sleep for a duration less than the context's timeout
		sleepDuration := executeTimeout / 2
		// Dynamically construct the query string as SurrealDB does not allow
		// parameters ($duration) for the SLEEP statement.
		// Use milliseconds to avoid floating point issues with the `s` suffix.
		query := fmt.Sprintf("SLEEP %dms;", sleepDuration.Milliseconds())

		// No need for parameters map since the value is embedded.
		err := client.Execute(ctx, query, nil)
		assert.NoError(t, err, "Execute should succeed when it's faster than the timeout")
	})

	t.Run("execute fails when slower than timeout", func(t *testing.T) {
		// 1. Set the context timeout to the configured limit. This is the deadline.
		ctx, cancel := context.WithTimeout(context.Background(), executeTimeout)
		defer cancel()

		// 2. Sleep for a duration slightly longer than the context's timeout
		sleepDuration := executeTimeout + (100 * time.Millisecond)
		query := fmt.Sprintf("SLEEP %dms;", sleepDuration.Milliseconds())

		err := client.Execute(ctx, query, nil)
		require.Error(t, err, "Execute should fail when it's slower than the timeout")
		// The error from the driver includes extra text, so we check for the core message.
		assert.Contains(t, err.Error(), context.DeadlineExceeded.Error(), "Error should be a context deadline exceeded error")
	})
}
