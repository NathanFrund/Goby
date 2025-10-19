package websocket

import (
	"fmt"
	"github.com/nfrund/goby/internal/topics"
)

// RegisterAll registers all WebSocket topics with the provided registry
// It skips topics that are already registered
func RegisterAll(reg *topics.TopicRegistry) error {
	// Register broadcast topics
	topicsToRegister := []topics.Topic{
		HTMLBroadcast,
		DataBroadcast,
		HTMLDirect,
		DataDirect,
	}

	for _, topic := range topicsToRegister {
		// Skip if already registered
		if _, exists := reg.Get(topic.Name()); exists {
			continue
		}
		if err := reg.Register(topic); err != nil {
			return fmt.Errorf("failed to register topic %s: %w", topic.Name(), err)
		}
	}

	return nil
}

// MustRegisterAll registers all WebSocket topics and panics on error
func MustRegisterAll(reg *topics.TopicRegistry) {
	if err := RegisterAll(reg); err != nil {
		panic("failed to register WebSocket topics: " + err.Error())
	}
}
