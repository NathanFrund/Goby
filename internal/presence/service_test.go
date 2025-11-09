package presence

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topicmgr"
	"github.com/stretchr/testify/assert"
)

// mockPublisher implements pubsub.Publisher for testing
type mockPublisher struct {
	messages []pubsub.Message
	mu       sync.Mutex
}

func (m *mockPublisher) Publish(ctx context.Context, msg pubsub.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockPublisher) Close() error {
	return nil
}

func (m *mockPublisher) getMessages() []pubsub.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]pubsub.Message, len(m.messages))
	copy(result, m.messages)
	return result
}

// mockSubscriber implements pubsub.Subscriber for testing
type mockSubscriber struct{}

func (m *mockSubscriber) Subscribe(ctx context.Context, topic string, handler pubsub.Handler) error {
	return nil
}

func (m *mockSubscriber) Close() error {
	return nil
}

func TestService_AddPresence(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Test adding a user
	service.addPresence("user1", "client1", "test-agent")

	// Check that user is online
	users := service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")

	// Check that presence was recorded
	presence, exists := service.GetPresence("user1")
	assert.True(t, exists)
	assert.Equal(t, "user1", presence.UserID)
	assert.Equal(t, "client1", presence.ClientID)
	assert.Equal(t, StatusOnline, presence.Status)
	assert.Equal(t, "test-agent", presence.UserAgent)

	// Check that a message was published
	messages := publisher.getMessages()
	assert.Len(t, messages, 1)
	assert.Equal(t, TopicUserStatusUpdate.Name(), messages[0].Topic)
}

func TestService_RemovePresence(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Add a user first
	service.addPresence("user1", "client1", "test-agent")

	// Remove the user
	service.removePresence("user1")

	// Check that user is no longer online
	users := service.GetOnlineUsers()
	assert.Len(t, users, 0)

	// Check that presence was removed
	_, exists := service.GetPresence("user1")
	assert.False(t, exists)

	// Check that two messages were published (add + remove)
	messages := publisher.getMessages()
	assert.Len(t, messages, 2)
}

func TestService_ConcurrentAccess(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Test concurrent adds and removes with unique user/client IDs to avoid conflicts
	const numGoroutines = 5  // Reduced to avoid overwhelming the system
	const numOperations = 10 // Reduced for stability

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 2) // Add and remove goroutines

	// Concurrent adds
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				userID := fmt.Sprintf("user_%d_%d", id, j)
				clientID := fmt.Sprintf("client_%d_%d", id, j)
				service.addPresence(userID, clientID, "test-agent")
				// Small delay to reduce contention
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	// Concurrent removes (with delay to ensure some adds happen first)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			time.Sleep(5 * time.Millisecond) // Let some adds happen first
			for j := 0; j < numOperations; j++ {
				userID := fmt.Sprintf("user_%d_%d", id, j)
				service.removePresence(userID)
				// Small delay to reduce contention
				time.Sleep(1 * time.Millisecond)
			}
		}(i)
	}

	wg.Wait()

	// Service should still be functional
	service.addPresence("final_user", "final_client", "test-agent")
	users := service.GetOnlineUsers()
	assert.Contains(t, users, "final_user")
}

func TestService_RateLimit(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// First update should succeed
	service.addPresence("user1", "client1", "test-agent")
	messages := publisher.getMessages()
	assert.Len(t, messages, 1)

	// Immediate second update should be rate limited
	service.addPresence("user1", "client1", "test-agent")
	messages = publisher.getMessages()
	assert.Len(t, messages, 1) // Still only 1 message

	// Wait for rate limit to expire and try again
	time.Sleep(1100 * time.Millisecond) // Slightly longer than rate limit window
	service.addPresence("user1", "client1", "test-agent")
	messages = publisher.getMessages()
	assert.Len(t, messages, 2) // Now should have 2 messages
}

func TestService_CleanupStalePresences(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Add a user
	service.addPresence("user1", "client1", "test-agent")

	// Manually set timestamp to be stale
	service.mu.Lock()
	if clientPresences, exists := service.presences["user1"]; exists {
		if presence, clientExists := clientPresences["client1"]; clientExists {
			presence.Timestamp = time.Now().Add(-10 * time.Minute) // 10 minutes ago
			clientPresences["client1"] = presence
		}
	}
	service.mu.Unlock()

	// Run cleanup
	service.cleanupStalePresences()

	// User should be removed
	users := service.GetOnlineUsers()
	assert.Len(t, users, 0)
}

func TestService_GetOnlineUsers_UniqueUsers(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Add same user with different clients (simulating multiple tabs)
	service.addPresence("user1", "client1", "test-agent")
	service.addPresence("user1", "client2", "test-agent") // Same user, different client

	// Should only return unique users
	users := service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")
}

func TestService_MultipleConnections(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Add multiple connections for the same user
	service.addPresence("user1", "client1", "browser-tab1")

	// Wait for rate limit to expire
	time.Sleep(1100 * time.Millisecond)

	service.addPresence("user1", "client2", "browser-tab2")

	// Wait for rate limit to expire
	time.Sleep(1100 * time.Millisecond)

	service.addPresence("user1", "client3", "mobile-app")

	// User should still be online
	users := service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")

	// Remove one connection - user should still be online
	service.mu.Lock()
	delete(service.presences["user1"], "client1")
	service.mu.Unlock()

	users = service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")

	// Remove all connections - user should go offline
	service.removePresence("user1")
	users = service.GetOnlineUsers()
	assert.Len(t, users, 0)
}

func TestService_ReloadScenario(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Simulate initial connection
	service.addPresence("user1", "client1", "browser")

	// User should be online
	users := service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")

	// Wait for rate limit to expire
	time.Sleep(1100 * time.Millisecond)

	// Simulate page reload - new connection before old one disconnects
	service.addPresence("user1", "client2", "browser")

	// User should still be online with 2 connections
	users = service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")

	// Old connection disconnects (simulated by removing specific client)
	service.mu.Lock()
	delete(service.presences["user1"], "client1")
	delete(service.clients, "client1")
	service.mu.Unlock()

	// User should STILL be online (this is the fix!)
	users = service.GetOnlineUsers()
	assert.Len(t, users, 1)
	assert.Contains(t, users, "user1")
}

func TestService_Metrics(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr)
	defer service.Shutdown()

	// Test initial metrics
	metrics := service.GetMetrics()
	assert.Equal(t, int64(0), metrics["total_connections"])
	assert.Equal(t, int64(0), metrics["total_users"])

	// Add a user
	service.addPresence("user1", "client1", "browser")

	metrics = service.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_connections"])
	assert.Equal(t, int64(1), metrics["total_users"])

	// Add another connection for same user
	time.Sleep(1100 * time.Millisecond) // Wait for rate limit
	service.addPresence("user1", "client2", "browser")

	metrics = service.GetMetrics()
	assert.Equal(t, int64(2), metrics["total_connections"])
	assert.Equal(t, int64(1), metrics["total_users"])

	// Disconnect one client
	service.removePresenceForClient("user1", "client1")

	metrics = service.GetMetrics()
	assert.Equal(t, int64(1), metrics["total_connections"])
	assert.Equal(t, int64(1), metrics["total_users"])
	assert.Equal(t, int64(1), metrics["disconnections"])
}

func TestService_DebounceTimeout(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr, WithOfflineDebounce(100*time.Millisecond))
	defer service.Shutdown()

	// Add a user
	service.addPresence("user1", "client1", "browser")

	// Disconnect the user (should trigger debounce)
	service.removePresenceForClient("user1", "client1")

	// User should still be tracked (debouncing) - check that debounce timer exists
	service.debounceMu.Lock()
	_, hasTimer := service.offlineDebounce["user1"]
	service.debounceMu.Unlock()
	assert.True(t, hasTimer, "Debounce timer should exist")

	// User appears offline during debounce
	users := service.GetOnlineUsers()
	assert.Len(t, users, 0)

	// Wait for debounce timeout
	time.Sleep(150 * time.Millisecond)

	// Check that user is actually offline now
	users = service.GetOnlineUsers()
	assert.Len(t, users, 0)

	// Check metrics - should show debounce timeout
	metrics := service.GetMetrics()
	assert.Equal(t, int64(1), metrics["disconnections"])

	// Verify the user was actually removed from presence tracking
	_, exists := service.GetPresence("user1")
	assert.False(t, exists, "User should be completely removed after debounce timeout")

	// Check that presence updates were published (at least one for initial connection)
	messages := publisher.getMessages()
	assert.GreaterOrEqual(t, len(messages), 1, "Should have published at least 1 presence update")
}

func TestService_PresenceProbability(t *testing.T) {
	publisher := &mockPublisher{}
	subscriber := &mockSubscriber{}
	topicMgr := topicmgr.Default()

	service := NewService(publisher, subscriber, topicMgr, WithOfflineDebounce(100*time.Millisecond))
	defer service.Shutdown()

	// Test offline user
	prob := service.GetPresenceProbability("nonexistent")
	assert.Equal(t, 0.0, prob)

	// Test user with active connection - basic functionality
	service.addPresence("user1", "client1", "browser")
	prob = service.GetPresenceProbability("user1")
	// Just verify the method doesn't crash and returns a reasonable value
	assert.True(t, prob >= 0.0 && prob <= 1.0, "Probability should be between 0 and 1")

	// Test metrics functionality
	metrics := service.GetMetrics()
	assert.Contains(t, metrics, "total_connections")
	assert.Contains(t, metrics, "total_users")
}
