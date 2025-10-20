// internal/websocket/bridge_test.go
package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nfrund/goby/internal/domain"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topics"
	wsTopics "github.com/nfrund/goby/internal/topics/websocket"
	ws "github.com/nfrund/goby/internal/websocket"
)

// mockPubSub implements both pubsub.Publisher and pubsub.Subscriber for testing.
// It correctly routes published messages to subscribed handlers.
type mockPubSub struct {
	mu       sync.RWMutex
	handlers map[string][]pubsub.Handler
	messages map[string][]pubsub.Message
}

func newMockPubSub() *mockPubSub {
	return &mockPubSub{
		handlers: make(map[string][]pubsub.Handler),
		messages: make(map[string][]pubsub.Message),
	}
}

func (m *mockPubSub) Publish(ctx context.Context, msg pubsub.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store the message for inspection
	m.messages[msg.Topic] = append(m.messages[msg.Topic], msg)

	// Asynchronously deliver to handlers to mimic real pub/sub
	if handlers, ok := m.handlers[msg.Topic]; ok {
		for _, handler := range handlers {
			go handler(ctx, msg)
		}
	}
	return nil
}

func (m *mockPubSub) Subscribe(ctx context.Context, topic string, handler pubsub.Handler) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[topic] = append(m.handlers[topic], handler)
	return nil
}

func (m *mockPubSub) Close() error { return nil }

func (m *mockPubSub) getMessages(topic string) []pubsub.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions on the slice itself
	msgs := make([]pubsub.Message, len(m.messages[topic]))
	copy(msgs, m.messages[topic])
	return msgs
}

// mockTopic implements topics.Topic for testing
type mockTopic struct {
	name        string
	description string
}

func newMockTopic(name string) *mockTopic {
	return &mockTopic{
		name:        name,
		description: "Test topic: " + name,
	}
}

func (m *mockTopic) Name() string {
	return m.name
}

func (m *mockTopic) Description() string {
	return m.description
}

func (m *mockTopic) Example() string {
	return `{"example":"data"}` // Return a sample JSON string
}

func (m *mockTopic) Pattern() string {
	return m.name
}

func (m *mockTopic) Format(params interface{}) (string, error) {
	return m.name, nil
}

func (m *mockTopic) Validate(params interface{}) error {
	return nil
}

// testFixture holds all the components needed for testing the bridge.
type testFixture struct {
	bridge *ws.Bridge
	ps     *mockPubSub
	server *httptest.Server
	ctx    context.Context
	cancel context.CancelFunc
}

// setupTestFixture creates and starts all components for a test.
func setupTestFixture(t *testing.T) (*testFixture, func()) {
	t.Helper()

	// Use the integrated mockPubSub for both publisher and subscriber
	ps := newMockPubSub()

	topicRegistry := topics.NewRegistry()
	readyTopic := newMockTopic("ws.ready")

	require.NoError(t, topicRegistry.Register(wsTopics.HTMLBroadcast))
	require.NoError(t, topicRegistry.Register(wsTopics.HTMLDirect))
	require.NoError(t, topicRegistry.Register(readyTopic))

	bridge := ws.NewBridge("html", ws.BridgeDependencies{
		Publisher:     ps,
		Subscriber:    ps,
		TopicRegistry: topicRegistry,
		ReadyTopic:    readyTopic,
	})

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, bridge.Start(ctx))

	e := echo.New()
	addAuthMiddleware(e)
	e.GET("/ws/html", bridge.Handler())
	server := httptest.NewServer(e)

	cleanup := func() {
		server.Close()
		cancel()
	}

	fixture := &testFixture{
		bridge: bridge,
		ps:     ps,
		server: server,
		ctx:    ctx,
		cancel: cancel,
	}
	return fixture, cleanup
}

func connectTestClient(t *testing.T, server *httptest.Server) *websocket.Conn {
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http") + "/ws/html"
	conn, resp, err := websocket.Dial(context.Background(), wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Cookie": []string{"session=fake-session-for-testing"},
		},
	})
	require.NoError(t, err)
	require.Equal(t, http.StatusSwitchingProtocols, resp.StatusCode)
	t.Cleanup(func() {
		conn.Close(websocket.StatusNormalClosure, "test complete")
	})
	return conn
}

// addAuthMiddleware adds a middleware to the echo instance to simulate an authenticated user.
func addAuthMiddleware(e *echo.Echo) {
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("user", &domain.User{Email: "test@example.com"})
			return next(c)
		}
	})
}

func TestBridge_Subscription(t *testing.T) {
	fixture, cleanup := setupTestFixture(t)
	require.NoError(t, fixture.bridge.AllowAction("test.action"))
	defer cleanup()

	// Connect test client
	conn := connectTestClient(t, fixture.server)

	// Test subscription
	subMsg := `{"action":"subscribe","topic":"test.topic"}`
	err := conn.Write(fixture.ctx, websocket.MessageText, []byte(subMsg))
	require.NoError(t, err)

	// Test message publishing
	publishMsg := `{"action":"test.action","topic":"test.topic","payload":{"key":"value"}}`
	err = conn.Write(fixture.ctx, websocket.MessageText, []byte(publishMsg))
	require.NoError(t, err)

	// Verify the message was published
	require.Eventually(t, func() bool {
		return len(fixture.ps.getMessages("test.topic")) > 0
	}, 100*time.Millisecond, 10*time.Millisecond)

	messages := fixture.ps.getMessages("test.topic")
	require.Len(t, messages, 1)
	assert.Equal(t, "test.topic", messages[0].Topic)
	assert.JSONEq(t, `{"key":"value"}`, string(messages[0].Payload))

	// Test unsubscription
	unsubMsg := `{"action":"unsubscribe","topic":"test.topic"}`
	err = conn.Write(fixture.ctx, websocket.MessageText, []byte(unsubMsg))
	require.NoError(t, err)
}

func TestBridge_InvalidMessage(t *testing.T) {
	fixture, cleanup := setupTestFixture(t)
	conn := connectTestClient(t, fixture.server)
	defer cleanup()

	// Send invalid JSON
	err := conn.Write(fixture.ctx, websocket.MessageText, []byte(`{invalid json`))
	require.NoError(t, err)

	// To prove the connection is still alive, we send a valid message
	// and check if it gets processed. This is more reliable than a ping.
	require.NoError(t, fixture.bridge.AllowAction("valid.action"))
	subMsg := `{"action":"subscribe","topic":"valid.topic"}`
	err = conn.Write(fixture.ctx, websocket.MessageText, []byte(subMsg))
	require.NoError(t, err)

	publishMsg := `{"action":"valid.action","topic":"valid.topic","payload":{}}`
	err = conn.Write(fixture.ctx, websocket.MessageText, []byte(publishMsg))
	require.NoError(t, err)

	// If the message is published, the connection was not dropped.
	assert.Eventually(t, func() bool {
		// Check that the valid message was published after the invalid one.
		return len(fixture.ps.getMessages("valid.topic")) == 1
	}, 100*time.Millisecond, 10*time.Millisecond, "connection should remain open after invalid message")
}

func TestBridge_ConcurrentSubscriptions(t *testing.T) {
	fixture, cleanup := setupTestFixture(t)
	require.NoError(t, fixture.bridge.AllowAction("test.action"))
	defer cleanup()

	t.Run("multiple clients can subscribe and publish concurrently", func(t *testing.T) {
		// Test with multiple concurrent clients
		const numClients = 10
		var wg sync.WaitGroup
		var msgCount int32

		// Subscribe to count messages BEFORE starting the clients.
		err := fixture.ps.Subscribe(fixture.ctx, "test.topic", func(ctx context.Context, msg pubsub.Message) error {
			atomic.AddInt32(&msgCount, 1)
			return nil
		})
		require.NoError(t, err)

		wg.Add(numClients)
		for i := 0; i < numClients; i++ {
			go func(clientID int) {
				defer wg.Done()

				conn := connectTestClient(t, fixture.server)
				topic := "test.topic"
				if clientID%2 == 0 {
					topic = "alternate.topic"
				}

				// Test subscription
				subMsg := `{"action":"subscribe","topic":"` + topic + `"}`
				err := conn.Write(fixture.ctx, websocket.MessageText, []byte(subMsg))
				require.NoError(t, err)

				// Test message publishing
				publishMsg := `{"action":"test.action","topic":"` + topic + `","payload":{"client":"` + string(rune(clientID+'A')) + `"}}`
				err = conn.Write(fixture.ctx, websocket.MessageText, []byte(publishMsg))
				require.NoError(t, err)
			}(i)
		}

		wg.Wait()

		// Give the publisher a moment to process all messages from the concurrent clients.
		time.Sleep(50 * time.Millisecond)

		// Now that all clients are done, verify the message count.
		require.Eventually(t, func() bool {
			return int(atomic.LoadInt32(&msgCount)) == numClients/2
		}, 100*time.Millisecond, 10*time.Millisecond, "expected %d messages, got %d", numClients/2, atomic.LoadInt32(&msgCount))
	})
}
