// Package chat demonstrates how to use the WebSocket bridge with a chat module
package chat

import (
	"github.com/nfrund/goby/internal/websocket"
)

// Action constants define the WebSocket actions that this module handles
const (
	// MessageAction is the action for sending chat messages
	MessageAction = "chat.message"
	// TypingAction is the action for sending typing status updates
	TypingAction = "chat.typing"
	// JoinRoomAction is the action for joining a chat room
	JoinRoomAction = "chat.join_room"
)

// RegisterChatActions registers the chat-related WebSocket actions with the bridge.
// This should be called during module initialization.
//
// Example:
//
//	func (m *ChatModule) Register(sl registry.ServiceLocator, cfg config.Provider) error {
//		wsBridge := sl.Get(registry.WebSocketBridgeKey).(*websocket.Bridge)
//		chat.RegisterChatActions(wsBridge)
//		// ... rest of registration
//		return nil
//	}
func RegisterChatActions(bridge *websocket.Bridge) {
	// Register allowed WebSocket actions for the chat module
	bridge.AllowAction(MessageAction)
	bridge.AllowAction(TypingAction)
	bridge.AllowAction(JoinRoomAction)
}

// Message represents a chat message
// This is an example of how to structure message payloads
// type Message struct {
// 	ID        string `json:"id"`
// 	UserID    string `json:"user_id"`
// 	RoomID    string `json:"room_id"`
// 	Content   string `json:"content"`
// 	Timestamp int64  `json:"timestamp"`
// }
