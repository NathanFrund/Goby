package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/database"
)

func main() {
	ctx := context.Background()
	cfg := config.New()

	db, err := database.NewDB(ctx, cfg)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close(ctx)

	userStore := database.NewUserStore(db)

	// Example usage in main()
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
