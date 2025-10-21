package topicmgr

import (
	"fmt"
	"sync"
	"time"
)

// Manager provides the main API for topic management with framework/module scoping
type Manager struct {
	registry  *Registry
	validator *Validator
	mu        sync.RWMutex
	started   bool
	startTime time.Time
}

// NewManager creates a new topic manager with registry and validator
func NewManager() *Manager {
	return &Manager{
		registry:  NewRegistry(),
		validator: NewValidator(),
		started:   false,
		startTime: time.Now(),
	}
}

// Start initializes the manager (for lifecycle management)
func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Message: "manager already started",
		}
	}

	m.started = true
	m.startTime = time.Now()
	return nil
}

// Stop shuts down the manager gracefully
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Message: "manager not started",
		}
	}

	m.started = false
	return nil
}

// IsStarted returns whether the manager is currently started
func (m *Manager) IsStarted() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.started
}

// GetStartTime returns when the manager was started
func (m *Manager) GetStartTime() time.Time {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.startTime
}

// DefineFramework creates a new typed topic for framework services
func DefineFramework(config TopicConfig) Topic {
	// Ensure scope is set to framework
	config.Scope = ScopeFramework
	config.Module = "" // Framework topics don't have a module

	return &TypedTopic{
		name:        config.Name,
		module:      config.Module,
		description: config.Description,
		pattern:     config.Pattern,
		example:     config.Example,
		metadata:    config.Metadata,
		scope:       config.Scope,
	}
}

// DefineModule creates a new typed topic for modules
func DefineModule(config TopicConfig) Topic {
	// Ensure scope is set to module
	config.Scope = ScopeModule

	return &TypedTopic{
		name:        config.Name,
		module:      config.Module,
		description: config.Description,
		pattern:     config.Pattern,
		example:     config.Example,
		metadata:    config.Metadata,
		scope:       config.Scope,
	}
}

// Register adds a topic to the central registry
func (m *Manager) Register(topic Topic) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate the topic before registration
	if err := m.validator.ValidateDefinition(topic); err != nil {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Topic:   topic.Name(),
			Module:  topic.Module(),
			Message: "topic validation failed",
			Cause:   err,
		}
	}

	return m.registry.Register(topic)
}

// Get retrieves a topic by name (for backward compatibility)
func (m *Manager) Get(name string) (Topic, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Get(name)
}

// List returns all registered topics
func (m *Manager) List() []Topic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.List()
}

// ListByModule returns topics for a specific module
func (m *Manager) ListByModule(module string) []Topic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.ListByModule(module)
}

// ListByScope returns topics for a specific scope (framework or module)
func (m *Manager) ListByScope(scope TopicScope) []Topic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.ListByScope(scope)
}

// ListFrameworkTopics returns all framework-level topics
func (m *Manager) ListFrameworkTopics() []Topic {
	return m.ListByScope(ScopeFramework)
}

// Validate checks if a topic usage is valid
func (m *Manager) Validate(topic Topic, context string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.validator.ValidateUsage(topic, context)
}

// ValidateTopicName checks if a topic name is valid without creating a topic
func (m *Manager) ValidateTopicName(name string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Use the validator's public name validation
	return m.validator.ValidateName(name)
}

// ValidateAndRegister validates and registers a topic in one operation
func (m *Manager) ValidateAndRegister(topic Topic) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// First validate
	if err := m.validator.ValidateDefinition(topic); err != nil {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Topic:   topic.Name(),
			Module:  topic.Module(),
			Message: "topic validation failed during registration",
			Cause:   err,
		}
	}

	// Then register
	return m.registry.Register(topic)
}

// ValidateConfiguration validates a topic configuration before creating a topic
func (m *Manager) ValidateConfiguration(config TopicConfig) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a temporary topic to validate the configuration
	var tempTopic Topic
	switch config.Scope {
	case ScopeFramework:
		tempTopic = DefineFramework(config)
	case ScopeModule:
		tempTopic = DefineModule(config)
	default:
		return &TopicError{
			Type:    ErrorInvalidScope,
			Topic:   config.Name,
			Module:  config.Module,
			Message: fmt.Sprintf("invalid scope: %s", config.Scope),
		}
	}

	return m.validator.ValidateDefinition(tempTopic)
}

// CheckTopicExists verifies if a topic is registered
func (m *Manager) CheckTopicExists(name string) bool {
	_, exists := m.Get(name)
	return exists
}

// ValidateTopicAccess checks if a topic can be accessed from a given context
func (m *Manager) ValidateTopicAccess(topicName, module, context string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	topic, exists := m.registry.Get(topicName)
	if !exists {
		return &TopicError{
			Type:    ErrorTopicNotFound,
			Topic:   topicName,
			Module:  module,
			Message: fmt.Sprintf("topic not found: %s", topicName),
		}
	}

	// Check scope-based access rules
	switch topic.Scope() {
	case ScopeFramework:
		// Framework topics can be accessed by anyone
		return nil
	case ScopeModule:
		// Module topics should typically only be accessed by their owning module
		// This is a soft validation - log a warning but don't fail
		if topic.Module() != module && module != "" {
			// In a real implementation, this might log a warning
			// For now, we'll allow cross-module access but could be configurable
		}
		return nil
	default:
		return &TopicError{
			Type:    ErrorValidationFailed,
			Topic:   topicName,
			Module:  module,
			Message: fmt.Sprintf("unknown topic scope: %s", topic.Scope()),
		}
	}
}

// Count returns the total number of registered topics
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry.Count()
}

// GetRegistry returns the underlying registry (for advanced usage)
func (m *Manager) GetRegistry() *Registry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.registry
}

// MustRegister registers a topic and panics on error (for static initialization)
func (m *Manager) MustRegister(topic Topic) {
	if err := m.Register(topic); err != nil {
		panic(fmt.Sprintf("failed to register topic %s: %v", topic.Name(), err))
	}
}

// GetStats returns comprehensive manager statistics
func (m *Manager) GetStats() ManagerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	registryStats := m.registry.GetStats()
	
	return ManagerStats{
		Started:         m.started,
		StartTime:       m.startTime,
		Uptime:          time.Since(m.startTime),
		RegistryStats:   registryStats,
		ValidatorActive: m.validator != nil,
	}
}

// ManagerStats provides comprehensive statistics about the manager
type ManagerStats struct {
	Started         bool          `json:"started"`
	StartTime       time.Time     `json:"start_time"`
	Uptime          time.Duration `json:"uptime"`
	RegistryStats   RegistryStats `json:"registry_stats"`
	ValidatorActive bool          `json:"validator_active"`
}

// Reset removes all registered topics (primarily for testing)
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.registry.Reset()
}

// Global manager instance
var (
	defaultManager     *Manager
	defaultManagerOnce sync.Once
)

// Default returns the default global manager
func Default() *Manager {
	defaultManagerOnce.Do(func() {
		defaultManager = NewManager()
	})
	return defaultManager
}

// Package-level convenience functions that use the default manager

// Register registers a topic with the default manager
func Register(topic Topic) error {
	return Default().Register(topic)
}

// Get retrieves a topic from the default manager
func Get(name string) (Topic, bool) {
	return Default().Get(name)
}

// List returns all topics from the default manager
func List() []Topic {
	return Default().List()
}

// ListByModule returns topics for a specific module from the default manager
func ListByModule(module string) []Topic {
	return Default().ListByModule(module)
}

// ListByScope returns topics for a specific scope from the default manager
func ListByScope(scope TopicScope) []Topic {
	return Default().ListByScope(scope)
}

// ListFrameworkTopics returns all framework topics from the default manager
func ListFrameworkTopics() []Topic {
	return Default().ListFrameworkTopics()
}

// Additional scoped discovery and filtering methods

// FindTopics searches for topics matching a pattern
func (m *Manager) FindTopics(pattern string) []Topic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []Topic
	allTopics := m.registry.List()
	
	for _, topic := range allTopics {
		// Simple pattern matching - could be enhanced with regex
		if matchesPattern(topic.Name(), pattern) {
			matches = append(matches, topic)
		}
	}
	
	return matches
}

// ListModules returns all unique module names that have registered topics
func (m *Manager) ListModules() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	moduleSet := make(map[string]bool)
	allTopics := m.registry.List()
	
	for _, topic := range allTopics {
		if topic.Scope() == ScopeModule && topic.Module() != "" {
			moduleSet[topic.Module()] = true
		}
	}
	
	modules := make([]string, 0, len(moduleSet))
	for module := range moduleSet {
		modules = append(modules, module)
	}
	
	return modules
}

// GetModuleTopicCount returns the number of topics for a specific module
func (m *Manager) GetModuleTopicCount(module string) int {
	topics := m.ListByModule(module)
	return len(topics)
}

// HasModule checks if a module has any registered topics
func (m *Manager) HasModule(module string) bool {
	return m.GetModuleTopicCount(module) > 0
}

// ListTopicsByPrefix returns topics whose names start with the given prefix
func (m *Manager) ListTopicsByPrefix(prefix string) []Topic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matches []Topic
	allTopics := m.registry.List()
	
	for _, topic := range allTopics {
		if len(topic.Name()) >= len(prefix) && topic.Name()[:len(prefix)] == prefix {
			matches = append(matches, topic)
		}
	}
	
	return matches
}

// matchesPattern performs simple pattern matching
// Supports wildcards (*) for basic pattern matching
func matchesPattern(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	// Simple wildcard matching - could be enhanced
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(name) >= len(prefix) && name[:len(prefix)] == prefix
	}
	
	return name == pattern
}

// Package-level convenience functions for scoped discovery

// FindTopics searches for topics matching a pattern using the default manager
func FindTopics(pattern string) []Topic {
	return Default().FindTopics(pattern)
}

// ListModules returns all module names using the default manager
func ListModules() []string {
	return Default().ListModules()
}

// GetModuleTopicCount returns topic count for a module using the default manager
func GetModuleTopicCount(module string) int {
	return Default().GetModuleTopicCount(module)
}

// HasModule checks if a module exists using the default manager
func HasModule(module string) bool {
	return Default().HasModule(module)
}

// ListTopicsByPrefix returns topics by prefix using the default manager
func ListTopicsByPrefix(prefix string) []Topic {
	return Default().ListTopicsByPrefix(prefix)
}

// Package-level convenience functions for validation

// Validate checks if a topic usage is valid using the default manager
func Validate(topic Topic, context string) error {
	return Default().Validate(topic, context)
}

// ValidateTopicName checks if a topic name is valid using the default manager
func ValidateTopicName(name string) error {
	return Default().ValidateTopicName(name)
}

// ValidateAndRegister validates and registers a topic using the default manager
func ValidateAndRegister(topic Topic) error {
	return Default().ValidateAndRegister(topic)
}

// ValidateConfiguration validates a topic configuration using the default manager
func ValidateConfiguration(config TopicConfig) error {
	return Default().ValidateConfiguration(config)
}

// CheckTopicExists verifies if a topic is registered using the default manager
func CheckTopicExists(name string) bool {
	return Default().CheckTopicExists(name)
}

// ValidateTopicAccess checks topic access permissions using the default manager
func ValidateTopicAccess(topicName, module, context string) error {
	return Default().ValidateTopicAccess(topicName, module, context)
}