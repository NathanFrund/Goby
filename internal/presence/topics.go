package presence

import "github.com/nfrund/goby/internal/topicmgr"

// Framework topics for presence service
// These topics are used to track user online/offline status and presence events

var (
	// TopicUserOnline is published when a user comes online
	TopicUserOnline = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.user.online",
		Description: "Published when a user comes online",
		Pattern:     "presence.user.online",
		Example:     `{"userID":"user123","timestamp":"2024-01-01T00:00:00Z","sessionID":"session456"}`,
		Metadata: map[string]interface{}{
			"event_type": "presence_change",
			"status":     "online",
			"payload_fields": []string{"userID", "timestamp", "sessionID"},
		},
	})

	// TopicUserOffline is published when a user goes offline
	TopicUserOffline = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.user.offline",
		Description: "Published when a user goes offline",
		Pattern:     "presence.user.offline",
		Example:     `{"userID":"user123","timestamp":"2024-01-01T00:00:00Z","sessionID":"session456","reason":"timeout"}`,
		Metadata: map[string]interface{}{
			"event_type": "presence_change",
			"status":     "offline",
			"payload_fields": []string{"userID", "timestamp", "sessionID", "reason"},
		},
	})

	// TopicUserStatusUpdate is published when a user's status changes (away, busy, etc.)
	TopicUserStatusUpdate = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.user.status",
		Description: "Published when a user's presence status changes",
		Pattern:     "presence.user.status",
		Example:     `{"userID":"user123","status":"away","message":"In a meeting","timestamp":"2024-01-01T00:00:00Z"}`,
		Metadata: map[string]interface{}{
			"event_type": "status_change",
			"payload_fields": []string{"userID", "status", "message", "timestamp"},
			"valid_statuses": []string{"online", "away", "busy", "offline"},
		},
	})

	// TopicPresenceHeartbeat is used for internal heartbeat mechanism
	TopicPresenceHeartbeat = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.heartbeat",
		Description: "Internal heartbeat for presence tracking",
		Pattern:     "presence.heartbeat",
		Example:     `{"userID":"user123","timestamp":"2024-01-01T00:00:00Z","sessionID":"session456"}`,
		Metadata: map[string]interface{}{
			"event_type": "internal",
			"purpose":    "heartbeat",
			"payload_fields": []string{"userID", "timestamp", "sessionID"},
		},
	})

	// TopicPresenceQuery is used to query current presence status
	TopicPresenceQuery = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.query",
		Description: "Query current presence status for users",
		Pattern:     "presence.query",
		Example:     `{"requestID":"req123","userIDs":["user123","user456"],"requesterID":"admin"}`,
		Metadata: map[string]interface{}{
			"event_type": "query",
			"payload_fields": []string{"requestID", "userIDs", "requesterID"},
		},
	})

	// TopicPresenceResponse is the response to presence queries
	TopicPresenceResponse = topicmgr.DefineFramework(topicmgr.TopicConfig{
		Name:        "presence.response",
		Description: "Response containing presence status information",
		Pattern:     "presence.response",
		Example:     `{"requestID":"req123","users":[{"userID":"user123","status":"online","lastSeen":"2024-01-01T00:00:00Z"}]}`,
		Metadata: map[string]interface{}{
			"event_type": "response",
			"payload_fields": []string{"requestID", "users"},
		},
	})
)

// RegisterTopics registers all presence framework topics with the topic manager
func RegisterTopics() error {
	manager := topicmgr.Default()
	
	topics := []topicmgr.Topic{
		TopicUserOnline,
		TopicUserOffline,
		TopicUserStatusUpdate,
		TopicPresenceHeartbeat,
		TopicPresenceQuery,
		TopicPresenceResponse,
	}
	
	for _, topic := range topics {
		if err := manager.Register(topic); err != nil {
			return err
		}
	}
	
	return nil
}

// MustRegisterTopics registers all presence framework topics and panics on error
func MustRegisterTopics() {
	if err := RegisterTopics(); err != nil {
		panic("failed to register presence framework topics: " + err.Error())
	}
}