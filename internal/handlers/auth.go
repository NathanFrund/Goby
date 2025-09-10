package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/models"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	userStore *database.UserStore
	emailer   email.EmailSender
	baseURL   string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userStore *database.UserStore, emailer email.EmailSender, baseURL string) *AuthHandler {
	return &AuthHandler{
		userStore: userStore,
		emailer:   emailer,
		baseURL:   baseURL,
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
	token, err := h.userStore.SignUp(c.Request().Context(), newUser, password)
	if err != nil {
		// The SignUp method will fail if the user already exists. The underlying error
		// from SurrealDB often contains "signup query failed" in this case.
		if strings.Contains(err.Error(), "signup query failed") || strings.Contains(err.Error(), "already exists") {
			return c.Render(http.StatusConflict, "register.html", map[string]interface{}{
				"Error": "A user with this email already exists.",
				"Email": email,
			})
		}
		slog.Error("Error creating user", "error", err)
		return c.Render(http.StatusInternalServerError, "register.html", map[string]interface{}{
			"Error": "Could not create user account.",
			"Email": email,
		})
	}

	// --- Session Management ---
	setAuthCookie(c, token)

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
	token, err := h.userStore.SignIn(c.Request().Context(), user, password)
	if err != nil {
		// The SignIn method will fail if credentials are invalid.
		slog.Warn("Failed login attempt", "email", email, "error", err)
		return c.Render(http.StatusUnauthorized, "login.html", map[string]interface{}{
			"Error": "Invalid email or password.",
			"Email": email,
		})
	}

	// --- Session Management ---
	setAuthCookie(c, token)

	// On success, redirect to the home page.
	return c.Redirect(http.StatusSeeOther, "/")
}

// Logout handles logging the user out by clearing their session cookie.
func (h *AuthHandler) Logout(c echo.Context) error {
	// To log a user out, we expire the authentication cookie immediately.
	// Setting MaxAge to -1 is the standard way to delete a cookie.
	setAuthCookie(c, "") // Set an empty token

	return c.Redirect(http.StatusSeeOther, "/login")
}

// ForgotPasswordGet handles rendering the forgot password page.
func (h *AuthHandler) ForgotPasswordGet(c echo.Context) error {
	return c.Render(http.StatusOK, "forgot-password.html", nil)
}

// ForgotPasswordPost handles the form submission for requesting a password reset.
func (h *AuthHandler) ForgotPasswordPost(c echo.Context) error {
	email := c.FormValue("email")

	token, err := h.userStore.GenerateResetToken(c.Request().Context(), email)
	if err != nil {
		// To prevent email enumeration attacks, we show a generic success message
		// even if the user was not found. The error is logged for debugging.
		slog.Info("Error generating reset token, hiding from user", "email", email, "error", err)
	}

	// In a real application, you would send an email with the reset link here.
	// For development, we'll log the token to the console.
	if token != "" && h.emailer != nil {
		resetLink := h.baseURL + "/reset-password?token=" + token
		htmlBody := fmt.Sprintf(`<p>Click the link below to reset your password:</p><a href="%s">Reset Password</a>`, resetLink)
		err = h.emailer.Send(email, "Reset Your Password", htmlBody)
		if err != nil {
			// Log the error but still show a success message to the user.
			slog.Error("Failed to send password reset email", "error", err, "email", email)
		}
	}

	return c.Render(http.StatusOK, "forgot-password.html", map[string]interface{}{
		"Success": "If an account with that email exists, a password reset link has been sent.",
	})
}

// ResetPasswordGet handles rendering the password reset page.
func (h *AuthHandler) ResetPasswordGet(c echo.Context) error {
	token := c.QueryParam("token")
	if token == "" {
		// If no token is provided, redirect to the forgot password page.
		return c.Redirect(http.StatusSeeOther, "/forgot-password")
	}

	return c.Render(http.StatusOK, "reset-password.html", map[string]interface{}{
		"Token": token,
	})
}

// ResetPasswordPost handles the form submission for setting a new password.
func (h *AuthHandler) ResetPasswordPost(c echo.Context) error {
	token := c.FormValue("token")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	if password != passwordConfirm {
		return c.Render(http.StatusBadRequest, "reset-password.html", map[string]interface{}{
			"Token": token,
			"Error": "Passwords do not match.",
		})
	}

	if len(password) < 8 {
		return c.Render(http.StatusBadRequest, "reset-password.html", map[string]interface{}{
			"Token": token,
			"Error": "Password must be at least 8 characters long.",
		})
	}

	user, err := h.userStore.ResetPassword(c.Request().Context(), token, password)
	if err != nil {
		slog.Warn("Error resetting password", "error", err)
		return c.Render(http.StatusUnauthorized, "reset-password.html", map[string]interface{}{"Error": "Invalid or expired reset link."})
	}

	// Automatically sign the user in after a successful password reset.
	sessionToken, err := h.userStore.SignIn(c.Request().Context(), user, password)
	if err != nil {
		// This is unlikely, but we should handle it. If sign-in fails,
		// redirect to the login page with a success message as a fallback.
		slog.Error("Failed to sign in user after password reset", "error", err, "user_id", user.ID)
		return c.Redirect(http.StatusSeeOther, "/login?reset=success")
	}

	setAuthCookie(c, sessionToken)

	return c.Redirect(http.StatusSeeOther, "/")
}

// setAuthCookie is a helper function to create and set the authentication cookie.
func setAuthCookie(c echo.Context, token string) {
	cookie := new(http.Cookie)
	cookie.Name = "auth_token"
	cookie.Value = token
	cookie.Path = "/"
	// The cookie will expire in 24 hours.
	if token == "" {
		// If the token is empty, we're logging out, so expire the cookie immediately.
		cookie.MaxAge = -1
	} else {
		cookie.Expires = time.Now().UTC().Add(24 * time.Hour)
	}
	// HttpOnly flag prevents client-side JavaScript from accessing the cookie,
	// which is a crucial security measure against XSS attacks.
	cookie.HttpOnly = true
	// Secure flag ensures the cookie is only sent over HTTPS connections.
	// The check `c.Request().TLS != nil` makes this work in production (with HTTPS)
	// and local development (without HTTPS).
	cookie.Secure = c.Request().TLS != nil
	// SameSite=Lax provides a good balance of security and usability for CSRF protection.
	cookie.SameSite = http.SameSiteLaxMode
	c.SetCookie(cookie)
}
