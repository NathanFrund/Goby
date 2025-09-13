package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/config"
	"github.com/nfrund/goby/internal/email"
	"github.com/nfrund/goby/internal/handlers"
	"github.com/nfrund/goby/internal/models"
	"github.com/stretchr/testify/assert"
	surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"
)

const testSessionSecret = "a-very-secret-key-for-testing-!"

// MockUserStore provides a mock implementation of the UserStore for testing.
type MockUserStore struct{}

func (m *MockUserStore) SignUp(ctx context.Context, user *models.User, password string) (string, error) {
	return "test-token", nil
}

func (m *MockUserStore) SignIn(ctx context.Context, user *models.User, password string) (string, error) {
	return "test-token", nil
}

func (m *MockUserStore) GenerateResetToken(ctx context.Context, email string) (string, error) {
	return "reset-token", nil
}

func (m *MockUserStore) ResetPassword(ctx context.Context, token, password string) (*models.User, error) {
	// Create a valid RecordID for the mock user.
	// In a real scenario, this would come from the database.
	parts := strings.Split("user:1", ":")
	table, id := parts[0], parts[1]
	recordID := surrealmodels.NewRecordID(table, id)

	return &models.User{ID: &recordID, Email: "test@example.com"}, nil
}

func setupAuthTest() (*echo.Echo, *handlers.AuthHandler) {
	e := echo.New()
	cfg := &config.Config{AppBaseURL: "http://localhost:8080"}
	mockStore := &MockUserStore{}
	mockEmailer, _ := email.NewEmailService(&config.Config{EmailProvider: "log"})
	authHandler := handlers.NewAuthHandler(mockStore, mockEmailer, cfg.AppBaseURL)

	// Setup session middleware
	cookieStore := sessions.NewCookieStore([]byte(testSessionSecret))
	e.Use(session.Middleware(cookieStore))

	return e, authHandler
}

// assertFlashMessage is a test helper to check for a specific flash message in the session.
func assertFlashMessage(t *testing.T, rec *httptest.ResponseRecorder, key, expectedMessage string) {
	t.Helper() // Marks this function as a test helper.

	// To check the session, we need to read the cookie set in the response.
	// We can then use the session store to decode it.
	cookieStore := sessions.NewCookieStore([]byte(testSessionSecret))
	sess, _ := cookieStore.Get(rec.Result().Request, "flash-session")

	flashes := sess.Flashes(key)
	assert.NotEmpty(t, flashes, "expected flash message but found none for key: %s", key)
	assert.Equal(t, expectedMessage, flashes[0])
}

func TestRegisterPost_FlashMessages(t *testing.T) {
	e, authHandler := setupAuthTest()

	t.Run("sets success flash on successful registration", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "password123")

		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := authHandler.RegisterPost(c)
		assert.NoError(t, err)

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, rec, "success", "Account created successfully!")
	})

	t.Run("sets error flash on password mismatch", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", "test2@example.com")
		form.Set("password", "password123")
		form.Set("password_confirm", "wrongpassword")

		req := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(form.Encode()))
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		err := authHandler.RegisterPost(c)
		assert.NoError(t, err)

		// Check for redirect
		assert.Equal(t, http.StatusSeeOther, rec.Code)

		// Check session for flash message
		assertFlashMessage(t, rec, "error", "Passwords do not match.")
	})
}
