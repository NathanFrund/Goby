package topics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nfrund/goby/internal/topics"
)

func TestRegistry(t *testing.T) {
	registry := topics.NewRegistry()

	t.Run("Register and Get", func(t *testing.T) {
		topic := topics.NewTestTopic(
			"test_topic",
			"A test topic",
			"test.pattern.{param}",
			"test.pattern.value",
		)

		err := registry.Register(topic)
		assert.NoError(t, err, "Register should succeed")

		found, exists := registry.Get("test_topic")
		assert.True(t, exists, "Topic should exist after registration")
		assert.Equal(t, topic.Name(), found.Name(), "Retrieved topic should match registered topic")
	})

	t.Run("Get Non-Existent Topic", func(t *testing.T) {
		_, exists := registry.Get("non_existent_topic")
		assert.False(t, exists, "Non-existent topic should not be found")
	})

	t.Run("List Topics", func(t *testing.T) {
		registry = topics.NewRegistry()

		t1 := topics.NewTestTopic("topic1", "Topic 1", "topic.1", "topic.1")
		t2 := topics.NewTestTopic("topic2", "Topic 2", "topic.2", "topic.2")

		err1 := registry.Register(t1)
		err2 := registry.Register(t2)
		assert.NoError(t, err1, "Register t1 should succeed")
		assert.NoError(t, err2, "Register t2 should succeed")

		all := registry.List()
		assert.Len(t, all, 2, "Should return all registered topics")
		var names []string
		for _, t := range all {
			names = append(names, t.Name())
		}
		assert.Contains(t, names, "topic1", "Should contain first topic")
		assert.Contains(t, names, "topic2", "Should contain second topic")
	})

	t.Run("Prevent Duplicate Registration", func(t *testing.T) {
		registry = topics.NewRegistry()

		topic := topics.NewTestTopic("duplicate", "Duplicate topic", "duplicate", "duplicate")
		err1 := registry.Register(topic)
		assert.NoError(t, err1, "First register should succeed")

		err2 := registry.Register(topic)
		assert.Error(t, err2, "Second register should fail")
		assert.Contains(t, err2.Error(), "already registered", "Error should indicate duplicate registration")
	})
}

func TestDefaultRegistry(t *testing.T) {
	t.Run("Default registry is a singleton", func(t *testing.T) {
		r1 := topics.Default()
		r2 := topics.Default()
		assert.Equal(t, r1, r2, "Default() should return the same instance")
	})

	t.Run("Register with default registry", func(t *testing.T) {
		// Reset the default registry for testing
		topics.Default().Reset()

		topic := topics.NewTestTopic(
			"default_registry_topic",
			"Topic for default registry test",
			"test.default",
			"test.default",
		)

		err := topics.Register(topic)
		assert.NoError(t, err, "Register with default registry should succeed")

		found, exists := topics.Get("default_registry_topic")
		assert.True(t, exists, "Topic should exist in default registry after registration")
		assert.Equal(t, topic.Name(), found.Name(), "Retrieved topic should match registered topic")
	})
}
