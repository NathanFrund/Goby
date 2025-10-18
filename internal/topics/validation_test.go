package topics_test

import (
	"testing"

	"github.com/nfrund/goby/internal/topics"
	"github.com/stretchr/testify/assert"
)

func TestTopic_Validate(t *testing.T) {
	tests := []struct {
		name    string
		topic   topics.Topic
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid topic",
			topic: topics.Topic{
				Name:        "valid_topic",
				Description: "A valid topic",
				Pattern:     "valid.topic.{param}",
				Example:     "valid.topic.value",
			},
			wantErr: false,
		},
		{
			name: "invalid name - uppercase",
			topic: topics.Topic{
				Name:        "InvalidTopic",
				Description: "Invalid topic name",
				Pattern:     "invalid.topic",
				Example:     "invalid.topic",
			},
			wantErr: true,
			errMsg:  topics.ErrInvalidTopicName.Error(),
		},
		{
			name: "invalid pattern - spaces",
			topic: topics.Topic{
				Name:        "invalid_pattern",
				Description: "Invalid pattern",
				Pattern:     "invalid pattern",
				Example:     "invalid.pattern",
			},
			wantErr: true,
			errMsg:  topics.ErrInvalidTopicPattern.Error(),
		},
		{
			name: "missing description",
			topic: topics.Topic{
				Name:    "missing_description",
				Pattern: "missing.description",
				Example: "missing.description",
			},
			wantErr: true,
			errMsg:  topics.ErrMissingDescription.Error(),
		},
		{
			name: "missing example",
			topic: topics.Topic{
				Name:        "missing_example",
				Description: "Missing example",
				Pattern:     "missing.example",
			},
			wantErr: true,
			errMsg:  topics.ErrMissingExample.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.topic.Validate()
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAndRegister(t *testing.T) {
	t.Run("valid topic", func(t *testing.T) {
		topics.ResetRegistryForTesting()
		
		topic := topics.Topic{
			Name:        "test_validate_register",
			Description: "Test validate and register",
			Pattern:     "test.validate.register",
			Example:     "test.validate.register",
		}

		err := topics.ValidateAndRegister(topic)
		assert.NoError(t, err)

		// Verify the topic was registered
		_, exists := topics.Get("test_validate_register")
		assert.True(t, exists, "Topic should be registered")
	})

	t.Run("invalid topic", func(t *testing.T) {
		topics.ResetRegistryForTesting()
		
		topic := topics.Topic{
			Name:        "invalid topic", // Invalid name (contains space)
			Description: "Invalid topic",
			Pattern:     "invalid.topic",
			Example:     "invalid.topic",
		}

		err := topics.ValidateAndRegister(topic)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), topics.ErrInvalidTopicName.Error())

		// Verify the topic was not registered
		_, exists := topics.Get("invalid topic")
		assert.False(t, exists, "Invalid topic should not be registered")
	})
}
