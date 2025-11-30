package events

// NewMessage represents a new chat message from a client.
type NewMessage struct {
	Content   string `json:"content"`
	User      string `json:"user"`
	Recipient string `json:"recipient,omitempty"`
}

// MessageHistory represents a request for chat message history.
type MessageHistory struct {
	UserID string `json:"userID"`
	Limit  int    `json:"limit"`
	Before string `json:"before,omitempty"`
}

// MessageDeleted represents a deleted chat message.
type MessageDeleted struct {
	MessageID string `json:"messageID"`
	DeletedBy string `json:"deletedBy"`
	Timestamp string `json:"timestamp"`
}

// UserTyping represents a typing indicator event.
type UserTyping struct {
	UserID    string `json:"userID"`
	IsTyping  bool   `json:"isTyping"`
	Timestamp string `json:"timestamp"`
}
