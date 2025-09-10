package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateLimiter(t *testing.T) {
	// Setup Echo
	e := echo.New()

	// Define a simple handler to be protected
	handler := func(c echo.Context) error {
		return c.String(http.StatusOK, "OK")
	}

	// Get the rate limiter middleware
	rateLimiter := RateLimiter()

	// Apply middleware to a test route
	e.GET("/", handler, rateLimiter)

	t.Run("allows requests within the limit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "192.0.2.1:1234" // Simulate a client IP
		rec := httptest.NewRecorder()

		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("blocks requests exceeding the limit", func(t *testing.T) {
		limit := 10
		clientIP := "192.0.2.2:1234"

		for i := 0; i < limit; i++ {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = clientIP
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "request %d should be allowed", i+1)
		}

		// The next request should be blocked
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = clientIP
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusTooManyRequests, rec.Code)
		assert.Contains(t, rec.Body.String(), "Too many requests")
	})
}
