package example

import (
	"github.com/nfrund/goby/internal/websocket"
)

const (
	// ChatMessageAction is the action for sending chat messages
	ChatMessageAction = "chat.message"
	// TypingStatusAction is the action for sending typing status updates
	TypingStatusAction = "chat.typing"
)

// RegisterChatActions registers the chat-related WebSocket actions
func RegisterChatActions(wsBridge *websocket.Bridge) {
	// Register allowed WebSocket actions for the chat module
	wsBridge.AllowAction(ChatMessageAction)
	wsBridge.AllowAction(TypingStatusAction)
}
