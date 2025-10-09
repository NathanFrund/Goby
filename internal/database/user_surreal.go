package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nfrund/goby/internal/domain"
	"github.com/surrealdb/surrealdb.go"
)

// SurrealUserStore encapsulates database operations for users using SurrealDB.
type SurrealUserStore struct {
	db     *surrealdb.DB
	ns     string
	dbName string
}

// NewSurrealUserStore creates a new SurrealUserStore.
func NewSurrealUserStore(db *surrealdb.DB, ns, dbName string) *SurrealUserStore {
	return &SurrealUserStore{db: db, ns: ns, dbName: dbName}
}

// FindUserByEmail queries for a single user by their email address.
func (s *SurrealUserStore) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	// Ensure the correct namespace and database are selected for this operation.
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	query := "SELECT * FROM user WHERE email = $email"
	params := map[string]any{"email": email}

	// Use the QueryOne helper which handles all the result processing
	user, err := QueryOne[domain.User](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return user, nil
}

func (s *SurrealUserStore) SignUp(ctx context.Context, user *domain.User, password string) (string, error) {
	// Format matches the JavaScript SDK's implementation
	token, err := s.db.SignUp(ctx, map[string]interface{}{
		"ns":       s.ns,      // lowercase 'ns' to match JS SDK
		"db":       s.dbName,  // lowercase 'db' to match JS SDK
		"ac":       "account", // access control namespace
		"email":    user.Email,
		"password": password,
	})

	// Check for a specific duplicate user error from the database driver.
	if err != nil && strings.Contains(err.Error(), "already exists") {
		return "", domain.ErrUserAlreadyExists // Return our domain-specific error.
	}

	if err == nil && token != "" {
		slog.Info(
			"Successfully signed up user",
			"email", user.Email,
			"token", token,
		)
	}

	return token, err
}

func (s *SurrealUserStore) SignIn(ctx context.Context, user *domain.User, password string) (string, error) {
	// Format matches the JavaScript SDK's implementation
	token, err := s.db.SignIn(ctx, map[string]interface{}{
		"ns":       s.ns,      // lowercase 'ns' to match JS SDK
		"db":       s.dbName,  // lowercase 'db' to match JS SDK
		"ac":       "account", // access control namespace
		"email":    user.Email,
		"password": password,
	})

	if err == nil && token != "" {
		slog.Info(
			"Successfully signed in user",
			"email", user.Email,
			"token", token,
		)
	}

	return token, err
}

// Authenticate validates a session token and returns the associated user.
func (s *SurrealUserStore) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	// Authenticate the connection using the provided token.
	// This validates the token against the 'account' scope.
	err := s.db.Authenticate(ctx, token) // This sets the auth context for subsequent queries on this connection.
	if err != nil {
		// This error indicates the token is invalid or expired.
		return nil, domain.ErrInvalidCredentials // Return a domain-specific error.
	}

	// After successful authentication, get the current user's information
	// using a direct query to get the current user
	users, err := Query[domain.User](ctx, s.db, "SELECT * FROM $auth", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get authenticated user: %w", err)
	}

	if len(users) == 0 || users[0].ID == nil {
		return nil, fmt.Errorf("no authenticated user found")
	}

	user := &users[0]

	// Clear the password before returning
	user.Password = ""

	return user, nil
}

// CreateUser creates a new user in the database.
// generateSecureToken creates a cryptographically secure random token
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// GenerateResetToken creates a secure reset token and sets its expiration
// CRITICAL: This uses Go's time package to set the expiration as an RFC3339 string.
// This is the "art that works" approach that avoids SurrealDB's time parsing errors.
func (s *SurrealUserStore) GenerateResetToken(ctx context.Context, email string) (string, error) {
	// Ensure the correct namespace and database are selected for this operation.
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return "", fmt.Errorf("failed to set database scope: %w", err)
	}

	// Find the user by email
	user, err := s.FindUserByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("error finding user: %w", err)
	}
	if user == nil {
		return "", errors.New("user not found")
	}

	// Generate a secure token
	token, err := generateSecureToken(32) // 32 bytes = 64 hex chars
	if err != nil {
		return "", fmt.Errorf("error generating token: %w", err)
	}

	// Set token expiration (24 hours from now)
	expires := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	slog.Debug(
		"Setting reset token for user",
		"user_id", user.ID,
		"token", token,
		"expires", expires,
	)

	// Update user with reset token and expiration
	query := `
		UPDATE $id SET
			resetToken = $reset_token,
			resetTokenExpires = $expires
	`
	params := map[string]interface{}{
		"id":          user.ID,
		"reset_token": token,
		"expires":     expires,
	}

	err = Execute(ctx, s.db, query, params)
	if err != nil {
		return "", fmt.Errorf("failed to update user with reset token: %w", err)
	}

	return token, nil
}

// GetUserByResetToken finds and validates a reset token using Go's time check.
// CRITICAL CHANGE: This function now serves only as a non-mutating validation step.
// The token is NOT invalidated here; invalidation is now handled atomically in ResetPassword.
func (s *SurrealUserStore) GetUserByResetToken(ctx context.Context, token string) (*domain.User, error) {
	if token == "" {
		return nil, errors.New("reset token cannot be empty")
	}

	// Ensure the correct namespace and database are selected for this operation.
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// Find the user with the given token
	query := `
		SELECT * FROM user
		WHERE resetToken = $reset_token
	`
	params := map[string]interface{}{
		"reset_token": token,
	}

	user, err := QueryOne[domain.User](ctx, s.db, query, params)
	if err != nil {
		slog.Debug("Error finding user by reset token", "error", err)
		return nil, fmt.Errorf("error finding user by reset token: %w", err)
	}

	// If no user found
	if user == nil || user.ResetToken == nil || *user.ResetToken != token {
		slog.Debug("No user found with the provided reset token or token mismatch")
		return nil, errors.New("invalid or expired reset link")
	}

	// Check if token has expired (using Go's time comparison - "art that works")
	if user.ResetTokenExpires == nil {
		slog.Debug("Reset token has no expiration time")
		return nil, errors.New("invalid or expired reset link")
	}

	expires, err := time.Parse(time.RFC3339, *user.ResetTokenExpires)
	if err != nil {
		slog.Debug("Error parsing reset token expiration", "error", err)
		return nil, fmt.Errorf("invalid reset token expiration format: %w", err)
	}

	if time.Now().After(expires) {
		slog.Debug("Reset token expired", "expires_at", *user.ResetTokenExpires)
		return nil, errors.New("invalid or expired reset link")
	}

	slog.Debug("Successfully validated reset token for user", "user_id", user.ID)
	// Return the user without modifying the database state
	return user, nil
}

// ResetPassword performs an atomic password reset.
// It uses a single query to verify the token's existence, check expiration, update the password,
// and invalidate the token. It does NOT call GetUserByResetToken to avoid conflict.
func (s *SurrealUserStore) ResetPassword(ctx context.Context, token, newPassword string) (*domain.User, error) {
	if token == "" || newPassword == "" {
		return nil, errors.New("token and password cannot be empty")
	}

	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// This single query performs the atomic update:
	// 1. It finds the user where the token matches AND is not expired.
	// 2. It sets the new password (using crypto::argon2::generate).
	// 3. It immediately invalidates the token by setting both fields to NONE.
	// 4. It returns the updated user object.
	// NOTE: We use type::datetime() on the expiration field to force the database to treat
	// the stored string as a datetime for comparison, resolving prior issues.
	query := `
		UPDATE user SET
			password = crypto::argon2::generate($password),
			resetToken = NONE,
			resetTokenExpires = NONE
		WHERE resetToken = $target_token AND type::datetime(resetTokenExpires) > time::now()
		RETURN AFTER; // Use RETURN AFTER to get the complete record
	`

	// Renaming the key to "target_token" avoids conflicts with SurrealDB's reserved 'token' keyword.
	params := map[string]interface{}{
		"target_token": token,
		"password":     newPassword,
	}

	// NOTE: Query is a hypothetical helper function that executes the SurrealQL.
	updatedUsers, err := Query[domain.User](ctx, s.db, query, params)
	if err != nil {
		slog.Error("Database error during atomic password reset", "error", err)
		return nil, fmt.Errorf("failed to execute atomic password reset: %w", err)
	}

	if len(updatedUsers) == 0 {
		// If no users were updated, the WHERE clause failed (token invalid or expired).
		slog.Warn("Password reset failed: Invalid or expired token")
		return nil, errors.New("invalid or expired reset link")
	}

	user := &updatedUsers[0]
	// Clear the password before returning (security best practice)
	user.Password = ""

	slog.Info("Successfully reset password", "user_id", user.ID)
	return user, nil
}

// WithTransaction creates a new transaction and executes the given function within it.
// If the function returns an error, the transaction is rolled back. Otherwise, it's committed.
// This implementation is specific to the surrealdb.go driver, which uses queries
// to manage transactions.
func (s *SurrealUserStore) WithTransaction(ctx context.Context, fn func(repo domain.UserRepository) error) error {
	// Begin a new transaction from the main DB connection.
	if _, err := surrealdb.Query[any](ctx, s.db, "BEGIN TRANSACTION;", nil); err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Use a deferred function with a flag to ensure rollback on any failure.
	var committed bool
	defer func() {
		if !committed {
			slog.WarnContext(ctx, "Rolling back transaction due to error or panic")
			if _, err := surrealdb.Query[any](ctx, s.db, "CANCEL TRANSACTION;", nil); err != nil {
				slog.ErrorContext(ctx, "CRITICAL: failed to cancel (rollback) transaction", "error", err)
			}
		}
	}()

	// Execute the provided function. We pass the same store `s` because its
	// underlying connection is now in a transactional state.
	if err := fn(s); err != nil {
		return err // The defer will handle the rollback.
	}

	// If the function succeeds, commit the transaction.
	if _, err := surrealdb.Query[any](ctx, s.db, "COMMIT TRANSACTION;", nil); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	committed = true
	return nil
}
