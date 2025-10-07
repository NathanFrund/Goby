package server_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthFlow_Integration(t *testing.T) {
	_, testServer, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Use a real HTTP client with a cookie jar to manage session/auth cookies automatically.
	// This makes the test more realistic, like a real browser.
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	client := &http.Client{
		Jar: jar,
		// Prevent the client from following redirects automatically, so we can inspect them.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// --- Test Data ---
	testEmail := fmt.Sprintf("auth-flow-user-%d@example.com", time.Now().UnixNano())
	testPassword := "a-secure-password-123"

	// 1. Register a new user
	t.Run("should register a new user", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", testEmail)
		form.Set("password", testPassword)
		form.Set("password_confirm", testPassword)

		res, err := client.Post(testServer.URL+"/auth/register", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusSeeOther, res.StatusCode, "Registration should redirect")
		assert.Equal(t, "/", res.Header.Get("Location"), "Should redirect to the home page")
	})

	// 2. Log in with the new user
	t.Run("should log in the new user", func(t *testing.T) {
		form := url.Values{}
		form.Set("email", testEmail)
		form.Set("password", testPassword)

		res, err := client.Post(testServer.URL+"/auth/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusSeeOther, res.StatusCode, "Login should redirect")
		assert.Equal(t, "/", res.Header.Get("Location"), "Should redirect to the home page after login")

		// The cookie jar now holds the auth token.
	})

	// 3. Access a protected route
	t.Run("should allow access to a protected route when logged in", func(t *testing.T) {
		res, err := client.Get(testServer.URL + "/app/chat")
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusOK, res.StatusCode, "Should be able to access /app/chat")
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Game State Monitor", "Chat page content should be present")
	})

	// 4. Log out
	t.Run("should log the user out", func(t *testing.T) {
		res, err := client.Get(testServer.URL + "/auth/logout")
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusSeeOther, res.StatusCode, "Logout should redirect")
		assert.Equal(t, "/auth/login", res.Header.Get("Location"), "Should redirect to the login page after logout")

		// The cookie jar should now have an expired/cleared auth cookie.
	})

	// 5. Fail to access protected route after logout
	t.Run("should deny access to protected route after logout", func(t *testing.T) {
		res, err := client.Get(testServer.URL + "/app/chat")
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusSeeOther, res.StatusCode, "Should redirect when not logged in")
		assert.Equal(t, "/auth/login", res.Header.Get("Location"), "Should redirect to login page")
	})

	// 6. Fail to access protected route with a fresh client (no cookies)
	t.Run("should deny access to protected route for unauthenticated client", func(t *testing.T) {
		// Create a new client with no cookies
		freshClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		res, err := freshClient.Get(testServer.URL + "/app/chat")
		require.NoError(t, err)
		defer res.Body.Close()

		assert.Equal(t, http.StatusSeeOther, res.StatusCode, "Should redirect when not logged in")
		assert.Equal(t, "/auth/login", res.Header.Get("Location"), "Should redirect to login page")
	})
}
