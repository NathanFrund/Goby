package middleware

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v4"
)

type contextKey string

const loggerKey = contextKey("logger")

// Logger is a middleware that injects a request-scoped logger into the context.
// This logger is pre-configured with the request ID from the RequestID middleware.
// It should be placed after the RequestID middleware in the chain.
func Logger(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		reqID := c.Response().Header().Get(echo.HeaderXRequestID)
		requestLogger := slog.Default().With("request_id", reqID)

		// Create a new context with the logger and set it on the request.
		newCtx := context.WithValue(c.Request().Context(), loggerKey, requestLogger)
		c.SetRequest(c.Request().WithContext(newCtx))

		return next(c)
	}
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
