package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/joho/godotenv"
	"github.com/nfrund/goby/internal/domain"
	"github.com/nfrund/goby/internal/modules/wargame" // Assuming this is used in the test
	"github.com/nfrund/goby/internal/server"
	"github.com/stretchr/testify/assert" // Note: Corrected a typo in the original file's import path
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Explicitly load and override environment variables from .env.test for this test run.
	// This ensures a consistent test environment, regardless of what's set in the host shell.
	// Using TestMain ensures this runs once for the entire package, before any tests are run.
	if err := godotenv.Overload("../../.env.test"); err != nil {
		log.Fatalf("Error loading .env.test file for integration tests: %v", err)
	}
	m.Run()
}

func TestTwoChannelArchitecture_Integration(t *testing.T) {
	// --- Test Setup ---

	// The server expects to be run from the project root, but the test runs from
	// internal/server. We temporarily change the working directory to the project
	// root so that file paths (like for templates) resolve correctly.
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir("../../")
	require.NoError(t, err)

	// We use defer to ensure we change back to the original directory after the
	// test completes, preventing side effects on other tests.
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Create a new server instance for testing
	s := server.New()
	s.RegisterRoutes()
	testServer := httptest.NewServer(s.E)
	defer testServer.Close()

	// Create a test user directly in the database
	testEmail := fmt.Sprintf("testuser-%d@example.com", time.Now().UnixNano())
	testPassword := "password123"
	// Use the SignUp method, which is the correct way to create a user via the interface.
	_, err = s.UserStore.SignUp(context.Background(), &domain.User{Email: testEmail}, testPassword)
	require.NoError(t, err)

	// --- Simulate Login to Get Session Cookie ---
	loginReqBody := strings.NewReader(fmt.Sprintf("email=%s&password=%s", testEmail, testPassword))
	req := httptest.NewRequest(http.MethodPost, "/auth/login", loginReqBody)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	s.E.ServeHTTP(rec, req)
	require.Equal(t, http.StatusSeeOther, rec.Code, "Login should redirect on success")

	// Extract the authentication token cookie from the response.
	// The application uses a cookie named "auth_token" for authentication, not a session cookie.
	cookies := rec.Result().Cookies()
	var authTokenCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "auth_token" {
			authTokenCookie = c
			break
		}
	}
	require.NotNil(t, authTokenCookie, "Auth token cookie not found after login")

	// --- Establish WebSocket Connections ---
	wsURL := "ws" + strings.TrimPrefix(testServer.URL, "http")
	headers := http.Header{"Cookie": {authTokenCookie.String()}}

	// Connect to the HTML channel
	htmlConn, _, err := websocket.Dial(context.Background(), wsURL+"/app/ws/html", &websocket.DialOptions{HTTPHeader: headers})
	require.NoError(t, err, "Failed to connect to /app/ws/html")
	defer htmlConn.Close(websocket.StatusNormalClosure, "")

	// Connect to the Data channel
	dataConn, _, err := websocket.Dial(context.Background(), wsURL+"/app/ws/data", &websocket.DialOptions{HTTPHeader: headers})
	require.NoError(t, err, "Failed to connect to /app/ws/data")
	defer dataConn.Close(websocket.StatusNormalClosure, "")

	// --- Concurrently Listen for Messages ---
	var wg sync.WaitGroup
	wg.Add(2)

	var receivedHTML []byte
	var receivedData []byte

	// Goroutine to listen on the HTML channel
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Loop to read messages until we find the one we want or time out.
		// This makes the test resilient to other messages (like the welcome message).
		for {
			_, msg, err := htmlConn.Read(ctx)
			if err != nil {
				return // Exit on error or timeout
			}
			// Check if this is the wargame message we are looking for.
			if bytes.Contains(msg, []byte("deals")) {
				receivedHTML = msg
				return // Found it, exit the loop.
			}
		}
	}()

	// Goroutine to listen on the Data channel
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// Loop to read messages until we find the one we want or time out.
		for {
			_, msg, err := dataConn.Read(ctx)
			if err != nil {
				return // Exit on error or timeout
			}
			// Check if this is the game state update we are looking for.
			if bytes.Contains(msg, []byte(`"eventType":"damage"`)) {
				receivedData = msg
				return // Found it, exit the loop.
			}
		}
	}()

	// --- Trigger the Event ---
	// Instead of calling the engine directly, we trigger the event via its debug HTTP endpoint.
	// This treats the server as a black box, which is better for an integration test.
	hitReq := httptest.NewRequest(http.MethodGet, "/app/debug/hit", nil)
	hitReq.AddCookie(authTokenCookie)
	hitRec := httptest.NewRecorder()
	s.E.ServeHTTP(hitRec, hitReq)
	require.Equal(t, http.StatusOK, hitRec.Code, "Triggering the hit event should succeed")

	// --- Wait for Listeners and Assert Results ---
	wg.Wait()

	// Assertions for the HTML channel
	assert.NotNil(t, receivedHTML, "Should have received a message on the HTML channel")
	if receivedHTML != nil {
		assert.True(t, bytes.Contains(receivedHTML, []byte("hx-swap-oob")), "HTML message should contain hx-swap-oob")
		assert.True(t, bytes.Contains(receivedHTML, []byte("deals")), "HTML message should contain wargame text")
	}

	// Assertions for the Data channel
	assert.NotNil(t, receivedData, "Should have received a message on the Data channel")
	if receivedData != nil {
		var gameState wargame.GameStateUpdate
		err := json.Unmarshal(receivedData, &gameState)
		assert.NoError(t, err, "Data message should be valid JSON")
		assert.Equal(t, "damage", gameState.EventType, "JSON eventType should be 'damage'")
		assert.Greater(t, gameState.DamageTaken, 0, "JSON damageTaken should be greater than 0")
	}
}

func TestServer_EmbeddedAssets_Integration(t *testing.T) {
	// Use t.Setenv to ensure environment variables are scoped to this test
	// and cleaned up automatically.
	t.Setenv("APP_TEMPLATES", "embed")
	t.Setenv("APP_STATIC", "embed")

	// Change to project root, similar to the other integration test.
	originalWD, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir("../../")
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	// Create a new server instance. It will now use embedded assets
	// because of the environment variables we set.
	s := server.New()
	s.RegisterRoutes()
	testServer := httptest.NewServer(s.E)
	defer testServer.Close()

	t.Run("it serves the home page from embedded templates", func(t *testing.T) {
		// Make a request to the root URL
		res, err := http.Get(testServer.URL + "/")
		require.NoError(t, err)
		defer res.Body.Close()

		// Check that the status code is 200 OK.
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Check that the body contains some text from the home page.
		body, err := io.ReadAll(res.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Goby", "The home page should contain the project name")
	})

	t.Run("it serves a static CSS file from embedded assets", func(t *testing.T) {
		// Make a request to a static asset
		res, err := http.Get(testServer.URL + "/static/css/style.css")
		require.NoError(t, err)
		defer res.Body.Close()

		// Check that the status code is 200 OK.
		assert.Equal(t, http.StatusOK, res.StatusCode)

		// Check that the Content-Type is correct for a CSS file.
		assert.Equal(t, "text/css; charset=utf-8", res.Header.Get("Content-Type"))
	})
}
