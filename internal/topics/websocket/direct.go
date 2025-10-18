package websocket

import (
	"fmt"
	"github.com/nfrund/goby/internal/topics"
)

// Direct message topics for WebSocket communication
var (
	// HTMLDirect is the topic for sending direct messages to specific HTML WebSocket clients
	// The recipient ID should be specified in the message metadata as "recipient_id"
	HTMLDirect = topics.NewBaseTopic(
		"ws.html.direct",
		"Direct message to a specific HTML WebSocket client",
		"ws.html.direct",
		"ws.html.direct",
	)

	// DataDirect is the topic for sending direct messages to specific Data WebSocket clients
	// The recipient ID should be specified in the message metadata as "recipient_id"
	DataDirect = topics.NewBaseTopic(
		"ws.data.direct",
		"Direct message to a specific Data WebSocket client",
		"ws.data.direct",
		"ws.data.direct",
	)
)

// RegisterDirectMessageTopics registers all WebSocket direct message topics
// It skips topics that are already registered
func RegisterDirectMessageTopics(reg *topics.TopicRegistry) error {
	topicsToRegister := []topics.Topic{
		HTMLDirect,
		DataDirect,
	}

	for _, topic := range topicsToRegister {
		// Skip if already registered
		if _, exists := reg.Get(topic.Name()); exists {
			continue
		}
		if err := reg.Register(topic); err != nil {
			return fmt.Errorf("failed to register direct message topic %s: %w", topic.Name(), err)
		}
	}

	return nil
}
