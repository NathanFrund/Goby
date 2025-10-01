package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/view"
	"github.com/nfrund/goby/internal/view/dto/auth"
	"github.com/nfrund/goby/web/src/templates/layouts"
	"github.com/nfrund/goby/web/src/templates/pages"
)

// AuthHandler handles authentication-related requests.
type AuthHandler struct {
	userStore domain.UserRepository
	emailer   domain.EmailSender
	baseURL   string
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(userStore domain.UserRepository, emailer domain.EmailSender, baseURL string) *AuthHandler {
	return &AuthHandler{
		userStore: userStore,
		emailer:   emailer,
		baseURL:   baseURL,
	}
}

// RegisterGetHandler renders the registration page (GET /auth/register).
// It retrieves flash messages and any pre-filled data (e.g., from a failed POST).
func (h *AuthHandler) RegisterGetHandler(c echo.Context) error {
	// 1. Retrieve the specific flash value for pre-filling the email.
	var prefilledEmail string
	if sess, err := session.Get("flash-session", c); err == nil {
		if flashes := sess.Flashes("form_email"); len(flashes) > 0 {
			if val, ok := flashes[0].(string); ok {
				prefilledEmail = val
			}
		}
		// CRITICAL: We must save the session here to clear the consumed "form_email" flash.
		_ = sess.Save(c.Request(), c.Response())
	}

	// 2. Retrieve General Flash Messages (Success/Error)
	flashes := view.GetFlashData(c)

	// 3. Prepare the View Model (DTO)
	// We use the retrieved email to populate the auth.RegisterData DTO.
	data := auth.RegisterData{
		Email: prefilledEmail,
	}

	// 4. Render the specific page content (pages.Register) with the DTO.
	pageContent := pages.Register(data)

	// 5. Wrap the content in the Base layout, passing the flashes.
	finalComponent := layouts.Base("Register", flashes, pageContent)

	// 6. Render the final HTML response.
	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
}

// RegisterPost handles the form submission for creating a new user.
func (h *AuthHandler) RegisterPost(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	// --- Validation ---
	if password != passwordConfirm {
		view.SetFlashError(c, "Passwords do not match.")
		return c.Redirect(http.StatusSeeOther, "/auth/register")
	}

	if len(password) < 8 {
		view.SetFlashError(c, "Password must be at least 8 characters long.")
		return c.Redirect(http.StatusSeeOther, "/auth/register")
	}

	// --- Database Interaction ---
	// Use the UserStore to create the user. This method handles hashing and
	// checking for duplicates, aligning with the successful test cases.
	// The user's name is not collected on the form, so we pass nil.
	newUser := &domain.User{
		Email: email,
		Name:  nil,
	}

	// Use the SignUp method, which is the correct high-level function for registration.
	token, err := h.userStore.SignUp(c.Request().Context(), newUser, password)
	if err != nil {
		// Check for the specific domain error for a duplicate user.
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			view.SetFlashError(c, "A user with this email already exists.")
		} else {
			slog.Error("Error creating user", "error", err)
			view.SetFlashError(c, "Could not create your account.")
		}
		return c.Redirect(http.StatusSeeOther, "/auth/register")
	}

	// --- Session Management ---
	setAuthCookie(c, token)

	// On success, redirect to the home page as a logged-in user.
	view.SetFlashSuccess(c, "Account created successfully!")
	return c.Redirect(http.StatusSeeOther, "/")
}

// LoginGetHandler renders the login page (GET /auth/login).
// It retrieves flash messages and any pre-filled data (e.g., from a failed POST).
func (h *AuthHandler) LoginGetHandler(c echo.Context) error {
	// 1. Retrieve the specific flash value for pre-filling the email (as done in the old function).
	var prefilledEmail string
	// Use "flash-session" as per your existing code. session.Get returns the session and no error.
	if sess, err := session.Get("flash-session", c); err == nil {
		if flashes := sess.Flashes("form_email"); len(flashes) > 0 {
			// Ensure we can assert the type safely.
			if val, ok := flashes[0].(string); ok {
				prefilledEmail = val
			}
		}
		// CRITICAL: We must save the session here to clear the consumed "form_email" flash.
		_ = sess.Save(c.Request(), c.Response())
	}

	// 2. Retrieve General Flash Messages (Success/Error)
	// This utility function also retrieves flashes and saves the session internally.
	flashes := view.GetFlashData(c)

	// 3. Prepare the View Model (DTO)
	// We use the retrieved email to populate the DTO.
	data := auth.LoginData{
		Email: prefilledEmail,
	}

	// 4. Render the specific page content (pages.Login) with the DTO.
	pageContent := pages.Login(data)

	// 5. Wrap the content in the Base layout, passing the flashes.
	finalComponent := layouts.Base("Login", flashes, pageContent)

	// 6. Render the final HTML response.
	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
}

// LoginPost handles the form submission for logging in a user.
func (h *AuthHandler) LoginPost(c echo.Context) error {
	email := c.FormValue("email")
	password := c.FormValue("password")

	// --- Database Interaction ---
	// The user model is only used to pass the email to the SignIn method.
	user := &domain.User{Email: email}
	token, err := h.userStore.SignIn(c.Request().Context(), user, password)
	if err != nil {
		// The SignIn method will fail if credentials are invalid.
		slog.Warn("Failed login attempt", "email", email, "error", err)
		view.SetFlashError(c, "Invalid email or password.")

		// Preserve the submitted email address for the next render of the login form.
		sess, _ := session.Get("flash-session", c)
		sess.AddFlash(email, "form_email")
		if err := sess.Save(c.Request(), c.Response()); err != nil {
			slog.Error("Failed to save session", "error", err)
		}

		return c.Redirect(http.StatusSeeOther, "/auth/login")
	}

	// --- Session Management ---
	setAuthCookie(c, token)

	// On success, redirect to the home page.
	view.SetFlashSuccess(c, "Logged in successfully!")
	return c.Redirect(http.StatusSeeOther, "/")
}

// Logout handles logging the user out by clearing their session cookie.
func (h *AuthHandler) Logout(c echo.Context) error {
	// To log a user out, we expire the authentication cookie immediately.
	// Setting MaxAge to -1 is the standard way to delete a cookie.
	setAuthCookie(c, "") // Set an empty token

	view.SetFlashSuccess(c, "You have been logged out.")
	return c.Redirect(http.StatusSeeOther, "/auth/login")
}

// ForgotPasswordGet handles rendering the forgot password page.
func (h *AuthHandler) ForgotPasswordGetHandler(c echo.Context) error {
	// 1. Retrieve flash data (errors/success messages) from the session.
	flashData := view.GetFlashData(c)

	// 2. Prepare the view data transfer object (DTO).
	// For a GET request, the email is usually empty unless retrieved from a flash session.
	authData := auth.ForgotPasswordData{}

	// 3. Define the core page content component.
	pageContent := pages.ForgotPassword(authData)

	// 4. Wrap the page content in the Base layout, passing the title and flash messages.
	finalComponent := layouts.Base("Forgot Password", flashData, pageContent)

	// 5. Render the final component directly using Templ's Render method.
	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
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
		resetLink := h.baseURL + "/auth/reset-password?token=" + token
		htmlBody := fmt.Sprintf(`<p>Click the link below to reset your password:</p><a href="%s">Reset Password</a>`, resetLink)
		err = h.emailer.Send(email, "Reset Your Password", htmlBody)
		if err != nil {
			// Log the error but still show a success message to the user.
			slog.Error("Failed to send password reset email", "error", err, "email", email)
		}
	}

	view.SetFlashSuccess(c, "If an account with that email exists, a password reset link has been sent.")
	return c.Redirect(http.StatusSeeOther, "/auth/forgot-password")
}

// ResetPasswordGetHandler handles rendering the password reset page (GET /auth/reset-password?token=...).
func (h *AuthHandler) ResetPasswordGetHandler(c echo.Context) error {
	// 1. Get the token from the query parameter.
	token := c.QueryParam("token")
	if token == "" {
		// If no token is provided, redirect to the forgot password page.
		// Set a flash error to inform the user why they were redirected.
		view.SetFlashError(c, "A valid reset token is required to change your password.")
		return c.Redirect(http.StatusSeeOther, "/auth/forgot-password")
	}

	// 2. Retrieve General Flash Messages (Success/Error)
	// Note: We don't need to check for form_email flashes here, as this is never a POST redirect target.
	flashes := view.GetFlashData(c)

	// 3. Prepare the View Model (DTO)
	// The DTO passes the token back to the template to be included in the hidden form field.
	data := auth.ResetPasswordData{
		Token: token,
	}

	// 4. Render the specific page content (pages.ResetPassword) with the DTO.
	pageContent := pages.ResetPassword(data)

	// 5. Wrap the content in the Base layout, passing the flashes.
	finalComponent := layouts.Base("Reset Password", flashes, pageContent)

	// 6. Render the final HTML response.
	c.Response().Header().Set(echo.HeaderContentType, "text/html; charset=utf-8")
	c.Response().WriteHeader(http.StatusOK)
	return finalComponent.Render(c.Request().Context(), c.Response().Writer)
}

// ResetPasswordPostHandler handles the form submission for setting a new password.
func (h *AuthHandler) ResetPasswordPostHandler(c echo.Context) error {
	token := c.FormValue("token")
	password := c.FormValue("password")
	passwordConfirm := c.FormValue("password_confirm")

	// --- Input Validation ---
	if password != passwordConfirm {
		view.SetFlashError(c, "Passwords do not match.")
		return c.Redirect(http.StatusSeeOther, "/auth/reset-password?token="+token)
	}

	if len(password) < 8 {
		view.SetFlashError(c, "Password must be at least 8 characters long.")
		return c.Redirect(http.StatusSeeOther, "/auth/reset-password?token="+token)
	}
	// --- End Input Validation ---

	// STEP 1: Perform the password reset (atomic data change).
	// This single method handles token validation, password update, and token invalidation.
	user, err := h.userStore.ResetPassword(c.Request().Context(), token, password)

	// Handle data change failure (e.g., invalid token, database error)
	if err != nil {
		slog.Warn("Password reset failed", "error", err)
		// Return a user-friendly error from the repository.
		view.SetFlashError(c, err.Error())
		return c.Redirect(http.StatusSeeOther, "/auth/reset-password?token="+token)
	}

	// STEP 2: After successful data update, perform the sign-in (auth change).
	// We use the updated 'user' object returned from ResetPassword.
	var sessionToken string
	// NOTE: We pass the raw password here as the user has just entered it.
	sessionToken, err = h.userStore.SignIn(c.Request().Context(), user, password)

	// Handle sign-in failure (unlikely if Step 1 succeeded, but good practice)
	if err != nil {
		slog.Error("Failed to sign in user after successful password reset", "error", err)
		view.SetFlashError(c, "Password reset successful, but failed to log you in automatically. Please log in manually.")
		return c.Redirect(http.StatusSeeOther, "/auth/login")
	}

	// STEP 3: Success
	setAuthCookie(c, sessionToken)
	view.SetFlashSuccess(c, "Your password has been reset successfully! You are now logged in.")
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
