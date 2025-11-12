package pubsub

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTracingMiddleware(t *testing.T) {
	// Setup OTel with a test tracer (we'll use a no-op tracer for this test)
	ctx := context.Background()
	config := TracingConfig{
		Enabled:     true,
		ServiceName: "test-service",
		ZipkinURL:   "http://localhost:9411/api/v2/spans",
	}
	tracer, cleanup, err := SetupOTel(ctx, config)
	require.NoError(t, err)
	defer cleanup()

	// Create a bridge with tracing
	bridge := NewWatermillBridgeWithTracer(tracer)
	require.NotNil(t, bridge)

	// Test publishing a message
	testMsg := Message{
		Topic:   "test.topic",
		UserID:  "user123",
		Payload: []byte(`{"message": "hello world", "data": {"key": "value"}}`),
		Metadata: map[string]string{
			"request_id": "req-123",
		},
	}

	err = bridge.Publish(ctx, testMsg)
	assert.NoError(t, err)

	// Test subscribing to messages
	handler := func(ctx context.Context, msg Message) error {
		// Handler just needs to exist for the test
		return nil
	}

	err = bridge.Subscribe(ctx, "test.topic", handler)
	assert.NoError(t, err)

	// Publish again to trigger the subscription
	err = bridge.Publish(ctx, testMsg)
	assert.NoError(t, err)

	// Note: In a real test, we'd need to wait for the message to be processed
	// For this basic test, we're just ensuring the setup works without errors

	// Clean up
	err = bridge.Close()
	assert.NoError(t, err)
}

func TestSetupOTel(t *testing.T) {
	ctx := context.Background()

	t.Run("disabled tracing", func(t *testing.T) {
		config := TracingConfig{Enabled: false}
		tracer, cleanup, err := SetupOTel(ctx, config)
		require.NoError(t, err)
		require.NotNil(t, tracer)
		require.NotNil(t, cleanup)

		// Should be a no-op tracer
		_, span := tracer.Start(ctx, "test")
		span.End()

		// Clean up
		cleanup()
	})

	t.Run("enabled tracing with invalid URL", func(t *testing.T) {
		config := TracingConfig{
			Enabled:     true,
			ServiceName: "test-service",
			ZipkinURL:   "http://invalid-url:9411/api/v2/spans",
		}
		tracer, cleanup, err := SetupOTel(ctx, config)
		require.NoError(t, err)
		require.NotNil(t, tracer)
		require.NotNil(t, cleanup)

		// Clean up
		cleanup()
	})

	t.Run("legacy SetupOTelSimple function", func(t *testing.T) {
		tracer, cleanup, err := SetupOTelSimple(ctx, "test-service", "http://invalid-url:9411/api/v2/spans")
		require.NoError(t, err)
		require.NotNil(t, tracer)
		require.NotNil(t, cleanup)

		// Clean up
		cleanup()
	})
}
