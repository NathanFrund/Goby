package websocket

import (
	"github.com/nfrund/goby/internal/topics"
)

// Client-related topics
var (
	// ClientReady is published when a new WebSocket client successfully connects and is ready
	ClientReady = topics.NewBaseTopic(
		"ws.client.ready",
		"A new WebSocket client has connected and is ready",
		"ws.client.ready",
		"ws.client.ready",
	)
)

// RegisterClientTopics registers all client-related topics
func RegisterClientTopics(reg *topics.TopicRegistry) error {
	if err := reg.Register(ClientReady); err != nil {
		return err
	}
	return nil
}
