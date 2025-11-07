package topics

import "github.com/nfrund/goby/internal/topicmgr"

// Module topics for the announcer system
// These topics handle system events that other modules can subscribe to

var (
	// TopicUserCreated represents a user account creation event
	TopicUserCreated = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "announcer.user.created",
		Module:      "announcer",
		Description: "A user account has been created (published by announcer module)",
		Pattern:     "announcer.user.created",
		Example:     `{"userID":"user123","email":"user@example.com","name":"John Doe","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type":     "user_created",
			"payload_fields": []string{"userID", "email", "name", "timestamp"},
		},
	})

	// TopicUserDeleted represents a user account deletion event
	TopicUserDeleted = topicmgr.DefineModule(topicmgr.TopicConfig{
		Name:        "announcer.user.deleted",
		Module:      "announcer",
		Description: "A user account has been deleted (published by announcer module)",
		Pattern:     "announcer.user.deleted",
		Example:     `{"userID":"user123","email":"user@example.com","name":"John Doe","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type":     "user_deleted",
			"payload_fields": []string{"userID", "email", "name", "timestamp"},
		},
	})
)

// RegisterTopics registers all announcer module topics with the topic manager
func RegisterTopics() error {
	manager := topicmgr.Default()

	topics := []topicmgr.Topic{
		TopicUserCreated,
		TopicUserDeleted,
	}

	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			return err
		}
	}

	return nil
}

// MustRegisterTopics registers all announcer module topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register announcer module topics: " + err.Error())
	}
}
