package chat

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// mockRenderer implements rendering.Renderer for testing
type mockRenderer struct {
	mock.Mock
}

func (m *mockRenderer) RenderComponent(ctx context.Context, component interface{}) ([]byte, error) {
	args := m.Called(ctx, component)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockRenderer) RenderPage(c echo.Context, status int, component interface{}) error {
	args := m.Called(c, status, component)
	return args.Error(0)
}

// mockPublisher implements pubsub.Publisher for testing
type mockChatPublisher struct {
	messages []pubsub.Message
	mu       sync.Mutex
}

func (m *mockChatPublisher) Publish(ctx context.Context, msg pubsub.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockChatPublisher) Close() error {
	return nil
}

func (m *mockChatPublisher) getMessages() []pubsub.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]pubsub.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// mockSubscriber implements pubsub.Subscriber for testing
type mockChatSubscriber struct{}

func (m *mockChatSubscriber) Subscribe(ctx context.Context, topic string, handler pubsub.Handler) error {
	return nil
}

func (m *mockChatSubscriber) Close() error {
	return nil
}

func TestPresenceSubscriber_HandlePresenceUpdate(t *testing.T) {
	// Setup
	mockRenderer := &mockRenderer{}
	mockPublisher := &mockChatPublisher{}
	mockSubscriber := &mockChatSubscriber{}

	subscriber := NewPresenceSubscriber(mockSubscriber, mockPublisher, mockRenderer)

	// Mock expectations
	expectedHTML := []byte("<div>User List</div>")
	mockRenderer.On("RenderComponent", mock.Anything, mock.Anything).
		Return(expectedHTML, nil)

	// Create test message
	update := struct {
		Type  string   `json:"type"`
		Users []string `json:"users"`
	}{
		Type:  "presence_update",
		Users: []string{"user1", "user2"},
	}

	payload, err := json.Marshal(update)
	assert.NoError(t, err)

	msg := pubsub.Message{
		Topic:   "presence.updates",
		Payload: payload,
	}

	// Handle the message
	err = subscriber.handlePresenceUpdate(context.Background(), msg)
	assert.NoError(t, err)

	// Verify renderer was called
	mockRenderer.AssertExpectations(t)

	// Verify message was published
	messages := mockPublisher.getMessages()
	assert.Len(t, messages, 1)

	expectedOOB := `<div hx-swap-oob="innerHTML:#presence-container">` + string(expectedHTML) + `</div>`
	assert.Equal(t, expectedOOB, string(messages[0].Payload))
}

func TestPresenceSubscriber_HandleMalformedMessage(t *testing.T) {
	mockRenderer := &mockRenderer{}
	mockPublisher := &mockChatPublisher{}
	mockSubscriber := &mockChatSubscriber{}

	subscriber := NewPresenceSubscriber(mockSubscriber, mockPublisher, mockRenderer)

	// Create malformed message
	msg := pubsub.Message{
		Topic:   "presence.updates",
		Payload: []byte("invalid json"),
	}

	// Handle the message - should not return error (just skip)
	err := subscriber.handlePresenceUpdate(context.Background(), msg)
	assert.NoError(t, err)

	// Verify no messages were published
	messages := mockPublisher.getMessages()
	assert.Len(t, messages, 0)

	// Verify renderer was not called
	mockRenderer.AssertNotCalled(t, "RenderComponent")
}
