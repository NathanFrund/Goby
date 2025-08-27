package database

import (
	"context"
	"fmt"

	"github.com/nfrund/goby/internal/models"
	"github.com/surrealdb/surrealdb.go"
)

// UserStore encapsulates database operations for users.
type UserStore struct {
	db *surrealdb.DB
}

// NewUserStore creates a new UserStore.
func NewUserStore(db *surrealdb.DB) *UserStore {
	return &UserStore{db: db}
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
