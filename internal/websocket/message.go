package websocket

import "encoding/json"

// Message represents a generic WebSocket message
type Message struct {
	Type    string      `json:"type"` // Message type (e.g., "html", "data", "command")
	Target  string      `json:"target,omitempty"`
	Payload interface{} `json:"payload"` // The actual message content (string or []byte)
}

// MarshalJSON customizes JSON marshaling to handle both string and []byte payloads
func (m Message) MarshalJSON() ([]byte, error) {
	// Create a type that doesn't have the custom marshal method to avoid recursion
	type Alias Message
	msg := struct {
		*Alias
		Payload interface{} `json:"payload"`
	}{
		Alias: (*Alias)(&m),
	}

	// Convert []byte to string for proper JSON encoding
	if b, ok := m.Payload.([]byte); ok {
		msg.Payload = string(b)
	} else {
		msg.Payload = m.Payload
	}

	return json.Marshal(msg)
}

// Command represents a command that can be sent to clients
type Command struct {
	Name    string      `json:"name"`              // Command name (e.g., "reload", "navigate")
	Payload interface{} `json:"payload,omitempty"` // Optional command data
}

// NewMessage creates a new message with the given type and payload
func NewMessage(msgType string, payload interface{}, target ...string) *Message {
	msg := &Message{
		Type:    msgType,
		Payload: payload,
	}
	if len(target) > 0 {
		msg.Target = target[0]
	}
	return msg
}

// NewHTMLMessage creates a new HTML message
func NewHTMLMessage(html string, target string) *Message {
	return NewMessage("html", html, target)
}

// NewDataMessage creates a new data message
func NewDataMessage(data interface{}) *Message {
	return NewMessage("data", data)
}

// NewCommand creates a new command message
func NewCommand(name string, payload ...interface{}) *Message {
	var p interface{} = nil
	if len(payload) > 0 {
		p = payload[0]
	}
	return NewMessage("command", Command{
		Name:    name,
		Payload: p,
	})
}

// Common command names
const (
	CmdReload           = "reload"
	CmdReconnect        = "reconnect"
	CmdNavigate         = "navigate"
	CmdShowNotification = "show_notification"
	CmdUpdateTitle      = "update_title"
	CmdAuthRequired     = "auth_required"
	CmdSessionExpired   = "session_expired"
)
