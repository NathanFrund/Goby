package websocket

import (
	"fmt"

	"github.com/nfrund/goby/internal/topics"
)

const (
	// TopicHTMLBroadcast is the topic for broadcasting messages to all HTML clients
	TopicHTMLBroadcast = "ws.html.broadcast"
	// TopicHTMLDirect is the topic prefix for direct messages to HTML clients
	TopicHTMLDirect = "ws.html.direct"
	// TopicDataBroadcast is the topic for broadcasting messages to all Data clients
	TopicDataBroadcast = "ws.data.broadcast"
	// TopicDataDirect is the topic prefix for direct messages to Data clients
	TopicDataDirect = "ws.data.direct"
)

// RegisterTopics registers all WebSocket-related topics with the provided registry.
// This should be called during application startup.
func RegisterTopics(reg interface{
	Register(topic topics.Topic) error
}) error {
	topicDefs := []struct {
		name        string
		description string
		routingKey  string
		topicName   string
	}{
		{
			name:        TopicHTMLBroadcast,
			description: "Broadcast message to all HTML WebSocket clients",
			topicName:   TopicHTMLBroadcast,
			routingKey:  TopicHTMLBroadcast,
		},
		{
			name:        TopicHTMLDirect,
			description: "Direct message to a specific HTML WebSocket client",
			topicName:   TopicHTMLDirect,
			routingKey:  TopicHTMLDirect, // No wildcard needed, using metadata for routing
		},
		{
			name:        TopicDataBroadcast,
			description: "Broadcast message to all Data WebSocket clients",
			topicName:   TopicDataBroadcast,
			routingKey:  TopicDataBroadcast,
		},
		{
			name:        TopicDataDirect,
			description: "Direct message to a specific Data WebSocket client",
			topicName:   TopicDataDirect,
			routingKey:  TopicDataDirect, // No wildcard needed, using metadata for routing
		},
	}

	for _, def := range topicDefs {
		topic := topics.NewBaseTopic(
			def.name,
			def.description,
			def.routingKey,
			def.topicName,
		)
		if err := reg.Register(topic); err != nil {
			return fmt.Errorf("failed to register topic %s: %w", def.name, err)
		}
	}

	return nil
}
