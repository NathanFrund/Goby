package websocket

import (
	"strings"

	"github.com/nfrund/goby/internal/topicmgr"
)

// Framework topics for WebSocket communication
// These topics are used by the WebSocket bridge for routing messages

var (
	// TopicHTMLBroadcast broadcasts HTML content to all connected HTML WebSocket clients
	TopicHTMLBroadcast = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.html.broadcast",
		Description: "Broadcast HTML content to all connected HTML WebSocket clients",
		Pattern:     "ws.html.broadcast",
		Example:     "ws.html.broadcast",
		Metadata: map[string]interface{}{
			"endpoint_type": "html",
			"routing_type":  "broadcast",
		},
	})

	// TopicHTMLDirect sends HTML content to a specific HTML WebSocket client
	// The recipient ID should be specified in the message metadata as "recipient_id"
	TopicHTMLDirect = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.html.direct",
		Description: "Send HTML content to a specific HTML WebSocket client",
		Pattern:     "ws.html.direct",
		Example:     "ws.html.direct",
		Metadata: map[string]interface{}{
			"endpoint_type": "html",
			"routing_type":  "direct",
			"requires":      []string{"recipient_id"},
		},
	})

	// TopicDataBroadcast broadcasts JSON data to all connected Data WebSocket clients
	TopicDataBroadcast = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.data.broadcast",
		Description: "Broadcast JSON data to all connected Data WebSocket clients",
		Pattern:     "ws.data.broadcast",
		Example:     "ws.data.broadcast",
		Metadata: map[string]interface{}{
			"endpoint_type": "data",
			"routing_type":  "broadcast",
		},
	})

	// TopicDataDirect sends JSON data to a specific Data WebSocket client
	// The recipient ID should be specified in the message metadata as "recipient_id"
	TopicDataDirect = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.data.direct",
		Description: "Send JSON data to a specific Data WebSocket client",
		Pattern:     "ws.data.direct",
		Example:     "ws.data.direct",
		Metadata: map[string]interface{}{
			"endpoint_type": "data",
			"routing_type":  "direct",
			"requires":      []string{"recipient_id"},
		},
	})

	// TopicClientReady is published when a new WebSocket client successfully connects and is ready
	TopicClientReady = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.client.ready",
		Description: "Published when a new WebSocket client successfully connects and is ready",
		Pattern:     "ws.client.ready",
		Example:     `{"endpoint":"html","userID":"user123","connectionID":"conn456"}`,
		Metadata: map[string]interface{}{
			"event_type":     "lifecycle",
			"payload_fields": []string{"endpoint", "userID", "connectionID"},
		},
	})

	// TopicClientDisconnected is published when a WebSocket client disconnects
	TopicClientDisconnected = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "ws.client.disconnected",
		Description: "Published when a WebSocket client disconnects",
		Pattern:     "ws.client.disconnected",
		Example:     `{"endpoint":"html","userID":"user123","connectionID":"conn456","reason":"client_closed"}`,
		Metadata: map[string]interface{}{
			"event_type":     "lifecycle",
			"payload_fields": []string{"endpoint", "userID", "connectionID", "reason"},
		},
	})
)

// RegisterTopics registers all WebSocket framework topics with the default topic manager
// This function is idempotent - it will not fail if topics are already registered
func RegisterTopics() error {
	return RegisterTopicsWithManager(topicmgr.Default())
}

// RegisterTopicsWithManager registers all WebSocket framework topics with the specified topic manager
// This function is idempotent - it will not fail if topics are already registered
func RegisterTopicsWithManager(manager *topicmgr.Manager) error {
	topics := []topicmgr.Topic{
		TopicHTMLBroadcast,
		TopicHTMLDirect,
		TopicDataBroadcast,
		TopicDataDirect,
		TopicClientReady,
		TopicClientDisconnected,
	}

	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			// Check if this is a "already registered" error, which we can ignore
			if strings.Contains(err.Error(), "already registered") {
				continue // Skip already registered topics
			}
			return err // Return other errors
		}
	}

	return nil
}

// MustRegisterTopics registers all WebSocket framework topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register WebSocket framework topics: " + err.Error())
	}
}
