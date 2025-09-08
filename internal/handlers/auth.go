package handlers

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/models"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	userStore *database.UserStore
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userStore *database.UserStore) *AuthHandler {
	return &AuthHandler{
		userStore: userStore,
	}
}

// RegisterGet handles the request to show the registration page.
func (h *AuthHandler) RegisterGet(c echo.Context) error {
	// This handler's only job is to render the registration page template.
	// The template name "pages/register" corresponds to the file
	// "web/src/templates/pages/register.html".
	return c.Render(http.StatusOK, "register.html", nil)
}

// RegisterPost handles the form submission for creating a new user.
func (h *AuthHandler) RegisterPost(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	// --- Validation ---
	if password != passwordConfirm {
		return c.Render(http.StatusBadRequest, "register.html", map[string]interface{}{
			"Error": "Passwords do not match.",
			"Email": email,
		})
	}

	if len(password) < 8 {
		return c.Render(http.StatusBadRequest, "register.html", map[string]interface{}{
			"Error": "Password must be at least 8 characters long.",
			"Email": email,
		})
	}

	// --- Database Interaction ---
	// Use the UserStore to create the user. This method handles hashing and
	// checking for duplicates, aligning with the successful test cases.
	// The user's name is not collected on the form, so we pass nil.
	newUser := &models.User{
		Email: email,
		Name:  nil,
	}

	// Use the SignUp method, which is the correct high-level function for registration.
	token, err := h.userStore.SignUp(context.Background(), newUser, password)
	if err != nil {
		// The SignUp method will fail if the user already exists. The underlying error
		// from SurrealDB often contains "signup query failed" in this case.
		if strings.Contains(err.Error(), "signup query failed") || strings.Contains(err.Error(), "already exists") {
			return c.Render(http.StatusConflict, "register.html", map[string]interface{}{
				"Error": "A user with this email already exists.",
				"Email": email,
			})
		}
		log.Printf("Error creating user: %v", err)
		return c.Render(http.StatusInternalServerError, "register.html", map[string]interface{}{
			"Error": "Could not create user account.",
			"Email": email,
		})
	}

	// --- Session Management ---
	// On successful registration, create a session cookie to log the user in.
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = token
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true                 // Prevents client-side JavaScript from accessing the cookie.
	cookie.Secure = c.Request().TLS != nil // Should be true in production (when using HTTPS).
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)

	// On success, redirect to the home page as a logged-in user.
	return c.Redirect(http.StatusSeeOther, "/")
}

// LoginGet handles the request to show the login page.
func (h *AuthHandler) LoginGet(c echo.Context) error {
	// This handler's only job is to render the login page template.
	return c.Render(http.StatusOK, "login.html", nil)
}

// LoginPost handles the form submission for logging in a user.
func (h *AuthHandler) LoginPost(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	// --- Database Interaction ---
	// The user model is only used to pass the email to the SignIn method.
	user := &models.User{Email: email}
	token, err := h.userStore.SignIn(context.Background(), user, password)
	if err != nil {
		// The SignIn method will fail if credentials are invalid.
		log.Printf("Failed login attempt for %s: %v", email, err)
		return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
			"Error": "Invalid email or password.",
		})
	}

	// --- Session Management ---
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = token
	cookie.Path = "/"
	cookie.Expires = time.Now().Add(24 * time.Hour)
	cookie.HttpOnly = true
	cookie.Secure = c.Request().TLS != nil
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)

	// On success, redirect to the home page.
	return c.Redirect(http.StatusSeeOther, "/")
}
