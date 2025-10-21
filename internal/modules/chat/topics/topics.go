package topics

import "github.com/nfrund/goby/internal/topicmgr"

// Module topics for the chat system
// These topics handle chat message routing and communication

var (
	// TopicNewMessage represents a new chat message from a client
	TopicNewMessage = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "client.chat.message.new",
		Module:      "chat",
		Description: "A new chat message sent by a client",
		Pattern:     "client.chat.message.new",
		Example:     `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
		Metadata: map[string]interface{}{
			"source":       "client",
			"message_type": "new",
			"payload_fields": []string{"content", "user"},
		},
	})

	// TopicMessages represents broadcast messages to all clients
	TopicMessages = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "chat.messages",
		Module:      "chat",
		Description: "Broadcasts a rendered chat message to all clients",
		Pattern:     "chat.messages",
		Example:     "chat.messages",
		Metadata: map[string]interface{}{
			"routing_type": "broadcast",
			"content_type": "rendered_html",
		},
	})

	// TopicDirectMessage represents direct messages to specific users
	TopicDirectMessage = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "chat.direct",
		Module:      "chat",
		Description: "Sends a rendered direct message to a specific user",
		Pattern:     "chat.direct.{userID}",
		Example:     "chat.direct.user123",
		Metadata: map[string]interface{}{
			"routing_type": "direct",
			"content_type": "rendered_html",
			"requires":     []string{"recipient_id"},
		},
	})

	// TopicMessageHistory represents requests for chat history
	TopicMessageHistory = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "chat.history.request",
		Module:      "chat",
		Description: "Request for chat message history",
		Pattern:     "chat.history.request",
		Example:     `{"userID":"user123","limit":50,"before":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"request_type": "history",
			"payload_fields": []string{"userID", "limit", "before"},
		},
	})

	// TopicMessageDeleted represents deleted messages
	TopicMessageDeleted = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "chat.message.deleted",
		Module:      "chat",
		Description: "Notification that a chat message was deleted",
		Pattern:     "chat.message.deleted",
		Example:     `{"messageID":"msg123","deletedBy":"user456","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "deletion",
			"payload_fields": []string{"messageID", "deletedBy", "timestamp"},
		},
	})

	// TopicUserTyping represents typing indicators
	TopicUserTyping = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "chat.user.typing",
		Module:      "chat",
		Description: "Indicates that a user is typing",
		Pattern:     "chat.user.typing",
		Example:     `{"userID":"user123","isTyping":true,"timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "typing_indicator",
			"payload_fields": []string{"userID", "isTyping", "timestamp"},
		},
	})
)

// RegisterTopics registers all chat module topics with the topic manager
func RegisterTopics() error {
	manager := topicmgr.Default()
	
	topics := []topicmgr.Topic{
		TopicNewMessage,
		TopicMessages,
		TopicDirectMessage,
		TopicMessageHistory,
		TopicMessageDeleted,
		TopicUserTyping,
	}
	
	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			return err
		}
	}
	
	return nil
}

// MustRegisterTopics registers all chat module topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register chat module topics: " + err.Error())
	}
}