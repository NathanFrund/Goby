package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/models"
	"github.com/nfrund/goby/internal/database"
)

func createTestUser(ctx context.Context, store *database.UserStore) error {
	email := "test@example.com"
	if len(os.Args) > 2 {
		email = os.Args[2] // Allow email to be passed as second argument
	}

	user := &models.User{
		Email: email,
		Name:  "Test User",
	}

	token, err := store.SignUp(ctx, user, "testpassword123")
	if err != nil {
		return fmt.Errorf("failed to create test user: %w", err)
	}

	log.Printf("Successfully created test user with email: %s\n", email)
	log.Printf("Authentication token: %s\n", token)
	return nil
}

func main() {
	ctx := context.Background()
	cfg := config.New()

	db, err := database.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close(ctx)

	userStore := database.NewUserStore(db, cfg.DBNs, cfg.DBDb)

	// Check for command line arguments
	if len(os.Args) > 1 {
		switch strings.ToLower(os.Args[1]) {
		case "create-test-user":
			if err := createTestUser(ctx, userStore); err != nil {
				log.Fatalf("Error: %v", err)
			}
			return
		default:
			log.Fatalf("Unknown command: %s", os.Args[1])
		}
	}

	// Default behavior: find a user
	user, err := userStore.FindUserByEmail(ctx, "nathan.frund@gmail.com")
	if err != nil {
		log.Fatalf("Error finding user: %v", err)
	}

	if user != nil {
		fmt.Printf("Found user: %+v\n", user)
	} else {
		fmt.Println("No user found with that email")
	}
}
