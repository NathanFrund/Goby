package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/nfrund/goby/internal/models"
	"github.com/surrealdb/surrealdb.go"
)

// UserStore encapsulates database operations for users.
type UserStore struct {
	db     *surrealdb.DB
	ns     string
	dbName string
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *surrealdb.DB, ns, dbName string) *UserStore {
	return &UserStore{db: db, ns: ns, dbName: dbName}
}

// FindUserByEmail queries for a single user by their email address.
func (s *UserStore) FindUserByEmail(ctx context.Context, email string) (*models.User, error) {
	// Ensure the correct namespace and database are selected for this operation.
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	query := "SELECT * FROM user WHERE email = $email"
	params := map[string]any{"email": email}

	// Use the QueryOne helper which handles all the result processing
	user, err := QueryOne[models.User](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return user, nil
}

func (s *UserStore) SignUp(ctx context.Context, user *models.User, password string) (string, error) {
	// Format matches the JavaScript SDK's implementation
	token, err := s.db.SignUp(ctx, map[string]interface{}{
		"ns":       s.ns,      // lowercase 'ns' to match JS SDK
		"db":       s.dbName,  // lowercase 'db' to match JS SDK
		"ac":       "account", // access control namespace
		"email":    user.Email,
		"password": password,
	})

	if err == nil && token != "" {
		log.Printf("Successfully signed up user %s. Token: %s", user.Email, token)
	}

	return token, err
}

func (s *UserStore) SignIn(ctx context.Context, user *models.User, password string) (string, error) {
	// Format matches the JavaScript SDK's implementation
	token, err := s.db.SignIn(ctx, map[string]interface{}{
		"ns":       s.ns,      // lowercase 'ns' to match JS SDK
		"db":       s.dbName,  // lowercase 'db' to match JS SDK
		"ac":       "account", // access control namespace
		"email":    user.Email,
		"password": password,
	})

	if err == nil && token != "" {
		log.Printf("Successfully signed in user %s. Token: %s", user.Email, token)
	}

	return token, err
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
func (s *UserStore) GenerateResetToken(ctx context.Context, email string) (string, error) {
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

	// Log the token and expiration for debugging
	log.Printf("DEBUG: Setting reset token for user %s: %q (expires: %s)", user.ID, token, expires)

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

// GetUserByResetToken finds and validates a reset token.
// It returns the user if the token is valid and not expired.
// The token is automatically invalidated after this call to prevent reuse.
func (s *UserStore) GetUserByResetToken(ctx context.Context, token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("reset token cannot be empty")
	}

	// Ensure the correct namespace and database are selected for this operation.
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// First, find the user with the given token
	// Note: Don't include semicolon as QueryOne will add LIMIT 1 if needed
	query := `
		SELECT * FROM user 
		WHERE resetToken = $reset_token
	`
	params := map[string]interface{}{
		"reset_token": token,
	}

	user, err := QueryOne[models.User](ctx, s.db, query, params)
	if err != nil {
		log.Printf("DEBUG: Error finding user by reset token: %v", err)
		return nil, fmt.Errorf("error finding user by reset token: %w", err)
	}

	// If no user found with this token
	if user == nil {
		log.Println("DEBUG: No user found with the provided reset token")
		return nil, nil
	}

	// Explicitly check if the token field exists and matches.
	// While the query should ensure this, this check prevents any ambiguity.
	if user.ResetToken == nil || *user.ResetToken != token {
		log.Println("DEBUG: User found, but reset token does not match or is nil")
		return nil, nil
	}

	// Check if token has expired
	if user.ResetTokenExpires == nil {
		log.Println("DEBUG: Reset token has no expiration time")
		return nil, nil
	}

	expires, err := time.Parse(time.RFC3339, *user.ResetTokenExpires)
	if err != nil {
		log.Printf("DEBUG: Error parsing reset token expiration: %v", err)
		return nil, fmt.Errorf("invalid reset token expiration format: %w", err)
	}

	if time.Now().After(expires) {
		log.Printf("DEBUG: Reset token expired at %s", *user.ResetTokenExpires)
		return nil, nil
	}

	// Invalidate the token to prevent reuse
	invalidateQuery := `
		UPDATE $id SET 
			resetToken = NONE,
			resetTokenExpires = NONE
	`
	invalidateParams := map[string]interface{}{
		"id": user.ID,
	}

	if err := Execute(ctx, s.db, invalidateQuery, invalidateParams); err != nil {
		// If we can't invalidate the token, we must not proceed.
		// This prevents the token from being reused if the database operation fails.
		return nil, fmt.Errorf("critical: failed to invalidate reset token for user %s: %w", user.ID, err)
	}

	log.Printf("DEBUG: Successfully validated reset token for user: %s", user.ID)
	return user, nil
}

// ResetPassword updates a user's password using a valid reset token
// The token is automatically invalidated as part of GetUserByResetToken
func (s *UserStore) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" {
		return errors.New("reset token cannot be empty")
	}
	if newPassword == "" {
		return errors.New("new password cannot be empty")
	}

	// Get and validate the user by token - this will also invalidate the token
	user, err := s.GetUserByResetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token: %w", err)
	}
	if user == nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Update the user's password using SurrealDB's built-in password hashing
	query := `
		UPDATE $id SET 
			password = crypto::argon2::generate($password)
	`
	params := map[string]interface{}{
		"id":       user.ID,
		"password": newPassword,
	}

	if err := Execute(ctx, s.db, query, params); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}
