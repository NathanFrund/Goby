package auth_errors

import "errors"

// Standard application domain errors related specifically to user authentication,
// registration, and password management.
var (
	// ErrUserAlreadyExists indicates a sign-up attempt failed because the user's
	// email address is already present in the system.
	ErrUserAlreadyExists = errors.New("user already exists")

	// ErrInvalidCredentials indicates a sign-in attempt failed due to an incorrect
	// email or password combination.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrInvalidResetToken indicates that a password reset request used a token
	// that is either expired, already used, or was never valid.
	ErrInvalidResetToken = errors.New("invalid or expired password reset token")
)
