package domain

import (
	"context"
	"errors"

	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

// User represents the core user model in the application domain.
type User struct {
	ID                *surrealmodels.RecordID `json:"id,omitempty"`
	Email             string                  `json:"email"`
	Password          string                  `json:"password,omitempty"`
	Name              *string                 `json:"name,omitempty"`
	ResetToken        *string                 `json:"resetToken,omitempty"`
	ResetTokenExpires *string                 `json:"resetTokenExpires,omitempty"`
}

// ErrUserAlreadyExists is returned when trying to create a user that already exists.
var ErrUserAlreadyExists = errors.New("user with this email already exists")

// UserRepository defines the contract for user data storage operations.
// It lives in the domain because it's a requirement OF the domain, not
// of the database implementation.
type UserRepository interface {
	SignUp(ctx context.Context, user *User, password string) (string, error)
	SignIn(ctx context.Context, user *User, password string) (string, error)
	Authenticate(ctx context.Context, token string) (*User, error)
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	GenerateResetToken(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, newPassword string) (*User, error)
	WithTransaction(ctx context.Context, fn func(repo UserRepository) error) error
}
