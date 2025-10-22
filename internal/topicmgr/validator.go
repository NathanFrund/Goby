package topicmgr

import (
	"fmt"
	"regexp"
	"strings"
)

// Validator provides validation for topic definitions and usage
type Validator struct {
	// namePattern defines valid topic name patterns
	namePattern *regexp.Regexp
}

// NewValidator creates a new topic validator
func NewValidator() *Validator {
	// Topic names should follow a hierarchical pattern: scope.module.action
	// Examples: ws.html.broadcast, chat.messages, presence.user.online
	namePattern := regexp.MustCompile(`^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$`)

	return &Validator{
		namePattern: namePattern,
	}
}

// ValidateDefinition validates a topic definition
func (v *Validator) ValidateDefinition(topic Topic) error {
	if topic == nil {
		return fmt.Errorf("topic cannot be nil")
	}

	// Validate name
	if err := v.validateName(topic.Name()); err != nil {
		return fmt.Errorf("invalid topic name: %w", err)
	}

	// Validate description
	if strings.TrimSpace(topic.Description()) == "" {
		return fmt.Errorf("topic description cannot be empty")
	}

	// Validate pattern
	if strings.TrimSpace(topic.Pattern()) == "" {
		return fmt.Errorf("topic pattern cannot be empty")
	}

	// Validate scope-specific rules
	switch topic.Scope() {
	case ScopeFramework:
		if err := v.validateFrameworkTopic(topic); err != nil {
			return fmt.Errorf("framework topic validation failed: %w", err)
		}
	case ScopeModule:
		if err := v.validateModuleTopic(topic); err != nil {
			return fmt.Errorf("module topic validation failed: %w", err)
		}
	default:
		return fmt.Errorf("invalid topic scope: %s", topic.Scope())
	}

	return nil
}

// ValidateUsage validates topic usage in a specific context
func (v *Validator) ValidateUsage(topic Topic, context string) error {
	if topic == nil {
		return fmt.Errorf("topic cannot be nil")
	}

	// Basic validation
	if err := v.ValidateDefinition(topic); err != nil {
		return fmt.Errorf("topic definition is invalid: %w", err)
	}

	// Context-specific validation could be added here
	// For example, checking if the topic is appropriate for the given context
	
	return nil
}

// ValidateName checks if a topic name follows the naming convention (public method)
func (v *Validator) ValidateName(name string) error {
	return v.validateName(name)
}

// validateName checks if a topic name follows the naming convention
func (v *Validator) validateName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("name too long (max 100 characters)")
	}

	if !v.namePattern.MatchString(name) {
		return fmt.Errorf("name must follow pattern: scope.module.action (lowercase, alphanumeric, dots only)")
	}

	// Check for reserved prefixes
	reservedPrefixes := []string{"system.", "internal.", "debug."}
	for _, prefix := range reservedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return fmt.Errorf("name cannot start with reserved prefix: %s", prefix)
		}
	}

	return nil
}

// validateFrameworkTopic validates framework-specific topic rules
func (v *Validator) validateFrameworkTopic(topic Topic) error {
	// Framework topics should not have a module
	if topic.Module() != "" {
		return fmt.Errorf("framework topics should not have a module")
	}

	// Framework topics should follow specific naming patterns
	name := topic.Name()
	validFrameworkPrefixes := []string{
		"ws.",        // WebSocket topics
		"presence.",  // Presence service topics
		"auth.",      // Authentication topics
		"server.",    // Server lifecycle topics
	}

	hasValidPrefix := false
	for _, prefix := range validFrameworkPrefixes {
		if strings.HasPrefix(name, prefix) {
			hasValidPrefix = true
			break
		}
	}

	if !hasValidPrefix {
		return fmt.Errorf("framework topic must start with a valid prefix: %v", validFrameworkPrefixes)
	}

	return nil
}

// validateModuleTopic validates module-specific topic rules
func (v *Validator) validateModuleTopic(topic Topic) error {
	// Module topics must have a module
	if strings.TrimSpace(topic.Module()) == "" {
		return fmt.Errorf("module topics must specify a module")
	}

	// Validate module name
	if err := v.validateModuleName(topic.Module()); err != nil {
		return fmt.Errorf("invalid module name: %w", err)
	}

	// Module topics should typically include the module name in the topic name
	name := topic.Name()
	module := topic.Module()
	
	// Check if the topic name contains the module name
	if !strings.Contains(name, module) {
		// This is a warning, not an error - some topics might have different naming schemes
		// We could log this as a warning in a real implementation
	}

	return nil
}

// validateModuleName validates a module name
func (v *Validator) validateModuleName(module string) error {
	if module == "" {
		return fmt.Errorf("module name cannot be empty")
	}

	if len(module) > 50 {
		return fmt.Errorf("module name too long (max 50 characters)")
	}

	// Module names should be simple identifiers
	modulePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	if !modulePattern.MatchString(module) {
		return fmt.Errorf("module name must be lowercase alphanumeric with underscores")
	}

	return nil
}