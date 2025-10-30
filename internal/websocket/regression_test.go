package websocket_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nfrund/goby/internal/topicmgr"
	wsTopics "github.com/nfrund/goby/internal/websocket"
)

// TestTopicRegistrationIdempotency ensures that topic registration is idempotent
// and doesn't fail when topics are already registered (regression test for database refactoring issues)
func TestTopicRegistrationIdempotency(t *testing.T) {
	// Create isolated topic manager
	manager := topicmgr.NewManager()

	// First registration should succeed
	err := wsTopics.RegisterTopicsWithManager(manager)
	require.NoError(t, err, "First topic registration should succeed")

	// Second registration should also succeed (idempotent)
	err = wsTopics.RegisterTopicsWithManager(manager)
	require.NoError(t, err, "Second topic registration should succeed (idempotent)")

	// Verify topics are registered
	topics := manager.List()
	assert.GreaterOrEqual(t, len(topics), 6, "Should have at least 6 WebSocket topics registered")

	// Verify specific topics exist
	_, exists := manager.Get("ws.html.broadcast")
	assert.True(t, exists, "HTML broadcast topic should be registered")

	_, exists = manager.Get("ws.data.broadcast")
	assert.True(t, exists, "Data broadcast topic should be registered")
}

// TestFrameworkTopicValidation ensures that framework topics don't have modules
// (regression test for topic validation issues)
func TestFrameworkTopicValidation(t *testing.T) {
	// Test that framework topics are created without modules
	topics := []topicmgr.Topic{
		wsTopics.TopicHTMLBroadcast,
		wsTopics.TopicHTMLDirect,
		wsTopics.TopicDataBroadcast,
		wsTopics.TopicDataDirect,
		wsTopics.TopicClientReady,
		wsTopics.TopicClientDisconnected,
	}

	for _, topic := range topics {
		t.Run(topic.Name(), func(t *testing.T) {
			// Framework topics should not have a module
			assert.Empty(t, topic.Module(), "Framework topic %s should not have a module", topic.Name())

			// Framework topics should have framework scope
			assert.Equal(t, topicmgr.ScopeFramework, topic.Scope(), "Topic %s should have framework scope", topic.Name())

			// Should be able to register without validation errors
			manager := topicmgr.NewManager()
			err := manager.Register(topic)
			assert.NoError(t, err, "Framework topic %s should register without validation errors", topic.Name())
		})
	}
}

// TestTestFixtureIsolation ensures that test fixtures properly isolate topic managers
// (regression test for test isolation issues)
func TestTestFixtureIsolation(t *testing.T) {
	// Create first test manager
	testMgr1 := NewTestTopicManager(t)
	defer testMgr1.Cleanup()

	// Register topics in first manager
	err := testMgr1.Manager().Register(wsTopics.TopicHTMLBroadcast)
	require.NoError(t, err)

	// Create second test manager
	testMgr2 := NewTestTopicManager(t)
	defer testMgr2.Cleanup()

	// Second manager should be isolated (empty)
	topics1 := testMgr1.Manager().List()
	topics2 := testMgr2.Manager().List()

	assert.Len(t, topics1, 1, "First manager should have 1 topic")
	assert.Len(t, topics2, 0, "Second manager should be isolated and empty")

	// Register same topic in second manager (should not conflict)
	err = testMgr2.Manager().Register(wsTopics.TopicHTMLBroadcast)
	require.NoError(t, err, "Should be able to register same topic in isolated manager")
}
