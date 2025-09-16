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
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

const testSessionSecret = "a-very-secret-key-for-testing-!"

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
	return "reset-token", nil
}

func (m *MockUserStore) ResetPassword(ctx context.Context, token, password string) (*domain.User, error) {
	// Create a valid RecordID for the mock user.
	// In a real scenario, this would come from the database.
	parts := strings.Split("user:1", ":")
	table, id := parts[0], parts[1]
	recordID := surrealmodels.NewRecordID(table, id)

	return &domain.User{ID: &recordID, Email: "test@example.com"}, nil
}

func (m *MockUserStore) Authenticate(ctx context.Context, token string) (*domain.User, error) {
	// In a real mock, you might check the token and return different users.
	// For this test, a simple successful authentication is sufficient.
	return &domain.User{Email: "test@example.com"}, nil
}

func (m *MockUserStore) FindUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	// This mock can assume the user is found for handler tests.
	// Error cases can be tested at the store level.
	return &domain.User{Email: email}, nil
}

// mockConfigProvider is a simple mock for the config.Provider interface.
type mockConfigProvider struct {
	config.Provider
	baseURL string
}

func (m *mockConfigProvider) GetAppBaseURL() string { return m.baseURL }

func setupAuthTest(store domain.UserRepository) (*echo.Echo, *handlers.AuthHandler) {
	e := echo.New()
	// Use a mock config provider for tests, though it's not strictly needed here.
	mockCfg := &mockConfigProvider{baseURL: "http://localhost:8080"}
	// For unit tests, it's better to create the mock emailer directly
	// instead of relying on the factory and a real config struct.
	mockEmailer := &email.LogSender{}
	// The handler now correctly depends only on interfaces and primitives.
	authHandler := handlers.NewAuthHandler(store, mockEmailer, mockCfg.GetAppBaseURL())

	// Setup session middleware
	cookieStore := sessions.NewCookieStore([]byte(testSessionSecret))
	e.Use(session.Middleware(cookieStore))

	return e, authHandler
}

// assertFlashMessage is a test helper to check for a specific flash message in the session.
func assertFlashMessage(t *testing.T, req *http.Request, key, expectedMessage string) {
	t.Helper() // Marks this function as a test helper.

	// To check the session, we need to read the cookie set in the response.
	// We can then use the session store to decode it.
	cookieStore := sessions.NewCookieStore([]byte(testSessionSecret))
	sess, _ := cookieStore.Get(req, "flash-session")

	flashes := sess.Flashes(key)
	assert.NotEmpty(t, flashes, "expected flash message but found none for key: %s", key)
	assert.Equal(t, expectedMessage, flashes[0])
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

// newUnitTextContext creates a minimal echo.Context for unit testing handlers.
// It includes an initialized session, which is required by handlers that use flash messages.
func newUnitTextContext(req *http.Request) (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Manually apply session middleware to the context for this unit test.
	// This ensures `session.Get` will work inside the handler.
	_ = session.Middleware(sessions.NewCookieStore([]byte(testSessionSecret)))(func(c echo.Context) error { return nil })(c)
	return c, rec
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
	c, rec := newUnitTextContext(req)

	// 3. Act: Call the handler method directly.
	err := authHandler.ForgotPasswordPost(c)

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
	e, authHandler := setupAuthTest(&MockUserStore{})
	e.POST("/auth/register", authHandler.RegisterPost)

	t.Run("sets success flash on successful registration", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "password123")

		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, req, "success", "Account created successfully!")
	})

	t.Run("sets error flash on password mismatch", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test2@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "wrongpassword")

		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, req, "error", "Passwords do not match.")
	})
}

func TestLoginPost_RepopulatesEmailOnError(t *testing.T) {
	// Create a mock store configured to return an error on SignIn
	mockStore := &MockUserStore{SignInShouldError: true}
	// Use the setup helper, passing in our configured mock store.
	e, authHandler := setupAuthTest(mockStore)
	e.POST("/auth/login", authHandler.LoginPost)

	// --- Test ---
	form := url.Values{}
	submittedEmail := "test@example.com"
	form.Set("email", submittedEmail)
	form.Set("password", "wrongpassword")

	req := httptest.NewRequest(http.MethodPost, "/auth/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- Assertions ---
	// Assert it redirects back to the login page
	assert.Equal(t, http.StatusSeeOther, rec.Code)
	assert.Equal(t, "/auth/login", rec.Header().Get("Location"))

	// Assert that the error flash message is set
	assertFlashMessage(t, req, "error", "Invalid email or password.")

	// Assert that the submitted email was also flashed to the session
	assertFlashMessage(t, req, "form_email", submittedEmail)
}
