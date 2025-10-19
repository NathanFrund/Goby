package topics

import (
	"fmt"
	"sync"
)

// TopicRegistry manages a collection of topics and provides methods to interact with them
type TopicRegistry struct {
	topics map[string]Topic
	mu     sync.RWMutex
}

// NewRegistry creates a new, empty topic registry
func NewRegistry() *TopicRegistry {
	return &TopicRegistry{
		topics: make(map[string]Topic),
	}
}

// Register adds a new topic to the registry
func (r *TopicRegistry) Register(topic Topic) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if topic == nil {
		return fmt.Errorf("cannot register nil topic")
	}

	name := topic.Name()
	if _, exists := r.topics[name]; exists {
		return fmt.Errorf("topic already registered: %s", name)
	}

	r.topics[name] = topic
	return nil
}

// MustRegister registers a topic and panics if registration fails
func (r *TopicRegistry) MustRegister(topic Topic) {
	if err := r.Register(topic); err != nil {
		panic(fmt.Sprintf("failed to register topic: %v", err))
	}
}

// Get returns a topic by name
func (r *TopicRegistry) Get(name string) (Topic, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topic, exists := r.topics[name]
	return topic, exists
}

// MustGet returns a topic by name and panics if not found
func (r *TopicRegistry) MustGet(name string) Topic {
	topic, exists := r.Get(name)
	if !exists {
		panic(fmt.Sprintf("topic not found: %s", name))
	}
	return topic
}

// List returns a copy of all registered topics
func (r *TopicRegistry) List() []Topic {
	r.mu.RLock()
	defer r.mu.RUnlock()

	topics := make([]Topic, 0, len(r.topics))
	for _, topic := range r.topics {
		topics = append(topics, topic)
	}
	return topics
}

// Count returns the number of registered topics
func (r *TopicRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.topics)
}

// Reset removes all registered topics
// Primarily for testing purposes
func (r *TopicRegistry) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.topics = make(map[string]Topic)
}

// DefaultRegistry is the global registry instance
var (
	defaultRegistry     *TopicRegistry
	defaultRegistryOnce sync.Once
)

// Default returns the default global registry
func Default() *TopicRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// Register registers a topic with the default registry
func Register(topic Topic) error {
	return Default().Register(topic)
}

// MustRegister registers a topic with the default registry and panics on error
func MustRegister(topic Topic) {
	Default().MustRegister(topic)
}

// Get retrieves a topic from the default registry
func Get(name string) (Topic, bool) {
	return Default().Get(name)
}

// MustGet retrieves a topic from the default registry and panics if not found
func MustGet(name string) Topic {
	return Default().MustGet(name)
}

// List returns all topics from the default registry
func List() []Topic {
	return Default().List()
}
