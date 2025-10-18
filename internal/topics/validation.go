package topics

import (
	"errors"
	"regexp"
	"strings"
)

var (
	// ErrInvalidTopicName is returned when a topic name is invalid
	ErrInvalidTopicName = errors.New("topic name must be lowercase with underscores, e.g., 'chat_message'")
	
	// ErrInvalidTopicPattern is returned when a topic pattern is invalid
	ErrInvalidTopicPattern = errors.New("topic pattern must use dot notation with optional {parameters}")
	
	// ErrMissingDescription is returned when a topic is missing a description
	ErrMissingDescription = errors.New("topic is missing a description")
	
	// ErrMissingExample is returned when a topic is missing an example
	ErrMissingExample = errors.New("topic is missing an example")
)

var (
	topicNameRegex   = regexp.MustCompile(`^[a-z][a-z0-9_]*(_[a-z0-9]+)*$`)
	// Allow any combination of alphanumeric characters, dots, and {param} placeholders
	topicPatternRegex = regexp.MustCompile(`^[a-zA-Z0-9.{}_]+$`)
)

// Validate checks if a topic is valid according to the following rules:
// 1. Name must be lowercase with underscores (snake_case)
// 2. Pattern must use dot notation with optional {parameters}
// 3. Description must be non-empty
// 4. Example must be non-empty and match the pattern format
func (t Topic) Validate() error {
	if !topicNameRegex.MatchString(t.Name) {
		return ErrInvalidTopicName
	}

	if !topicPatternRegex.MatchString(t.Pattern) {
		return ErrInvalidTopicPattern
	}

	if strings.TrimSpace(t.Description) == "" {
		return ErrMissingDescription
	}

	if strings.TrimSpace(t.Example) == "" {
		return ErrMissingExample
	}

	// Verify that the example matches the pattern structure (simplified check)
	// This is a basic check and might need to be more sophisticated
	if !strings.Contains(t.Example, ".") && t.Pattern != t.Example {
		return errors.New("example should match the pattern structure")
	}

	return nil
}

// ValidateAndRegister validates the topic and registers it if valid
func ValidateAndRegister(topic Topic) error {
	if err := topic.Validate(); err != nil {
		return err
	}
	Register(topic)
	return nil
}
