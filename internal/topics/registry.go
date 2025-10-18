package topics

import "sync"

var (
	registry     = make(map[string]Topic)
	registryLock sync.RWMutex
)

// Register registers a new topic
func Register(topic Topic) {
	registryLock.Lock()
	defer registryLock.Unlock()
	
	if _, exists := registry[topic.Name]; exists {
		panic("topic already registered: " + topic.Name)
	}
	registry[topic.Name] = topic
}

// Get returns a topic by name
func Get(name string) (Topic, bool) {
	registryLock.RLock()
	defer registryLock.RUnlock()
	
	topic, exists := registry[name]
	return topic, exists
}

// List returns all registered topics
func List() []Topic {
	registryLock.RLock()
	defer registryLock.RUnlock()
	
	result := make([]Topic, 0, len(registry))
	for _, topic := range registry {
		result = append(result, topic)
	}
	return result
}

// ResetRegistryForTesting clears all registered topics.
// This should only be used in tests.
func ResetRegistryForTesting() {
	registryLock.Lock()
	defer registryLock.Unlock()
	
	// Clear the registry
	registry = make(map[string]Topic)
}
