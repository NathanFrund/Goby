package topics_test

import (
	"testing"

	"github.com/nfrund/goby/internal/topics"
	"github.com/stretchr/testify/assert"
)

func TestTopicRegistry(t *testing.T) {
	// Reset the registry for testing
	resetRegistry()

	t.Run("Register and Get", func(t *testing.T) {
		topic := topics.Topic{
			Name:        "test_topic",
			Description: "A test topic",
			Pattern:     "test.pattern.{param}",
			Example:     "test.pattern.value",
		}

		topics.Register(topic)

		found, exists := topics.Get("test_topic")
		assert.True(t, exists, "Topic should exist after registration")
		assert.Equal(t, topic, found, "Retrieved topic should match registered topic")
	})

	t.Run("Get Non-Existent Topic", func(t *testing.T) {
		_, exists := topics.Get("non_existent_topic")
		assert.False(t, exists, "Non-existent topic should not be found")
	})

	t.Run("List Topics", func(t *testing.T) {
		resetRegistry()

		t1 := topics.Topic{Name: "topic1", Description: "Topic 1", Pattern: "topic.1", Example: "topic.1"}
		t2 := topics.Topic{Name: "topic2", Description: "Topic 2", Pattern: "topic.2", Example: "topic.2"}

		topics.Register(t1)
		topics.Register(t2)

		all := topics.List()
		assert.Len(t, all, 2, "Should return all registered topics")
		assert.Contains(t, all, t1, "Should contain first topic")
		assert.Contains(t, all, t2, "Should contain second topic")
	})

	t.Run("Prevent Duplicate Registration", func(t *testing.T) {
		resetRegistry()

		topic := topics.Topic{Name: "duplicate", Description: "Duplicate topic", Pattern: "duplicate", Example: "duplicate"}
		topics.Register(topic)

		assert.Panics(t, func() {
			topics.Register(topic)
		}, "Should panic when registering duplicate topic")
	})
}

func TestTopic_Format(t *testing.T) {
	topic := topics.Topic{
		Name:    "formattable",
		Pattern: "test.{param1}.{param2}",
	}

	t.Run("Format with all parameters", func(t *testing.T) {
		result := topic.Format(map[string]string{
			"param1": "value1",
			"param2": "value2",
		})
		assert.Equal(t, "test.value1.value2", result)
	})

	t.Run("Format with missing parameters", func(t *testing.T) {
		result := topic.Format(map[string]string{
			"param1": "value1",
			// param2 is missing
		})
		assert.Equal(t, "test.value1.{param2}", result, "Should leave placeholders for missing parameters")
	})

	t.Run("Format with extra parameters", func(t *testing.T) {
		result := topic.Format(map[string]string{
			"param1":    "value1",
			"param2":    "value2",
			"extra_param": "should_be_ignored",
		})
		assert.Equal(t, "test.value1.value2", result, "Should ignore extra parameters")
	})
}

// resetRegistry clears the topic registry for testing
func resetRegistry() {
	// This is a hack to reset the package-level registry for testing
	topics.ResetRegistryForTesting() // We'll need to add this function to the topics package
}
