package v2

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUser is a local test struct that embeds domain.User
// and adds a Password field for testing purposes.
type TestUser struct {
	domain.User
	Password string `json:"password"`
}

// setupUserStoreTest is a test helper that creates a connection to the test database
// and returns a fully initialized UserStore along with a cleanup function.
func setupUserStoreTest(t *testing.T) (*UserStore, Client[TestUser], func()) {
	cfg := testutils.ConfigForTests(t)
	conn := NewConnection(cfg)
	err := conn.Connect(context.Background())
	require.NoError(t, err, "Failed to connect to test database with new connection manager")
	conn.StartMonitoring()

	// Client for the UserStore, typed to the domain model.
	domainClient, err := NewClient[domain.User](conn, cfg)
	require.NoError(t, err)

	// Client for the test functions, typed to the test model to handle the password field.
	testClient, err := NewClient[TestUser](conn, cfg)
	require.NoError(t, err)

	store := NewUserStore(domainClient, cfg).(*UserStore)

	cleanup := func() {
		conn.Close(context.Background())
	}
	return store, testClient, cleanup
}

func TestUserStore_CRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, client, cleanup := setupUserStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Create a user for the test
	name := "CRUD User"
	email := fmt.Sprintf("crud-%d@example.com", time.Now().UnixNano())
	userToCreate := TestUser{
		User: domain.User{
			Name:  &name,
			Email: email,
		},
		Password: "some-initial-password", // Provide a password for direct creation
	}
	createdUser, err := client.Create(ctx, "user", &userToCreate)
	require.NoError(t, err)
	t.Cleanup(func() { _ = client.Delete(ctx, createdUser.ID.String()) })

	// 2. Test GetByID and FindUserByEmail
	fetchedUser, err := store.GetByID(ctx, createdUser.ID.String())
	require.NoError(t, err)
	require.NotNil(t, fetchedUser)
	assert.Equal(t, createdUser.ID, fetchedUser.ID)

	fetchedByEmail, err := store.FindUserByEmail(ctx, email)
	require.NoError(t, err)
	require.NotNil(t, fetchedByEmail)
	assert.Equal(t, createdUser.ID, fetchedByEmail.ID)

	// Test FindUserByEmail with empty string
	fetchedByEmptyEmail, err := store.FindUserByEmail(ctx, "")
	require.NoError(t, err, "FindUserByEmail with empty string should not return an error")
	assert.Nil(t, fetchedByEmptyEmail, "FindUserByEmail with empty string should return a nil user")

	// 3. Test Update
	updatedName := "Updated CRUD User"
	createdUser.Name = &updatedName
	updatedUser, err := store.Update(ctx, &createdUser.User)
	require.NoError(t, err)
	require.NotNil(t, updatedUser)
	assert.Equal(t, updatedName, *updatedUser.Name)

	// 4. Test Delete
	err = store.Delete(ctx, createdUser.ID.String())
	require.NoError(t, err)
	deletedUser, err := store.GetByID(ctx, createdUser.ID.String())
	require.Error(t, err)
	assert.Nil(t, deletedUser)
}

func TestUserStore_Authentication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, _, cleanup := setupUserStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test data
	t.Run("SignUp and SignIn Flow", func(t *testing.T) {
		email := fmt.Sprintf("auth-%d@example.com", time.Now().UnixNano())
		name := "Auth User"
		password := "S3cureP@ssw0rd!"
		user := &domain.User{Name: &name, Email: email}

		// 1. Test Successful SignUp
		token, err := store.SignUp(ctx, user, password)
		require.NoError(t, err, "SignUp should succeed for a new user")
		assert.NotEmpty(t, token, "SignUp should return a session token")
		t.Cleanup(func() {
			// The user ID is not populated by the current SignUp flow, so we find by email to delete.
			if u, _ := store.FindUserByEmail(context.Background(), email); u != nil && u.ID != nil {
				_ = store.Delete(context.Background(), u.ID.String())
			}
		})

		// 2. Test Duplicate SignUp
		_, err = store.SignUp(ctx, user, password)
		assert.ErrorIs(t, err, domain.ErrUserAlreadyExists, "SignUp should fail for an existing user")

		// 3. Test Successful SignIn
		signInToken, err := store.SignIn(ctx, user, password)
		require.NoError(t, err, "SignIn should succeed with correct credentials")
		assert.NotEmpty(t, signInToken, "SignIn should return a token")

		// 4. Test Failed SignIn (Wrong Password)
		_, err = store.SignIn(ctx, user, "wrongpassword")
		assert.Error(t, err, "SignIn should fail with an incorrect password")

		// 5. Test Successful Authenticate
		authedUser, err := store.Authenticate(ctx, signInToken)
		require.NoError(t, err, "Authenticate should succeed with a valid token")
		require.NotNil(t, authedUser, "Authenticated user should not be nil")
		assert.Equal(t, email, authedUser.Email, "Authenticated user's email should match")
	})

	t.Run("Authentication with invalid token", func(t *testing.T) {
		// Test Authenticate with a completely invalid token
		_, err := store.Authenticate(ctx, "this-is-not-a-valid-token")
		assert.Error(t, err, "Authenticate should fail with an invalid token")
	})
}

func TestUserStore_PasswordReset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, client, cleanup := setupUserStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Create a user via SignUp to test with
	email := fmt.Sprintf("reset-%d@example.com", time.Now().UnixNano())
	name := "Reset User"
	initialPassword := "initial-pw-123"

	// Use the test client to create the user directly, avoiding SignUp's side effects.
	userToCreate := TestUser{
		User:     domain.User{Name: &name, Email: email},
		Password: fmt.Sprintf("crypto::argon2::generate('%s')", initialPassword),
	}
	createdUser, err := client.Create(ctx, "user", &userToCreate)
	require.NoError(t, err)
	require.NotNil(t, createdUser.ID, "Created user should have an ID")
	t.Cleanup(func() { _ = store.Delete(ctx, createdUser.ID.String()) })

	// 2. Generate Token
	resetToken, err := store.GenerateResetToken(ctx, email)
	require.NoError(t, err)
	assert.NotEmpty(t, resetToken)

	// 3. Reset Password
	newPassword := "NewS3cureP@ssw0rd!"
	resetUser, err := store.ResetPassword(ctx, resetToken, newPassword)
	require.NoError(t, err)
	require.NotNil(t, resetUser)
	assert.Equal(t, createdUser.ID, resetUser.ID)

	// 4. Verify token is invalidated by trying to use it again
	_, err = store.ResetPassword(ctx, resetToken, "another-new-password")
	assert.Error(t, err, "Resetting password with the same token should fail")

	// 5. Verify the new password works by signing in
	// Create a new user object for sign-in, as the createdUser object doesn't contain the password.
	signInUser := &domain.User{Email: email}
	_, err = store.SignIn(ctx, signInUser, newPassword)
	require.NoError(t, err, "Should be able to sign in with the new password")

	t.Run("fails with expired token", func(t *testing.T) {
		// Generate a new token
		expiredToken, err := store.GenerateResetToken(ctx, email)
		require.NoError(t, err)

		// Manually expire the token in the database
		expiredTime := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		_, err = client.Query(ctx, "UPDATE user SET resetTokenExpires = $expires WHERE email = $email", map[string]any{
			"email":   email,
			"expires": expiredTime,
		})
		require.NoError(t, err, "failed to manually expire token")

		// Attempt to reset password with the expired token
		_, err = store.ResetPassword(ctx, expiredToken, "password-from-expired-token")
		require.Error(t, err, "ResetPassword should fail with an expired token")
		assert.Contains(t, err.Error(), "invalid or expired reset link")
	})

	t.Run("fails with invalid token", func(t *testing.T) {
		_, err := store.ResetPassword(ctx, "this-is-a-fake-token", "some-password")
		require.Error(t, err, "ResetPassword should fail with a fake token")
	})
}
