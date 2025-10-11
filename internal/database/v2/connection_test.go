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

func TestConnection_ReconnectsOnFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 1. Setup a standard connection
	cfg := testutils.ConfigForTests(t)
	conn := NewConnection(cfg)
	err := conn.Connect(context.Background())
	require.NoError(t, err, "Initial connection should succeed")
	defer conn.Close(context.Background())

	// 2. Define a function that fails on the first call and succeeds on the second
	var callCount int
	simulatedFailureError := errors.New("simulated connection drop")

	testFunc := func(db *surrealdb.DB) error {
		callCount++
		if callCount == 1 {
			// First call: simulate a failure
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
}
