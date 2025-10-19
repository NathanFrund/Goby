package websocket

import (
	"fmt"
	"github.com/nfrund/goby/internal/topics"
)

// Broadcast topics for WebSocket communication
var (
	// HTMLBroadcast is the topic for broadcasting messages to all HTML WebSocket clients
	HTMLBroadcast = topics.NewBaseTopic(
		"ws.html.broadcast",
		"Broadcast message to all HTML WebSocket clients",
		"ws.html.broadcast",
		"ws.html.broadcast",
	)

	// DataBroadcast is the topic for broadcasting messages to all Data WebSocket clients
	DataBroadcast = topics.NewBaseTopic(
		"ws.data.broadcast",
		"Broadcast message to all Data WebSocket clients",
		"ws.data.broadcast",
		"ws.data.broadcast",
	)
)

// RegisterBroadcastTopics registers all WebSocket broadcast topics
// It skips topics that are already registered
func RegisterBroadcastTopics(reg *topics.TopicRegistry) error {
	topicsToRegister := []topics.Topic{
		HTMLBroadcast,
		DataBroadcast,
	}

	for _, topic := range topicsToRegister {
		// Skip if already registered
		if _, exists := reg.Get(topic.Name()); exists {
			continue
		}
		if err := reg.Register(topic); err != nil {
			return fmt.Errorf("failed to register broadcast topic %s: %w", topic.Name(), err)
		}
	}

	return nil
}
