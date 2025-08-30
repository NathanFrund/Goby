package database

import (
	"context"
	"fmt"
	"log"

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
