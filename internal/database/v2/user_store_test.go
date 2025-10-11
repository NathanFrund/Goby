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
	email := fmt.Sprintf("auth-%d@example.com", time.Now().UnixNano())
	name := "Auth User"
	password := "S3cureP@ssw0rd!"
	user := &domain.User{Name: &name, Email: email}

	// 1. Test SignUp
	token, err := store.SignUp(ctx, user, password)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
	t.Cleanup(func() { _ = store.Delete(ctx, user.ID.String()) })

	// 2. Test duplicate sign up
	_, err = store.SignUp(ctx, user, password)
	assert.ErrorIs(t, err, domain.ErrUserAlreadyExists)

	// 3. Test SignIn
	signInToken, err := store.SignIn(ctx, user, password)
	require.NoError(t, err)
	assert.NotEmpty(t, signInToken)

	// 4. Test wrong password
	_, err = store.SignIn(ctx, user, "wrongpassword")
	assert.Error(t, err)

	// 5. Test Authenticate
	authedUser, err := store.Authenticate(ctx, signInToken)
	require.NoError(t, err)
	require.NotNil(t, authedUser)
	assert.Equal(t, email, authedUser.Email)
}

func TestUserStore_PasswordReset(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	store, _, cleanup := setupUserStoreTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// 1. Create a user via SignUp to test with
	email := fmt.Sprintf("reset-%d@example.com", time.Now().UnixNano())
	name := "Reset User"
	initialPassword := "initial-pw-123"
	user := &domain.User{Name: &name, Email: email}
	_, err := store.SignUp(ctx, user, initialPassword)
	require.NoError(t, err)
	t.Cleanup(func() { _ = store.Delete(ctx, user.ID.String()) })

	// 2. Generate Token
	resetToken, err := store.GenerateResetToken(ctx, email)
	require.NoError(t, err)
	assert.NotEmpty(t, resetToken)

	// 3. Reset Password
	newPassword := "NewS3cureP@ssw0rd!"
	resetUser, err := store.ResetPassword(ctx, resetToken, newPassword)
	require.NoError(t, err)
	require.NotNil(t, resetUser)
	assert.Equal(t, user.ID, resetUser.ID)

	// 4. Verify token is invalidated by trying to use it again
	_, err = store.ResetPassword(ctx, resetToken, "another-new-password")
	assert.Error(t, err, "Resetting password with the same token should fail")

	// 5. Verify the new password works by signing in
	_, err = store.SignIn(ctx, user, newPassword)
	require.NoError(t, err, "Should be able to sign in with the new password")
}
