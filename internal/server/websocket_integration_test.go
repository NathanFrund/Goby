package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBridge_Integration verifies that both the HTML and Data bridges
// correctly receive messages from the pub/sub system and forward them to the
// appropriate WebSocket clients.
func TestBridge_Integration(t *testing.T) {
	s, testServer, cleanup := setupIntegrationTest(t)
	defer cleanup()

	// Create a context with timeout for the test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Channel to signal when the test is done
	done := make(chan struct{})
	defer close(done)

	// Establish WebSocket connections once for all sub-tests.
	header := http.Header{}
	header.Add("Cookie", "session=fake-session-for-testing")

	// HTML Connection
	htmlWsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws/html"
	htmlConn, _, err := websocket.DefaultDialer.DialContext(ctx, htmlWsURL, header)
	require.NoError(t, err, "Failed to connect to HTML websocket")
	defer func() {
		htmlConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		htmlConn.Close()
	}()

	// Data Connection
	dataWsURL := "ws" + strings.TrimPrefix(testServer.URL, "http") + "/ws/data"
	dataConn, _, err := websocket.DefaultDialer.DialContext(ctx, dataWsURL, header)
	require.NoError(t, err, "Failed to connect to Data websocket")
	defer func() {
		dataConn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		dataConn.Close()
	}()

// --- HTML Bridge Test ---
	t.Run("HTML bridge broadcasts HTML fragments", func(t *testing.T) {
		// Publish a message with an HTML payload to the broadcast topic
		htmlPayload := `<div id="test-id">Hello, HTMX!</div>`
		err := s.PubSub.Publish(context.Background(), pubsub.Message{
			Topic:   "ws.html.broadcast",
			Payload: []byte(htmlPayload),
		})
		require.NoError(t, err, "Failed to publish to html-broadcast")

		// Read messages until we find our test message or time out
		timeout := time.After(2 * time.Second)
		for {
			select {
			case <-timeout:
				t.Fatal("Timed out waiting for test message")
			default:
				htmlConn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
				_, p, err := htmlConn.ReadMessage()
				if err != nil {
					if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
						t.Fatalf("Unexpected error reading message: %v", err)
					}
					continue
				}
				
				// Skip chat-related messages
				if strings.Contains(string(p), "chat-messages") {
					continue
				}
				
				// Found our test message
				assert.Equal(t, htmlPayload, string(p), "HTML websocket should receive the correct HTML fragment")
				return
			}
		}
	})

// --- Data Bridge Test ---
	t.Run("Data bridge broadcasts JSON data", func(t *testing.T) {
		// Prepare and publish a structured data message
		dataPayload := map[string]interface{}{
			"type": "test_event",
			"data": map[string]interface{}{
				"message": "Hello, Data Client!",
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		payloadBytes, err := json.Marshal(dataPayload)
		require.NoError(t, err)

		err = s.PubSub.Publish(context.Background(), pubsub.Message{
			Topic:   "ws.data.broadcast",
			Payload: payloadBytes,
		})
		require.NoError(t, err, "Failed to publish to data-broadcast")

		// Read the message from the WebSocket and assert its content
		dataConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, p, err := dataConn.ReadMessage()
		if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			require.NoError(t, err, "Failed to read message from Data websocket")
		}

		// Parse the received JSON and verify its structure
		var received map[string]interface{}
		err = json.Unmarshal(p, &received)
		require.NoError(t, err, "Failed to unmarshal received message")
		assert.Equal(t, "test_event", received["type"], "Received message should have the correct type")
		assert.NotEmpty(t, received["timestamp"], "Received message should have a timestamp")
	})

	// --- Data Bridge Test ---
	t.Run("Data bridge broadcasts JSON data", func(t *testing.T) {
		// 2. Prepare and publish a structured data message
		dataPayload := map[string]interface{}{
			"type": "test_event",
			"data": map[string]interface{}{
				"message": "Hello, Data Client!",
			},
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		}
		payloadBytes, err := json.Marshal(dataPayload)
		require.NoError(t, err)

		err = s.PubSub.Publish(context.Background(), pubsub.Message{
			Topic:   "ws.data.broadcast",
			Payload: payloadBytes,
		})
		require.NoError(t, err, "Failed to publish to data-broadcast")

		// 3. Read the message from the WebSocket and assert its content
		dataConn.SetReadDeadline(time.Now().Add(2 * time.Second))
		_, p, err := dataConn.ReadMessage()
		if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			require.NoError(t, err, "Failed to read message from Data websocket")
		}

		// Unmarshal both the original and received payloads to compare them structurally
		var receivedPayload map[string]interface{}
		err = json.Unmarshal(p, &receivedPayload)
		require.NoError(t, err, "Failed to unmarshal received JSON payload")

		// Assert fields individually for better error messages
		assert.Equal(t, dataPayload["type"], receivedPayload["type"])
		assert.Equal(t, dataPayload["data"], receivedPayload["data"])
		// Timestamps can have slight precision differences, so checking for presence is enough
		assert.NotEmpty(t, receivedPayload["timestamp"])
	})
}
