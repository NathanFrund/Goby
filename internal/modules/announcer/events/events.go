package events

// UserCreated represents a user account creation event.
type UserCreated struct {
	UserID    string `json:"userID"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
}

// UserDeleted represents a user account deletion event.
type UserDeleted struct {
	UserID    string `json:"userID"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Timestamp string `json:"timestamp"`
}
