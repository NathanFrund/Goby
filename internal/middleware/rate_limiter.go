package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RateLimiter creates a new rate limiter middleware with a sensible default configuration.
// It limits requests to 10 per minute per IP address for the routes it's applied to.
func RateLimiter() echo.MiddlewareFunc {
	config := middleware.RateLimiterConfig{
		// The store is responsible for saving request counts.
		// NewRateLimiterMemoryStore is a simple in-memory store suitable for single-instance deployments.
		Store: middleware.NewRateLimiterMemoryStore(10), // 10 requests per minute

		// We identify clients by their real IP address.
		IdentifierExtractor: func(c echo.Context) (string, error) {
			return c.RealIP(), nil
		},
		DenyHandler: func(c echo.Context, identifier string, err error) error {
			return c.String(http.StatusTooManyRequests, "Too many requests. Please try again later.")
		},
	}
	return middleware.RateLimiterWithConfig(config)
}
