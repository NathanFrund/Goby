package topics

import (
	"github.com/nfrund/goby/internal/modules/announcer/events"
	"github.com/nfrund/goby/internal/pubsub"
)

// Module topics for the announcer system
// These topics handle system events that other modules can subscribe to

var (
	// TopicUserCreated represents a user account creation event
	TopicUserCreated = pubsub.NewEvent[events.UserCreated]("announcer.user.created", "A user account has been created (published by announcer module)")

	// TopicUserDeleted represents a user account deletion event
	TopicUserDeleted = pubsub.NewEvent[events.UserDeleted]("announcer.user.deleted", "A user account has been deleted (published by announcer module)")
)

// RegisterTopics is now a no-op because NewEvent auto-registers topics.
// Kept for backward compatibility with module interface.
func RegisterTopics() error {
	return nil
}

// MustRegisterTopics is now a no-op.
func MustRegisterTopics() {
}
