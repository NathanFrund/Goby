package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/database"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// Setup
	ctx := context.Background()
	cfg := testutils.ConfigForTests(t)

	// Use the new connection manager
	conn := database.NewConnection(cfg)
	err := conn.Connect(ctx)
	require.NoError(t, err)
	conn.StartMonitoring()

	cleanup := func() {
		conn.Close(context.Background())
	}
	defer cleanup()

	// Create a client and user store
	userDBClient, err := database.NewClient[domain.User](conn)
	require.NoError(t, err)
	userStore := database.NewUserStore(userDBClient, conn)

	authMiddleware := Auth(userStore)

	// Create Echo instance for testing
	e := echo.New()
	// We don't need a full renderer for this test.
	// e.Renderer = templates.NewRenderer("../../web/src/templates")

	// A simple test handler that runs after the middleware.
	// It checks if the user was correctly placed in the context.
	testDashboardHandler := func(c echo.Context) error {
		user := c.Get(UserContextKey).(*domain.User)
		return c.String(http.StatusOK, "Welcome "+user.Email)
	}
	e.GET("/app/dashboard", testDashboardHandler, authMiddleware)
	e.GET("/auth/login", func(c echo.Context) error {
		return c.String(http.StatusOK, "Login Page")
	}) // Dummy login page for redirect checks

	t.Run("unauthenticated user is redirected to login", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		// Assert that the user is redirected
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
	})

	t.Run("authenticated user can access protected route", func(t *testing.T) {
		// 1. Create a user and sign them in to get a valid token
		testEmail := "auth-middleware-test@example.com"
		testPassword := "password123"
		testName := "Auth Middleware User"

		_, err := userStore.SignUp(ctx, &domain.User{Email: testEmail, Name: &testName}, testPassword)
		require.NoError(t, err, "failed to sign up user for auth test")

		token, err := userStore.SignIn(ctx, &domain.User{Email: testEmail}, testPassword)
		require.NoError(t, err, "failed to sign in user for auth test")
		require.NotEmpty(t, token, "sign in should return a token")

		// Cleanup the user after the test
		t.Cleanup(func() {
			u, findErr := userStore.FindUserByEmail(ctx, testEmail)
			if findErr == nil && u != nil {
				_ = userStore.Delete(ctx, u.ID.String())
			}
		})

		// 2. Create a request with the auth cookie
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: token,
			Path:  "/",
		})

		// 3. Serve the request
		e.ServeHTTP(rec, req)

		// 4. Assert that the request was successful
		assert.Equal(t, http.StatusOK, rec.Code)
		assert.Contains(t, rec.Body.String(), "Welcome")
		assert.Contains(t, rec.Body.String(), testEmail)
	})

	t.Run("user with invalid token is redirected", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
		rec := httptest.NewRecorder()
		req.AddCookie(&http.Cookie{
			Name:  "auth_token",
			Value: "this-is-an-invalid-token",
			Path:  "/",
		})

		e.ServeHTTP(rec, req)

		// Assert that the user is redirected
		assert.Equal(t, http.StatusSeeOther, rec.Code)
		assert.Equal(t, "/auth/login", rec.Header().Get("Location"))
	})
}
