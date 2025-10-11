package handlers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

const testSessionSecret = "a-very-secret-key-for-testing-!"

// Create a single session store to be used by all tests in this file.
var testCookieStore = sessions.NewCookieStore([]byte(testSessionSecret))

// MockUserStore provides a mock implementation of the UserStore for testing.
type MockUserStore struct {
	SignInShouldError bool
	SignUpShouldError bool
}

func (m *MockUserStore) SignUp(ctx context.Context, user *domain.User, password string) (string, error) {
	if m.SignUpShouldError {
		return "", fmt.Errorf("mock sign up error")
	}
	return "test-token", nil
}

func (m *MockUserStore) SignIn(ctx context.Context, user *domain.User, password string) (string, error) {
	if m.SignInShouldError {
		return "", fmt.Errorf("mock sign in error")
	}
	return "test-token", nil
}

func (m *MockUserStore) GenerateResetToken(ctx context.Context, email string) (string, error) {
	return "test-reset-token", nil
}

func (m *MockUserStore) ResetPassword(ctx context.Context, token, password string) (*domain.User, error) {
	// Create a valid RecordID for the mock user.
	// In a real scenario, this would come from the database.
	recordID := surrealmodels.NewRecordID("user", "1")

	return &domain.User{ID: &recordID, Email: "test@example.com"}, nil
}

func (m *MockUserStore) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	// In a real mock, you might check the token and return different users.
	// For this test, a simple successful authentication is sufficient.
	recordID := surrealmodels.NewRecordID("user", "1")
	return &domain.User{ID: &recordID, Email: "test@example.com"}, nil
}

func (m *MockUserStore) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	// This mock can assume the user is found for handler tests.
	// Error cases can be tested at the store level.
	recordID := surrealmodels.NewRecordID("user", "1")
	return &domain.User{ID: &recordID, Email: email}, nil
}

func (m *MockUserStore) WithTransaction(ctx context.Context, fn func(repo domain.UserRepository) error) error {
	// For the mock, we just execute the function directly, passing the mock itself.
	return fn(m)
}

func (m *MockUserStore) Delete(ctx context.Context, id string) error {
	// This is a mock implementation and can be empty for these tests.
	return nil
}

// setupAuthTest creates an AuthHandler for testing.
func setupAuthTest(store domain.UserRepository) *handlers.AuthHandler {
	// For unit tests, it's better to create the mock emailer directly.
	mockEmailer := &email.LogSender{}
	authHandler := handlers.NewAuthHandler(store, mockEmailer, "http://test.local")
	return authHandler
}

// newTestContext creates a minimal echo.Context for unit testing handlers.
// It includes an initialized session, which is required by handlers that use flash messages.
func newTestContext(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Manually apply session middleware to the context for this unit test.
	// This ensures `session.Get` will work inside the handler.
	_ = session.Middleware(testCookieStore)(func(c echo.Context) error { return nil })(c)
	return c, rec
}

// assertFlashMessage is a test helper to check for a specific flash message in the session.
func assertFlashMessage(t *testing.T, c echo.Context, key, expectedMessage string) {
	t.Helper() // Marks this function as a test helper.
	sess, err := session.Get("flash-session", c)
	require.NoError(t, err, "Failed to get session from context")
	flashes := sess.Flashes(key)
	assert.NotEmpty(t, flashes, "expected flash message but found none for key: %s", key)
	assert.Equal(t, expectedMessage, flashes[0].(string))
}

// --- Pure Unit Tests ---
// These tests focus on the handler's logic in isolation, using mocks
// to simulate dependencies. They do not require a database or network connection.

// unitTestEmailSender is a mock that implements domain.EmailSender for unit tests.
type unitTestEmailSender struct {
	SendCalled bool
	LastTo     string
	LastSub    string
	LastBody   string
}

func (m *unitTestEmailSender) Send(to, subject, htmlBody string) error {
	m.SendCalled = true
	m.LastTo = to
	m.LastSub = subject
	m.LastBody = htmlBody
	return nil
}

// unitTestUserRepo is a mock that implements domain.UserRepository for unit tests.
type unitTestUserRepo struct {
	// Make all interface methods no-ops by default
	domain.UserRepository
	GenerateResetTokenFunc func(ctx context.Context, email string) (string, error)
}

func (m *unitTestUserRepo) GenerateResetToken(ctx context.Context, email string) (string, error) {
	if m.GenerateResetTokenFunc != nil {
		return m.GenerateResetTokenFunc(ctx, email)
	}
	return "unit-test-token", nil
}

func TestAuthHandler_ForgotPasswordPost_Unit(t *testing.T) {
	// 1. Setup: Create mocks and instantiate the handler directly.
	mockRepo := &unitTestUserRepo{}
	mockEmailer := &unitTestEmailSender{}
	authHandler := handlers.NewAuthHandler(mockRepo, mockEmailer, "http://test.local")

	// 2. Create a minimal Echo context, no server needed.
	form := url.Values{}
	form.Set("email", "user@example.com")
	req := httptest.NewRequest(http.MethodPost, "/auth/forgot-password", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	c, rec := newTestContext(req) // Use the test helper for consistency

	// 3. Act: Call the handler method directly, wrapped in the session middleware.
	err := session.Middleware(testCookieStore)(authHandler.ForgotPasswordPost)(c)

	// 4. Assert: Verify the behavior.
	require.NoError(t, err)
	assert.Equal(t, http.StatusSeeOther, rec.Code, "should redirect on success")

	// Assert that the handler interacted with its dependencies correctly.
	assert.True(t, mockEmailer.SendCalled, "expected EmailSender.Send to be called")
	assert.Equal(t, "user@example.com", mockEmailer.LastTo, "email should be sent to the correct user")
	assert.Contains(t, mockEmailer.LastBody, "http://test.local/auth/reset-password?token=unit-test-token", "email body should contain the correct reset link")
}

func TestRegisterPost_FlashMessages(t *testing.T) {
	// Use the setup helper with a standard, non-erroring mock store.
	authHandler := setupAuthTest(&MockUserStore{})

	t.Run("sets success flash on successful registration", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "password123")
		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		c, rec := newTestContext(req)

		// Wrap the handler call in the session middleware. The middleware will handle
		// the redirect error, so we expect a nil error from the middleware itself.
		err := session.Middleware(testCookieStore)(authHandler.RegisterPost)(c)
		require.NoError(t, err, "session middleware should handle the redirect and not return an error")

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, c, "flash_success", "Account created successfully!")
	})

	t.Run("sets error flash on password mismatch", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test2@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "wrongpassword")
		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		c, rec := newTestContext(req)

		err := session.Middleware(testCookieStore)(authHandler.RegisterPost)(c)
		require.NoError(t, err, "session middleware should handle the redirect and not return an error")

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, c, "flash_error", "Passwords do not match.")
	})
}

func TestLoginPost_RepopulatesEmailOnError(t *testing.T) {
	// Create a mock store configured to return an error on SignIn
	mockStore := &MockUserStore{SignInShouldError: true}
	authHandler := setupAuthTest(mockStore)

	// --- Test ---
	form := url.Values{}
	submittedEmail := "test@example.com"
	form.Set("email", submittedEmail)
	form.Set("password", "wrongpassword")
	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	c, rec := newTestContext(req)

	err := session.Middleware(testCookieStore)(authHandler.LoginPost)(c)
	require.NoError(t, err, "session middleware should handle the redirect and not return an error")

	// --- Assertions ---
	// Assert it redirects back to the login page
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))

	// Assert that the error flash message is set
	assertFlashMessage(t, c, "flash_error", "Invalid email or password.")

	// Assert that the submitted email was also flashed to the session
	assertFlashMessage(t, c, "form_email", submittedEmail)
}
