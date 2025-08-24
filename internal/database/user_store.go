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
	query := "SELECT * FROM user WHERE email = $email LIMIT 1"
	params := map[string]any{"email": email}

	// surrealdb.Query sends one or more statements and returns a slice of results.
	// The type of `results` is *[]surrealdb.QueryResult[[]models.User].
	results, err := surrealdb.Query[[]models.User](ctx, db, query, params)
	if err != nil {
		// If the driver returns an error, it encapsulates the failure reason.
		// This is the primary way to detect database or connection errors.
		return nil, fmt.Errorf("database query failed: %w", err)
	}

	// Defensive check: ensure the database returned at least one result structure.
	// This should not typically happen if `err` is nil, but it's robust to check.
	if results == nil || len(*results) == 0 {
		return nil, fmt.Errorf("database returned no result for the query")
	}

	// We sent one query, so we look at the first result.
	queryResult := (*results)[0]

	// The driver should have returned an error if the status was not "OK",
	// but we check it here as a final safeguard. There is no .Detail field.
	if queryResult.Status != "OK" {
		return nil, fmt.Errorf("database query returned non-OK status: %s", queryResult.Status)
	}

	// The `Result` field contains the slice of users returned by the SELECT statement.
	// If no user was found, this slice will be empty. This is the "not found" case.
	if len(queryResult.Result) == 0 {
		return nil, nil // User not found, which is not an application error.
	}

	// If we reach here, the query was successful and returned at least one user.
	return &queryResult.Result[0], nil
}
