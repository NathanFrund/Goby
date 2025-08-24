package models

import "github.com/surrealdb/surrealdb.go/pkg/models"

// User represents a user in the database.
type User struct {
	ID    *models.RecordID `json:"id,omitempty"`
	Name  string           `json:"name"`
	Email string           `json:"email"`
}
