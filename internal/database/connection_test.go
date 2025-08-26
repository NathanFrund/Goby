package database

import (
	"context"
	"os"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	// Load test environment variables from .env.test
	if err := godotenv.Load("../../.env.test"); err != nil {
		panic("Error loading .env.test file")
	}
	os.Exit(m.Run())
}

func TestNewConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	t.Run("successful connection", func(t *testing.T) {
		ctx := context.Background()
		db, err := NewConnection(ctx)

		if err != nil {
			t.Fatalf("failed to connect: %v", err)
		}
		if db == nil {
			t.Fatal("expected db connection, got nil")
		}

		// Rest of your test remains the same...
		// ...
	})

	t.Run("missing required config", func(t *testing.T) {
		// Save original values
		origURL := os.Getenv("SURREAL_URL")
		origNS := os.Getenv("SURREAL_NS")
		origDB := os.Getenv("SURREAL_DB")

		// Clear required environment variables
		os.Unsetenv("SURREAL_URL")
		os.Unsetenv("SURREAL_NS")
		os.Unsetenv("SURREAL_DB")

		// Restore original values after test
		defer func() {
			os.Setenv("SURREAL_URL", origURL)
			os.Setenv("SURREAL_NS", origNS)
			os.Setenv("SURREAL_DB", origDB)
		}()

		ctx := context.Background()
		db, err := NewConnection(ctx)

		if err == nil {
			t.Error("expected error for missing config, got nil")
		}
		if db != nil {
			t.Error("expected nil db on error")
		}
	})
}

// Rest of your test file...
