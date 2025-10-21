package topicmgr

import (
	"time"
)

// Topic represents a strongly-typed topic identifier with compile-time safety
type Topic interface {
	// Name returns the unique string identifier for this topic
	Name() string

	// Module returns the module that owns this topic (empty for framework topics)
	Module() string

	// Description returns human-readable documentation
	Description() string

	// Pattern returns the routing pattern (for wildcards)
	Pattern() string

	// Example returns a usage example
	Example() string

	// Metadata returns additional topic information
	Metadata() map[string]interface{}

	// Scope returns whether this is a framework or module topic
	Scope() TopicScope
}

// TypedTopic provides compile-time safety for topic usage
type TypedTopic struct {
	name        string
	module      string
	description string
	pattern     string
	example     string
	metadata    map[string]interface{}
	scope       TopicScope
}

// Compile-time interface compliance check
var _ Topic = (*TypedTopic)(nil)

// TopicConfig holds configuration for creating a new topic
type TopicConfig struct {
	Name        string                 `json:"name"`        // Unique identifier
	Module      string                 `json:"module"`      // Owning module (empty for framework topics)
	Scope       TopicScope             `json:"scope"`       // Framework or module scope
	Description string                 `json:"description"` // Human-readable description
	Pattern     string                 `json:"pattern"`     // Routing pattern
	Example     string                 `json:"example"`     // Usage example
	Metadata    map[string]interface{} `json:"metadata"`    // Additional data
}

// TopicScope defines whether a topic belongs to framework or module level
type TopicScope string

const (
	ScopeFramework TopicScope = "framework" // Core framework topics (presence, websocket, etc.)
	ScopeModule    TopicScope = "module"    // Module-specific topics (chat, wargame, etc.)
)

// RegistryEntry represents a topic entry in the registry with metadata
type RegistryEntry struct {
	Topic        Topic     `json:"topic"`
	RegisteredAt time.Time `json:"registered_at"`
	Module       string    `json:"module"`
	UsageCount   int64     `json:"usage_count"`
}

// TopicError represents structured errors in the topic management system
type TopicError struct {
	Type    ErrorType `json:"type"`
	Topic   string    `json:"topic"`
	Module  string    `json:"module"`
	Message string    `json:"message"`
	Cause   error     `json:"cause,omitempty"`
}

// ErrorType defines the type of topic management error
type ErrorType string

const (
	ErrorTopicNotFound         ErrorType = "topic_not_found"
	ErrorDuplicateRegistration ErrorType = "duplicate_registration"
	ErrorInvalidPattern        ErrorType = "invalid_pattern"
	ErrorValidationFailed      ErrorType = "validation_failed"
	ErrorInvalidScope          ErrorType = "invalid_scope"
)

// Error implements the error interface
func (e *TopicError) Error() string {
	if e.Cause != nil {
		return e.Message + ": " + e.Cause.Error()
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *TopicError) Unwrap() error {
	return e.Cause
}

// Implementation of Topic interface for TypedTopic

// Name returns the topic's unique identifier
func (t *TypedTopic) Name() string {
	return t.name
}

// Module returns the module that owns this topic
func (t *TypedTopic) Module() string {
	return t.module
}

// Description returns human-readable documentation
func (t *TypedTopic) Description() string {
	return t.description
}

// Pattern returns the routing pattern
func (t *TypedTopic) Pattern() string {
	return t.pattern
}

// Example returns a usage example
func (t *TypedTopic) Example() string {
	return t.example
}

// Metadata returns additional topic information
func (t *TypedTopic) Metadata() map[string]interface{} {
	if t.metadata == nil {
		return make(map[string]interface{})
	}
	// Return a copy to prevent external modification
	result := make(map[string]interface{})
	for k, v := range t.metadata {
		result[k] = v
	}
	return result
}

// Scope returns whether this is a framework or module topic
func (t *TypedTopic) Scope() TopicScope {
	return t.scope
}

// String returns the topic name for easy debugging
func (t *TypedTopic) String() string {
	return t.name
}