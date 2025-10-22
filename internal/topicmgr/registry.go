package topicmgr

import (
	"fmt"
	"sync"
	"time"
)

// Registry manages the collection of registered topics with metadata
type Registry struct {
	entries map[string]*RegistryEntry
	mu      sync.RWMutex
}

// NewRegistry creates a new topic registry
func NewRegistry() *Registry {
	return &Registry{
		entries: make(map[string]*RegistryEntry),
	}
}

// Register adds a topic to the registry
func (r *Registry) Register(topic Topic) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if topic == nil {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Message: "cannot register nil topic",
		}
	}

	name := topic.Name()
	if name == "" {
		return &TopicError{
			Type:    ErrorValidationFailed,
			Topic:   name,
			Message: "topic name cannot be empty",
		}
	}

	// Check for duplicate registration
	if _, exists := r.entries[name]; exists {
		return &TopicError{
			Type:    ErrorDuplicateRegistration,
			Topic:   name,
			Module:  topic.Module(),
			Message: fmt.Sprintf("topic already registered: %s", name),
		}
	}

	// Create registry entry
	entry := &RegistryEntry{
		Topic:        topic,
		RegisteredAt: time.Now(),
		Module:       topic.Module(),
		UsageCount:   0,
	}

	r.entries[name] = entry
	return nil
}

// Get retrieves a topic by name
func (r *Registry) Get(name string) (Topic, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists {
		return nil, false
	}

	// Increment usage count
	go func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if entry, exists := r.entries[name]; exists {
			entry.UsageCount++
		}
	}()

	return entry.Topic, true
}

// List returns all registered topics
func (r *Registry) List() []Topic {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topics := make([]Topic, 0, len(r.entries))
	for _, entry := range r.entries {
		topics = append(topics, entry.Topic)
	}
	return topics
}

// ListByModule returns topics for a specific module
func (r *Registry) ListByModule(module string) []Topic {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var topics []Topic
	for _, entry := range r.entries {
		if entry.Topic.Module() == module {
			topics = append(topics, entry.Topic)
		}
	}
	return topics
}

// ListByScope returns topics for a specific scope
func (r *Registry) ListByScope(scope TopicScope) []Topic {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var topics []Topic
	for _, entry := range r.entries {
		if entry.Topic.Scope() == scope {
			topics = append(topics, entry.Topic)
		}
	}
	return topics
}

// GetEntry retrieves a registry entry by topic name
func (r *Registry) GetEntry(name string) (*RegistryEntry, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.entries[name]
	if !exists {
		return nil, false
	}

	// Return a copy to prevent external modification
	entryCopy := &RegistryEntry{
		Topic:        entry.Topic,
		RegisteredAt: entry.RegisteredAt,
		Module:       entry.Module,
		UsageCount:   entry.UsageCount,
	}

	return entryCopy, true
}

// Count returns the number of registered topics
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.entries)
}

// Reset removes all registered topics (primarily for testing)
func (r *Registry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries = make(map[string]*RegistryEntry)
}

// GetStats returns registry statistics
func (r *Registry) GetStats() RegistryStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := RegistryStats{
		TotalTopics:     len(r.entries),
		FrameworkTopics: 0,
		ModuleTopics:    0,
		ModuleBreakdown: make(map[string]int),
	}

	for _, entry := range r.entries {
		switch entry.Topic.Scope() {
		case ScopeFramework:
			stats.FrameworkTopics++
		case ScopeModule:
			stats.ModuleTopics++
			module := entry.Topic.Module()
			stats.ModuleBreakdown[module]++
		}
	}

	return stats
}

// RegistryStats provides statistics about the registry
type RegistryStats struct {
	TotalTopics     int            `json:"total_topics"`
	FrameworkTopics int            `json:"framework_topics"`
	ModuleTopics    int            `json:"module_topics"`
	ModuleBreakdown map[string]int `json:"module_breakdown"`
}