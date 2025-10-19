package topics

import (
	"errors"
	"fmt"
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
	// nameRegex defines the valid format for topic names
	nameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z0-9_]+)*$`)
)

// ValidationError represents a topic validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s: %s", e.Field, e.Message)
}

// Validate checks if a topic is valid according to the following rules:
// 1. Name must be lowercase with dots or underscores
// 2. Description must be non-empty
// 3. Pattern must be valid
// 4. Example must be non-empty and match the pattern format
func Validate(t Topic) error {
	if t == nil {
		return ValidationError{"Topic", "cannot be nil"}
	}

	// Validate name
	name := t.Name()
	if name == "" {
		return ValidationError{"Name", "cannot be empty"}
	}

	if !nameRegex.MatchString(name) {
		return ValidationError{"Name", "must be lowercase alphanumeric with dots or underscores"}
	}

	// Validate description
	desc := t.Description()
	if strings.TrimSpace(desc) == "" {
		return ValidationError{"Description", "cannot be empty"}
	}

	// Validate pattern
	pattern := t.Pattern()
	if pattern == "" {
		return ValidationError{"Pattern", "cannot be empty"}
	}

	// Validate example
	example := t.Example()
	if strings.TrimSpace(example) == "" {
		return ValidationError{"Example", "cannot be empty"}
	}

	// Test format with example variables
	if _, err := t.Format(map[string]string{"test": "value"}); err != nil {
		return ValidationError{"Format", fmt.Sprintf("failed to format with test values: %v", err)}
	}

	return nil
}

// ValidateAndRegister validates the topic and registers it if valid
func ValidateAndRegister(registry *TopicRegistry, topic Topic) error {
	if err := Validate(topic); err != nil {
		return err
	}
	return registry.Register(topic)
}
