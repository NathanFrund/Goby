package topics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/nfrund/goby/internal/topics"
)

func TestTopic_Validate(t *testing.T) {
	validTopic := topics.NewTestTopic("valid_topic", "A valid topic", "valid.pattern", "valid.pattern")
	
	t.Run("valid topic", func(t *testing.T) {
		err := topics.Validate(validTopic)
		assert.NoError(t, err)
	})

	t.Run("invalid name with space", func(t *testing.T) {
		topic := topics.NewTestTopic("invalid topic", "Invalid topic name", "invalid.pattern", "invalid.pattern")
		err := topics.Validate(topic)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be lowercase alphanumeric")
	})

	t.Run("invalid name with uppercase", func(t *testing.T) {
		topic := topics.NewTestTopic("InvalidTopic", "Invalid topic name", "invalid.pattern", "invalid.pattern")
		err := topics.Validate(topic)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be lowercase alphanumeric")
	})

	t.Run("empty name", func(t *testing.T) {
		topic := topics.NewTestTopic("", "Empty topic name", "invalid.pattern", "invalid.pattern")
		err := topics.Validate(topic)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Name: cannot be empty")
	})
}

// testTopic is now defined in test_helpers.go

func TestValidateAndRegister(t *testing.T) {
	t.Run("valid topic", func(t *testing.T) {
		registry := topics.NewRegistry()
		
		topic := topics.NewTestTopic(
			"test_validate_register",
			"Test validate and register",
			"test.validate.register",
			"test.validate.register",
		)

		err := topics.ValidateAndRegister(registry, topic)
		assert.NoError(t, err)

		// Verify the topic was registered
		_, exists := registry.Get("test_validate_register")
		assert.True(t, exists, "Topic should be registered")
	})

	t.Run("invalid topic", func(t *testing.T) {
		registry := topics.NewRegistry()
		
		topic := topics.NewTestTopic(
			"invalid topic", // Invalid name (contains space)
			"Invalid topic",
			"invalid.topic",
			"invalid.topic",
		)

		err := topics.ValidateAndRegister(registry, topic)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be lowercase alphanumeric")

		// Verify the topic was not registered
		_, exists := registry.Get("invalid topic")
		assert.False(t, exists, "Invalid topic should not be registered")
	})
}
