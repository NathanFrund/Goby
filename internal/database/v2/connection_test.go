package v2

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/surrealdb/surrealdb.go"
)

func TestConnection_WithConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Setup a standard connection
	cfg := testutils.ConfigForTests(t)
	conn := NewConnection(cfg)
	err := conn.Connect(context.Background())
	require.NoError(t, err, "Initial connection should succeed")
	defer conn.Close(context.Background())

	t.Run("reconnects on connection error", func(t *testing.T) {
		// 2. Define a function that fails on the first call and succeeds on the second
		var callCount int
		// Use an error message that isConnectionError will detect
		simulatedFailureError := errors.New("simulated error: unexpected eof")

		testFunc := func(db *surrealdb.DB) error {
			callCount++
			if callCount == 1 {
				// First call: simulate a connection failure
				return simulatedFailureError
			}
			// Second call: perform a real, simple query that should succeed
			_, err := surrealdb.Query[any](context.Background(), db, "RETURN 1", nil)
			return err
		}

		// 3. Execute the test
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// This should trigger the reconnection logic and succeed on the second attempt.
		err = conn.WithConnection(ctx, testFunc)

		// 4. Assert the outcome
		require.NoError(t, err, "WithConnection should ultimately succeed after reconnecting")
		assert.Equal(t, 2, callCount, "The function should have been called twice (initial + retry)")
	})

	t.Run("does not reconnect on application error", func(t *testing.T) {
		var callCount int
		// Use a standard application-level error
		appError := errors.New("application-level error: record not found")

		testFunc := func(db *surrealdb.DB) error {
			callCount++
			return appError
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := conn.WithConnection(ctx, testFunc)

		// Assert that the original error was returned and the function was only called once
		require.Error(t, err)
		assert.ErrorIs(t, err, appError, "The original application error should be returned")
		assert.Equal(t, 1, callCount, "The function should only be called once")
	})
}
