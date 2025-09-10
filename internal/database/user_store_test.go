package database

import (
	"context"
	"testing"
	"time"

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
	testUserName := "Test User"
	testUser := &models.User{
		Email: "test@example.com",
		Name:  &testUserName,
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
		assert.Equal(t, *testUser.Name, *foundUser.Name)
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

func TestSignIn(t *testing.T) {
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

	t.Run("success - signs in with correct credentials", func(t *testing.T) {
		// Test data
		signInUserName := "Test SignIn User"
		testUser := &models.User{
			Name:  &signInUserName,
			Email: "signin-test@example.com",
		}
		testPassword := "securepassword123"

		// Create a user first using SignUp
		_, err := store.SignUp(ctx, testUser, testPassword)
		require.NoError(t, err, "failed to create test user")

		// Cleanup after test
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testUser.Email})
		})

		// Test SignIn
		token, err := store.SignIn(ctx, testUser, testPassword)

		// Verify results
		require.NoError(t, err, "SignIn should not return an error with correct credentials")
		assert.NotEmpty(t, token, "SignIn should return a non-empty token")
	})

	t.Run("error - invalid password", func(t *testing.T) {
		// Test data
		testEmail := "signin-fail@example.com"
		testPassword := "correctpassword"

		// Create a user first using SignUp
		signInFailName := "Test SignIn Fail"
		_, err := store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  &signInFailName,
		}, testPassword)
		require.NoError(t, err, "failed to create test user")

		// Cleanup after test
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// Test SignIn with wrong password
		token, err := store.SignIn(ctx, &models.User{
			Email: testEmail,
		}, "wrongpassword")

		// Verify results
		assert.Error(t, err, "SignIn should return an error with incorrect password")
		assert.Empty(t, token, "Token should be empty on error")
	})

	t.Run("error - non-existent user", func(t *testing.T) {
		nonExistentEmail := "nonexistent@example.com"
		token, err := store.SignIn(ctx, &models.User{
			Email: nonExistentEmail,
		}, "somepassword")

		assert.Error(t, err, "SignIn should return an error for non-existent user")
		assert.Empty(t, token, "Token should be empty on error")
	})
}

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
			Name:  &testName,
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
		firstUserName := "First User"
		_, err := store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  &firstUserName,
		}, testPassword)
		require.NoError(t, err)

		// Try to create another user with the same email
		duplicateUserName := "Duplicate User"
		_, err = store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  &duplicateUserName,
		}, "anotherpassword")

		// Verify error (actual error from SurrealDB)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "The record access signup query failed")
	})
}

func TestPasswordResetFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	defer cleanup()
	cfg := config.New()
	store := NewUserStore(db, cfg.DBNs, cfg.DBDb)

	t.Run("success - full password reset flow", func(t *testing.T) {
		// Test data
		testEmail := "reset-test@example.com"
		initialPassword := "initialPassword123"
		newPassword := "newSecurePassword456"
		testName := "Reset Test User"

		// 1. Create a user to test with
		_, err := store.SignUp(ctx, &models.User{
			Email: testEmail,
			Name:  &testName,
		}, initialPassword)
		require.NoError(t, err, "failed to sign up initial user for reset test")

		// Cleanup the user after the test
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// 2. Generate a reset token for the user
		resetToken, err := store.GenerateResetToken(ctx, testEmail)
		require.NoError(t, err, "GenerateResetToken should not return an error")
		require.NotEmpty(t, resetToken, "GenerateResetToken should return a non-empty token")

		// Verify the token was stored correctly by fetching the user again
		user, err := store.FindUserByEmail(ctx, testEmail)
		require.NoError(t, err, "Should be able to find user by email")
		require.NotNil(t, user, "User should exist")
		require.NotNil(t, user.ResetToken, "Reset token should be set")
		require.NotEmpty(t, *user.ResetToken, "Reset token should not be empty")
		require.Equal(t, resetToken, *user.ResetToken, "Stored reset token should match generated token")

		// 3. Reset the password using the token
		_, err = store.ResetPassword(ctx, resetToken, newPassword)
		require.NoError(t, err, "ResetPassword should not return an error with a valid token")

		// 4. Verify the password was changed by signing in with the new password
		token, err := store.SignIn(ctx, &models.User{Email: testEmail}, newPassword)
		require.NoError(t, err, "SignIn with new password should succeed")
		assert.NotEmpty(t, token, "Should receive a token after signing in with the new password")

		// 5. Verify the old password no longer works
		_, err = store.SignIn(ctx, &models.User{Email: testEmail}, initialPassword)
		assert.Error(t, err, "SignIn with old password should fail")
	})

	t.Run("error - expired token", func(t *testing.T) {
		// Test data
		testEmail := "expired-token-test@example.com"
		initialPassword := "initialPassword123"
		newPassword := "newSecurePassword456"
		testName := "Expired Token Test User"

		// 1. Create user
		_, err := store.SignUp(ctx, &models.User{Email: testEmail, Name: &testName}, initialPassword)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// 2. Generate a reset token
		resetToken, err := store.GenerateResetToken(ctx, testEmail)
		require.NoError(t, err)
		require.NotEmpty(t, resetToken)

		// 3. Manually expire the token in the database
		user, err := store.FindUserByEmail(ctx, testEmail)
		require.NoError(t, err)
		require.NotNil(t, user)

		// Set expiration to yesterday
		expiredTime := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		_, err = surrealdb.Query[any](ctx, db, "UPDATE $id SET resetTokenExpires = $expires", map[string]any{
			"id":      user.ID,
			"expires": expiredTime,
		})
		require.NoError(t, err, "failed to manually expire token")

		// 4. Attempt to reset password with the expired token
		_, err = store.ResetPassword(ctx, resetToken, newPassword)
		require.Error(t, err, "ResetPassword should fail with an expired token")
		assert.Contains(t, err.Error(), "invalid or expired reset token")

		// 5. Verify the password was NOT changed
		// Sign in with new password should fail
		_, err = store.SignIn(ctx, &models.User{Email: testEmail}, newPassword)
		assert.Error(t, err, "SignIn with new password should fail")

		// Sign in with old password should succeed
		token, err := store.SignIn(ctx, &models.User{Email: testEmail}, initialPassword)
		assert.NoError(t, err, "SignIn with original password should still succeed")
		assert.NotEmpty(t, token, "Should receive a token when signing in with original password")
	})

	t.Run("error - invalid token", func(t *testing.T) {
		// Test data
		testEmail := "invalid-token-test@example.com"
		initialPassword := "initialPassword123"
		newPassword := "newSecurePassword456"
		testName := "Invalid Token Test User"

		// 1. Create user
		_, err := store.SignUp(ctx, &models.User{Email: testEmail, Name: &testName}, initialPassword)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// 2. Attempt to reset password with a completely invalid token
		invalidToken := "this-is-not-a-real-token"
		_, err = store.ResetPassword(ctx, invalidToken, newPassword)
		require.Error(t, err, "ResetPassword should fail with an invalid token")
		assert.Contains(t, err.Error(), "invalid or expired reset token")

		// 3. Verify the password was NOT changed
		// Sign in with new password should fail
		_, err = store.SignIn(ctx, &models.User{Email: testEmail}, newPassword)
		assert.Error(t, err, "SignIn with new password should fail")

		// Sign in with old password should succeed
		token, err := store.SignIn(ctx, &models.User{Email: testEmail}, initialPassword)
		assert.NoError(t, err, "SignIn with original password should still succeed")
		assert.NotEmpty(t, token, "Should receive a token when signing in with original password")
	})

	t.Run("error - token reuse", func(t *testing.T) {
		// Test data
		testEmail := "token-reuse-test@example.com"
		initialPassword := "initialPassword123"
		firstNewPassword := "newPassword456"
		secondNewPassword := "anotherPassword789"
		testName := "Token Reuse Test User"

		// 1. Create user
		_, err := store.SignUp(ctx, &models.User{Email: testEmail, Name: &testName}, initialPassword)
		require.NoError(t, err)
		t.Cleanup(func() {
			_, _ = surrealdb.Query[any](ctx, db, "DELETE user WHERE email = $email", map[string]any{"email": testEmail})
		})

		// 2. Generate a reset token
		resetToken, err := store.GenerateResetToken(ctx, testEmail)
		require.NoError(t, err)
		require.NotEmpty(t, resetToken)

		// 3. Use the token successfully for the first time
		_, err = store.ResetPassword(ctx, resetToken, firstNewPassword)
		require.NoError(t, err, "first password reset should succeed")

		// Verify the new password works
		_, err = store.SignIn(ctx, &models.User{Email: testEmail}, firstNewPassword)
		require.NoError(t, err, "sign in with the new password should work")

		// 4. Attempt to reuse the same token
		_, err = store.ResetPassword(ctx, resetToken, secondNewPassword)
		require.Error(t, err, "ResetPassword should fail on token reuse")

		// 5. Verify the password was NOT changed by the second attempt
		_, err = store.SignIn(ctx, &models.User{Email: testEmail}, secondNewPassword)
		assert.Error(t, err, "SignIn with the second new password should fail")
	})
}
