package view_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/view"
	"github.com/stretchr/testify/assert"
)

const testSessionSecret = "a-very-secret-key-for-testing-!"

func setupTestContext() (echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// Create a new session store for the test.
	store := sessions.NewCookieStore([]byte(testSessionSecret))
	// Create a middleware function that will be used to wrap our test handler.
	sessionMiddleware := session.Middleware(store)

	// Create a dummy handler that will be wrapped by the session middleware.
	// This ensures the session is properly initialized in the context.
	var c echo.Context
	handler := func(ctx echo.Context) error { c = ctx; return nil }
	sessionMiddleware(handler)(e.NewContext(req, rec))

	return c, rec
}

func TestFlashMessages(t *testing.T) {
	t.Run("Set and Get Success Flash", func(t *testing.T) {
		c, _ := setupTestContext()

		// Set a success flash
		view.SetFlashSuccess(c, "It worked!")

		// Get flashes
		flashes := view.GetFlashes(c)

		assert.NotNil(t, flashes)
		assert.Contains(t, flashes, "success")
		assert.Equal(t, "It worked!", flashes["success"][0])

		// Get flashes again to ensure they are cleared
		flashesAfterRead := view.GetFlashes(c)
		assert.NotContains(t, flashesAfterRead, "success", "Flashes should be cleared after being read")
	})

	t.Run("Set and Get Error Flash", func(t *testing.T) {
		c, _ := setupTestContext()

		// Set an error flash
		view.SetFlashError(c, "It failed!")

		// Get flashes
		flashes := view.GetFlashes(c)

		assert.NotNil(t, flashes)
		assert.Contains(t, flashes, "error")
		assert.Equal(t, "It failed!", flashes["error"][0])
	})

	t.Run("GetFlashes with no flashes set", func(t *testing.T) {
		c, _ := setupTestContext()

		flashes := view.GetFlashes(c)
		assert.Empty(t, flashes, "Should return an empty map when no flashes are set")
	})
}
