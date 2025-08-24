package database

import (
	"context"
	"fmt"
	"log"

	"github.com/nfrund/goby/internal/config"
	"github.com/surrealdb/surrealdb.go"
)

// NewDB creates and configures a new SurrealDB connection.
func NewDB(ctx context.Context, cfg *config.Config) (*surrealdb.DB, error) {
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	if err = db.Use(ctx, cfg.DBNs, cfg.DBDb); err != nil {
		return nil, fmt.Errorf("failed to use namespace/db: %w", err)
	}

	authData := &surrealdb.Auth{
		Username: cfg.DBUser,
		Password: cfg.DBPass,
	}

	if _, err = db.SignIn(ctx, authData); err != nil {
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	log.Println("Successfully signed in to SurrealDB")
	return db, nil
}
