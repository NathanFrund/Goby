package topics

import (
	"fmt"
	"strings"
)

// Topic defines the interface that all topic types must implement
type Topic interface {
	// Name returns the unique identifier for this topic
	Name() string

	// Description returns a human-readable description of the topic
	Description() string

	// Pattern returns the topic's pattern string with placeholders
	Pattern() string

	// Format generates the full topic string using the provided variables
	Format(vars interface{}) (string, error)

	// Example returns an example of how to use this topic
	Example() string

	// Validate checks if the provided variables are valid for this topic
	Validate(vars interface{}) error
}

// BaseTopic provides a base implementation of the Topic interface
type BaseTopic struct {
	name        string
	description string
	pattern     string
	example     string
}

// Pattern returns the topic's pattern string with placeholders
func (t BaseTopic) Pattern() string {
	return t.pattern
}

// NewBaseTopic creates a new BaseTopic
func NewBaseTopic(name, description, pattern, example string) BaseTopic {
	return BaseTopic{
		name:        name,
		description: description,
		pattern:     pattern,
		example:     example,
	}
}

// Name returns the topic's name
func (t BaseTopic) Name() string {
	return t.name
}

// Description returns the topic's description
func (t BaseTopic) Description() string {
	return t.description
}

// Example returns an example of how to use this topic
func (t BaseTopic) Example() string {
	return t.example
}

// Format formats the topic with the given variables
func (t BaseTopic) Format(vars interface{}) (string, error) {
	params, ok := vars.(map[string]string)
	if !ok {
		return "", fmt.Errorf("expected map[string]string, got %T", vars)
	}

	result := t.pattern
	for k, v := range params {
		result = strings.ReplaceAll(result, "{"+k+"}", v)
	}

	// Verify all placeholders were replaced
	if strings.Contains(result, "{") || strings.Contains(result, "}") {
		return "", fmt.Errorf("missing required parameters in topic format")
	}

	return result, nil
}

// Validate checks if the provided variables are valid for this topic
func (t BaseTopic) Validate(vars interface{}) error {
	// Default implementation just checks if vars is a non-nil map
	if vars == nil {
		return fmt.Errorf("vars cannot be nil")
	}
	
	if _, ok := vars.(map[string]string); !ok {
		return fmt.Errorf("expected map[string]string, got %T", vars)
	}
	
	return nil
}

// String returns the topic's name
func (t BaseTopic) String() string {
	return t.name
}
