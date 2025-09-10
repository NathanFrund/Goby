package database

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/nfrund/goby/internal/config"
	"github.com/surrealdb/surrealdb.go"
)

// NewDB creates and configures a new SurrealDB connection.
func NewDB(ctx context.Context, cfg *config.Config) (*surrealdb.DB, error) {
	db, err := surrealdb.FromEndpointURLString(ctx, cfg.DBUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to surrealdb: %w", err)
	}

	authData := &surrealdb.Auth{
		Username: cfg.DBUser,
		Password: cfg.DBPass,
	}

	if _, err = db.SignIn(ctx, authData); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to sign in: %w", err)
	}

	if err = db.Use(ctx, cfg.DBNs, cfg.DBDb); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to use namespace/db: %w", err)
	}

	slog.Info("Successfully signed in to SurrealDB")
	return db, nil
}
