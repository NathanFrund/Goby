package database

import (
	"context"

	"github.com/nfrund/goby/internal/models"
)

// UserStore defines the interface for user data storage operations.
// This allows for dependency injection and easier testing of handlers.
type UserStore interface {
	SignUp(ctx context.Context, user *models.User, password string) (string, error)
	SignIn(ctx context.Context, user *models.User, password string) (string, error)
	Authenticate(ctx context.Context, token string) (*models.User, error)
	FindUserByEmail(ctx context.Context, email string) (*models.User, error)
	GenerateResetToken(ctx context.Context, email string) (string, error)
	ResetPassword(ctx context.Context, token, newPassword string) (*models.User, error)
}
