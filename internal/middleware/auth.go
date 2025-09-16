package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
)

const UserContextKey = "user"

// Auth creates a middleware that protects routes that require authentication.
func Auth(store database.UserStore) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// 1. Get the token from the cookie.
			cookie, err := c.Cookie("auth_token")
			if err != nil || cookie.Value == "" {
				// If the cookie is not set or is empty, redirect to login.
				return c.Redirect(http.StatusSeeOther, "/auth/login")
			}
			token := cookie.Value

			// 2. Validate the token and get the user.
			// We use the context from the request to pass it down to the store.
			user, err := store.Authenticate(c.Request().Context(), token)
			if err != nil {
				// If there's an error (e.g., token invalid), redirect to login.
				// It's good practice to also clear the invalid cookie.
				c.SetCookie(&http.Cookie{
					Name:   "auth_token",
					Value:  "",
					Path:   "/",
					MaxAge: -1,
				})
				return c.Redirect(http.StatusSeeOther, "/auth/login")
			}

			if user == nil {
				// This case should ideally not be hit if Authenticate returns an error
				// for an invalid token, but as a safeguard:
				return c.Redirect(http.StatusSeeOther, "/auth/login")
			}

			// 3. Store user information in the context for downstream handlers.
			c.Set(UserContextKey, user)

			// 4. User is authenticated, proceed to the next handler.
			return next(c)
		}
	}
}
