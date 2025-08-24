package main

import (
	"context"
	"fmt"
	"log"

	"github.com/nfrund/goby/internal/database"
)

func main() {

	ctx := context.Background()
	db, err := database.NewConnection(ctx)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close(ctx)

	// Example usage in main()
	user, err := database.FindUserByEmail(context.Background(), db, "nathan.frund@gmail.com")
	if err != nil {
		// Handle error (e.g., database connection issues)
		log.Fatalf("Error finding user: %v", err)
	}

	if user != nil {
		// User found
		fmt.Printf("Found user: %+v\n", user)
	} else {
		// User not found
		fmt.Println("No user found with that email")
	}

}
