package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/surrealdb/surrealdb.go"
	"github.com/surrealdb/surrealdb.go/pkg/models"
)

type User struct {
	ID    *models.RecordID `json:"id,omitempty"`
	Name  string           `json:"name"`
	Email string           `json:"email"`
}

func FirstResult(results *[]surrealdb.QueryResult[[]User]) *User {
	if results == nil || len(*results) == 0 {
		return nil
	}
	if (*results)[0].Status != "OK" || len((*results)[0].Result) == 0 {
		return nil
	}
	return &((*results)[0].Result[0])
}

func main() {
	fmt.Println("Hello!")

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	dbURL := os.Getenv("SURREAL_URL")
	dbUser := os.Getenv("SURREAL_USER")
	dbPass := os.Getenv("SURREAL_PASS")
	dbNS := os.Getenv("SURREAL_NS")
	dbDB := os.Getenv("SURREAL_DB")

	if dbURL == "" || dbNS == "" || dbDB == "" {
		log.Fatal("Required environment variables SURREALDB_URL, SURREALDB_NS, or SURREALDB_DB are not set.")
	}

	db, err := surrealdb.FromEndpointURLString(context.Background(), dbURL)
	if err != nil {
		panic(err)
	}

	if err = db.Use(context.Background(), dbNS, dbDB); err != nil {
		panic(err)
	}

	authData := &surrealdb.Auth{
		Username: dbUser,
		Password: dbPass,
	}

	token, err := db.SignIn(context.Background(), authData)
	if err != nil {
		panic(err)
	}
	fmt.Println(token)

	users, err := surrealdb.Select[[]User](context.Background(), db, models.Table("user"))
	if err != nil {
		panic(err)
	}

	fmt.Printf("Selected all in persons table: %+v\n", users)

	fmt.Println("---")

	// Query a single user by email
	users, err = surrealdb.Select[[]User](context.Background(), db, models.Table("user"))
	if err != nil {
		panic(err)
	}

	// Filter the results by email
	var foundUser *User
	for i := 0; i < len(*users); i++ {
		if (*users)[i].Email == "nathan.frund@gmail.com" {
			foundUser = &(*users)[i]
			break
		}
	}

	if foundUser != nil {
		fmt.Printf("Found user: %+v\n", *foundUser)
	} else {
		fmt.Println("User not found")
	}

	// ----

	results, err := surrealdb.Query[[]User](
		context.Background(),
		db,
		"SELECT * FROM user WHERE email = $email",
		map[string]any{
			"email": "nathan.frund@gmail.com",
		},
	)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Query results: %+v\n", results)

	// Extract users from results

	var foundUsers []User
	if len(*results) > 0 && (*results)[0].Status == "OK" {
		// results[0].Result is []User
		foundUsers = (*results)[0].Result
	}

	foundUser = FirstResult(results)
	if foundUser != nil {
		fmt.Println("---")
		fmt.Printf("Found user: %+v\n", foundUsers[0])
		fmt.Printf("Found user ID: %+v\n", foundUsers[0].ID)
	} else {
		fmt.Println("User not found")
	}

}
