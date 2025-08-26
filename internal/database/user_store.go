package database

import (
	"context"
	"fmt"

	"github.com/nfrund/goby/internal/models"
	"github.com/surrealdb/surrealdb.go"
)

// FindUserByEmail queries for a single user by their email address.
// This function demonstrates a functional/procedural approach, taking the db connection
// as a direct argument rather than being a method on a store/repository struct.
func FindUserByEmail(ctx context.Context, db *surrealdb.DB, email string) (*models.User, error) {
	query := "SELECT * FROM user WHERE email = $email"
	params := map[string]any{"email": email}

	// Use the QueryOne helper which handles all the result processing
	user, err := QueryOne[models.User](ctx, db, query, params)
	if err != nil {
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	return user, nil
}
