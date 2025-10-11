package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
)

// UserStore implements the domain.UserRepository interface using the new
// type-safe v2 database client.
type UserStore struct {
	client Client[domain.User]
	ns     string
	dbName string
}

// NewUserStore creates a new user repository with a type-safe client.
func NewUserStore(dbClient Client[domain.User], cfg config.Provider) domain.UserRepository {
	return &UserStore{
		client: dbClient,
		ns:     cfg.GetDBNs(),
		dbName: cfg.GetDBDb(),
	}
}

// Create inserts a new user record into the database.
func (s *UserStore) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	return s.client.Create(ctx, "user", user)
}

// GetByID retrieves a user by their unique ID.
func (s *UserStore) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.client.Select(ctx, id)
}

// FindUserByEmail retrieves a user by their email address. It is an alias for GetByEmail.
func (s *UserStore) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	// This method is required by the UserRepository interface.
	return s.GetByEmail(ctx, email)
}

// GetByEmail retrieves a user by their email address.
func (s *UserStore) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := "SELECT * FROM user WHERE email = $email"
	params := map[string]any{"email": email}
	return s.client.QueryOne(ctx, query, params)
}

// Update modifies an existing user record.
func (s *UserStore) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	if user.ID == nil || user.ID.String() == "" {
		return nil, NewDBError(ErrInvalidInput, "user ID is required for update")
	}
	return s.client.Update(ctx, user.ID.String(), user)
}

// Delete removes a user record from the database.
func (s *UserStore) Delete(ctx context.Context, id string) error {
	return s.client.Delete(ctx, id)
}

// GetUserWithPassword retrieves a user and their password hash by email.
// This is a special case that requires selecting a protected field.
func (s *UserStore) GetUserWithPassword(ctx context.Context, email string) (*domain.User, error) {
	// Note: This assumes the password field is protected by SurrealDB permissions
	// and this query is being run with appropriate (e.g., ROOT) scope.
	query := "SELECT *, password FROM user WHERE email = $email"
	params := map[string]any{"email": email}
	user, err := s.client.QueryOne(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get user with password: %w", err)
	}
	return user, nil
}

// --- Authentication Methods ---

// SignUp registers a new user atomically. It checks for an existing user and
// creates a new one with a hashed password in a single query.
func (s *UserStore) SignUp(ctx context.Context, user *domain.User, password string) (string, error) {
	db, err := s.client.DB()
	if err != nil {
		return "", fmt.Errorf("could not get database connection for sign up: %w", err)
	}

	// Use the driver's built-in SignUp method. It handles the scope-based user creation.
	// The returned token is discarded here; signing in is a separate, explicit step.
	// Format matches the JavaScript SDK's implementation
	_, err = db.SignUp(ctx, map[string]interface{}{
		"ns":       s.ns,
		"db":       s.dbName,
		"ac":       "account", // 'ac' for access control, as expected by the driver
		"email":    user.Email,
		"password": password,
	})

	if err != nil {
		// The driver returns a generic "signup query failed" error for duplicates.
		// We also check for the older messages for backward compatibility.
		if strings.Contains(err.Error(), "signup query failed") || strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "record with the ID") {
			return "", domain.ErrUserAlreadyExists
		}
		return "", fmt.Errorf("failed to sign up user: %w", err)
	}

	// After successful creation, sign the user in to get a session token.
	return s.SignIn(ctx, user, password)
}

// SignIn validates user credentials and returns a session token.
func (s *UserStore) SignIn(ctx context.Context, user *domain.User, password string) (string, error) {
	db, err := s.client.DB()
	if err != nil {
		return "", fmt.Errorf("could not get database connection for sign in: %w", err)
	}

	// Use the driver's built-in SignIn method, which is the most reliable way.
	data := map[string]interface{}{
		"ns":       s.ns,
		"db":       s.dbName,
		"ac":       "account", // 'ac' for access control, as expected by the driver
		"email":    user.Email,
		"password": password,
	}

	return db.SignIn(ctx, data)
}

// Authenticate validates a session token and returns the associated user.
func (s *UserStore) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	db, err := s.client.DB()
	if err != nil {
		return nil, fmt.Errorf("could not get database connection for authentication: %w", err)
	}

	// Use the driver's Authenticate method to validate the token and set the session.
	if err := db.Authenticate(ctx, token); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	// After successful authentication, get the current user's information from $auth.
	return s.client.QueryOne(ctx, "SELECT * FROM $auth", nil)
}

// --- Password Reset Methods ---

// GenerateResetToken creates a secure reset token and sets its expiration.
func (s *UserStore) GenerateResetToken(ctx context.Context, email string) (string, error) {
	token, err := generateSecureToken(32)
	if err != nil {
		return "", fmt.Errorf("error generating token: %w", err)
	}

	expires := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)

	// Use an atomic UPDATE query that finds the user and sets the token in one step.
	// This avoids a separate SELECT that would fail due to the missing password field in domain.User.
	query := `UPDATE user SET resetToken = $reset_token, resetTokenExpires = $expires WHERE email = $email RETURN AFTER`
	params := map[string]any{
		"email":       email,
		"reset_token": token,
		"expires":     expires,
	}

	// We only need to know if the update succeeded; we don't need the returned user object here.
	// QueryOne will return nil if no record was updated.
	updatedUser, err := s.client.QueryOne(ctx, query, params)
	if err != nil {
		return "", fmt.Errorf("failed to update user with reset token: %w", err)
	}
	if updatedUser == nil {
		return "", NewDBError(ErrNotFound, "user not found")
	}

	return token, nil
}

// ResetPassword performs an atomic password reset and invalidation of the token.
func (s *UserStore) ResetPassword(ctx context.Context, token, newPassword string) (*domain.User, error) {
	query := `
		UPDATE user SET
			password = crypto::argon2::generate($password),
			resetToken = NONE,
			resetTokenExpires = NONE
		WHERE resetToken = $target_token AND type::datetime(resetTokenExpires) > time::now() RETURN AFTER
	`
	params := map[string]any{
		"target_token": token,
		"password":     newPassword,
	}

	// We use QueryOne because the UPDATE statement might not find a match,
	// in which case it returns an empty result set.
	user, err := s.client.QueryOne(ctx, query, params)
	if err != nil {
		return nil, fmt.Errorf("database error during password reset: %w", err)
	}
	if user == nil {
		return nil, errors.New("invalid or expired reset link")
	}

	return user, nil
}

// generateSecureToken creates a cryptographically secure random token.
// This is a private helper function, co-located with its usage.
func generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}
