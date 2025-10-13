package testutils

import "github.com/nfrund/goby/internal/domain"

// TestUser is a helper struct for creating users in integration tests.
// It includes the password field which is required for the SIGNUP process
// in SurrealDB but is not part of the domain.User model that is returned.
type TestUser struct {
	domain.User
	Password string `json:"password"`
}
