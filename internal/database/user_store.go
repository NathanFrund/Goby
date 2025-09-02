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
	expires := time.Now().Add(24 * time.Hour).Format(time.RFC3339)

	// Update user with reset token and expiration
	query := `
		UPDATE $id SET 
			resetToken = $token,
			resetTokenExpires = $expires
	`
	params := map[string]interface{}{
		"id":    user.ID,
		"token": token,
		"expires": expires,
	}

	_, err = surrealdb.Query[any](ctx, s.db, query, params)
	if err != nil {
		return "", fmt.Errorf("failed to update user with reset token: %w", err)
	}

	return token, nil
}

// GetUserByResetToken finds a user by their reset token if it's still valid
func (s *UserStore) GetUserByResetToken(ctx context.Context, token string) (*models.User, error) {
	if token == "" {
		return nil, errors.New("reset token cannot be empty")
	}

	query := `
		SELECT * FROM user 
		WHERE resetToken = $token 
		AND resetTokenExpires > time::now()
		LIMIT 1
	`
	params := map[string]interface{}{
		"token": token,
	}

	user, err := QueryOne[models.User](ctx, s.db, query, params)
	if err != nil {
		return nil, fmt.Errorf("error finding user by reset token: %w", err)
	}

	return user, nil
}

// ResetPassword updates a user's password using a valid reset token
func (s *UserStore) ResetPassword(ctx context.Context, token, newPassword string) error {
	if token == "" {
		return errors.New("reset token cannot be empty")
	}
	if newPassword == "" {
		return errors.New("new password cannot be empty")
	}

	// Get user by token (this also checks if token is expired)
	user, err := s.GetUserByResetToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token: %w", err)
	}

	// Update the password and clear the reset token
	query := `
		UPDATE $id SET 
			password = crypto::argon2::generate($password),
			resetToken = NONE,
			resetTokenExpires = NONE
	`
	params := map[string]interface{}{
		"id":       user.ID,
		"password": newPassword,
	}

	_, err = surrealdb.Query[any](ctx, s.db, query, params)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// CreateUser creates a new user in the database.
func (s *UserStore) CreateUser(ctx context.Context, user *models.User, password string) (*models.User, error) {
	// First check if user with this email already exists
	existingUser, err := s.FindUserByEmail(ctx, user.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", user.Email)
	}

	// Set the namespace and database for the scope of this operation
	if err := s.db.Use(ctx, s.ns, s.dbName); err != nil {
		log.Printf("Failed to set database scope (ns: %s, db: %s): %v", s.ns, s.dbName, err)
		return nil, fmt.Errorf("failed to set database scope: %w", err)
	}

	// Log the user creation attempt
	log.Printf("Attempting to create user with email: %s", user.Email)

	// Create the user using a direct query
	query := `
		CREATE user SET 
			email = $email, 
			name = $name, 
			password = crypto::argon2::generate($password)
	`
	params := map[string]interface{}{
		"email":    user.Email,
		"name":     user.Name,
		"password": password,
	}

	_, err = surrealdb.Query[any](ctx, s.db, query, params)
	if err != nil {
		log.Printf("Failed to create user %s: %v", user.Email, err)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Return the created user by querying it back
	createdUser, err := s.FindUserByEmail(ctx, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch created user: %w", err)
	}

	return createdUser, nil
}
