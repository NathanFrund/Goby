package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/surrealdb/surrealdb.go"
)

// DBConfig holds database configuration
type DBConfig struct {
	URL      string
	Username string
	Password string
	NS       string
	DB       string
}

// NewConnection creates a new database connection
func NewConnection(ctx context.Context) (*surrealdb.DB, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := DBConfig{
		URL:      os.Getenv("SURREAL_URL"),
		Username: os.Getenv("SURREAL_USER"),
		Password: os.Getenv("SURREAL_PASS"),
		NS:       os.Getenv("SURREAL_NS"),
		DB:       os.Getenv("SURREAL_DB"),
	}

	// Validate required config
	if cfg.URL == "" || cfg.NS == "" || cfg.DB == "" {
		return nil, fmt.Errorf("missing required database configuration")
	}

	// Create database connection with timeout
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	db, err := surrealdb.FromEndpointURLString(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// log.Println(cfg.URL)
	// log.Println(cfg.Username)
	// log.Println(cfg.Password)
	// log.Println(cfg.NS)
	// log.Println(cfg.DB)

	// Sign in if credentials are provided
	if cfg.Username != "" && cfg.Password != "" {
		authData := &surrealdb.Auth{
			Username: cfg.Username,
			Password: cfg.Password,
		}

		if _, err := db.SignIn(ctx, authData); err != nil {
			db.Close(ctx)
			return nil, fmt.Errorf("database authentication failed: %w", err)
		}
	}

	// Use the specified namespace and database
	if err := db.Use(ctx, cfg.NS, cfg.DB); err != nil {
		db.Close(ctx)
		return nil, fmt.Errorf("failed to use namespace/database: %w", err)
	}

	return db, nil
}
