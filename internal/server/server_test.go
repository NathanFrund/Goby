package server

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPErrorHandler_WithStackTrace(t *testing.T) {
	// --- Setup ---
	e := echo.New()

	// 1. Capture log output
	// We temporarily redirect slog's output to a buffer to inspect it.
	var logBuffer bytes.Buffer
	// Create a new logger that writes to our buffer
	handler := slog.NewTextHandler(&logBuffer, &slog.HandlerOptions{
		AddSource: true,
	})
	logger := slog.New(handler)
	// Store the original default logger and defer its restoration
	originalLogger := slog.Default()
	slog.SetDefault(logger)
	defer slog.SetDefault(originalLogger)

	// 2. Set up the error handler we want to test
	setupErrorHandling(e)

	// 3. Define a route that will always produce an unhandled error
	e.GET("/test-unhandled-error", func(c echo.Context) error {
		// This is the kind of error that should trigger our stack trace logging.
		return errors.New("a deliberate unhandled error occurred")
	})

	// --- Act ---
	req := httptest.NewRequest(http.MethodGet, "/test-unhandled-error", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	// --- Assert ---
	// First, check that the HTTP response is correct (a 500 error)
	require.Equal(t, http.StatusInternalServerError, rec.Code, "Expected a 500 Internal Server Error response")

	// Now, check the captured log output
	logOutput := logBuffer.String()

	// Assert that the log contains the key pieces of information
	assert.Contains(t, logOutput, "Internal Server Error (Unhandled)", "Log message should indicate an unhandled error")
	assert.Contains(t, logOutput, "error=\"a deliberate unhandled error occurred\"", "Log should contain the original error message")
	assert.Contains(t, logOutput, "stack_trace=", "Log must contain the stack_trace field")

	// A good stack trace will contain the path to the Go runtime and this test file.
	// This is a strong indicator that a real stack trace was captured.
	assert.Contains(t, logOutput, "runtime/debug/stack.go", "Stack trace should originate from the debug package")
	assert.Contains(t, logOutput, "internal/server/server_test.go", "Stack trace should point back to this test file")
}
