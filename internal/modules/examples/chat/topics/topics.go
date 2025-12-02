package topics

import (
	"github.com/nfrund/goby/internal/modules/examples/chat/events"
	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topicmgr"
)

// Module topics for the chat system
// These topics handle chat message routing and communication

var (
	// TopicNewMessage represents a new chat message from a client
	TopicNewMessage = pubsub.NewEvent[events.NewMessage]("client.chat.message.new", "A new chat message sent by a client")

	// TopicMessages represents broadcast messages to all clients
	// Note: This is for rendered HTML, not typed data
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
	// Note: This is for rendered HTML, not typed data
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
	TopicMessageHistory = pubsub.NewEvent[events.MessageHistory]("chat.history.request", "Request for chat message history")

	// TopicMessageDeleted represents deleted messages
	TopicMessageDeleted = pubsub.NewEvent[events.MessageDeleted]("chat.message.deleted", "Notification that a chat message was deleted")

	// TopicUserTyping represents typing indicators
	TopicUserTyping = pubsub.NewEvent[events.UserTyping]("chat.user.typing", "Indicates that a user is typing")
)

// RegisterTopics is now a no-op because NewEvent auto-registers topics.
// Kept for backward compatibility with module interface.
func RegisterTopics() error {
	// Manually register the non-typed topics (Messages and DirectMessage)
	manager := topicmgr.Default()
	if err := manager.Register(TopicMessages); err != nil {
		return err
	}
	if err := manager.Register(TopicDirectMessage); err != nil {
		return err
	}
	return nil
}

// MustRegisterTopics registers topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register chat module topics: " + err.Error())
	}
}
