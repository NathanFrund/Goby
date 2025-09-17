package chat

import "time"

// Message defines the structure for a chat message.
// This is the specific event type that our chat components will publish and
// consume via the generic Hub.
type Message struct {
	UserID   string    `json:"userId"`
	Username string    `json:"username"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sentAt"`
}
