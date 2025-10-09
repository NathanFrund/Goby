package domain

import "errors"

// Sentinel errors for the domain layer. These provide consistent, checkable
// errors for common business logic failures.
var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials provided")
	ErrNotFound           = errors.New("requested resource not found")
)
