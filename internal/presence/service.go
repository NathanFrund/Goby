package presence

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topicmgr"
)

type Status string

const (
	StatusOnline  Status = "online"
	StatusOffline Status = "offline"
)

type Presence struct {
	UserID    string    `json:"user_id"`
	Status    Status    `json:"status"`
	ClientID  string    `json:"client_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"user_agent,omitempty"`
}

type Service struct {
	mu        sync.RWMutex
	presences map[string]Presence // userID -> Presence
	publisher pubsub.Publisher
	logger    *slog.Logger
}

// Now returns the current time in UTC
func Now() time.Time {
	return time.Now().UTC()
}

// NewService creates a new presence service with the provided dependencies.
func NewService(publisher pubsub.Publisher, subscriber pubsub.Subscriber, topicMgr *topicmgr.Manager) *Service {
	svc := &Service{
		presences: make(map[string]Presence),
		publisher: publisher,
		logger:    slog.Default().With("service", "presence"),
	}

	// Register presence framework topics
	if err := RegisterTopics(); err != nil {
		svc.logger.Error("failed to register presence topics", "error", err)
	}

	// Subscribe to WebSocket client events using the new typed topics
	ctx := context.Background()
	if err := subscriber.Subscribe(ctx, TopicUserOnline.Name(), svc.handleClientConnected); err != nil {
		svc.logger.Error("failed to subscribe to user online events", "error", err)
	}
	if err := subscriber.Subscribe(ctx, TopicUserOffline.Name(), svc.handleClientDisconnected); err != nil {
		svc.logger.Error("failed to subscribe to user offline events", "error", err)
	}

	svc.logger.Info("Presence service initialized")
	return svc
}

func (s *Service) handleClientConnected(ctx context.Context, msg pubsub.Message) error {
	s.logger.Debug("Received client connected event",
		"message", string(msg.Payload),
		"topic", msg.Topic,
	)

	var event struct {
		UserID    string `json:"user_id"`
		ClientID  string `json:"client_id"`
		UserAgent string `json:"user_agent"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	s.addPresence(event.UserID, event.ClientID, event.UserAgent)

	return nil
}

func (s *Service) addPresence(userID, clientID, userAgent string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if user already has a presence
	if existing, exists := s.presences[userID]; exists {
		s.logger.Debug("Updating existing presence",
			"user_id", userID,
			"client_id", clientID,
			"previous_client_id", existing.ClientID,
			"user_agent", userAgent)
	} else {
		s.logger.Info("User came online",
			"user_id", userID,
			"client_id", clientID,
			"user_agent", userAgent)
	}

	s.presences[userID] = Presence{
		UserID:    userID,
		Status:    StatusOnline,
		ClientID:  clientID,
		Timestamp: Now(),
		UserAgent: userAgent,
	}

	s.logger.Debug("Current online users",
		"count", len(s.presences),
		"users", s.GetOnlineUsers())

	s.publishPresenceUpdate()
}

func (s *Service) handleClientDisconnected(ctx context.Context, msg pubsub.Message) error {
	s.logger.Debug("Received client disconnected event",
		"message", string(msg.Payload),
		"topic", msg.Topic,
	)

	var event struct {
		ClientID string `json:"client_id"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		return err
	}

	s.removePresence(event.ClientID)

	return nil
}

func (s *Service) removePresence(clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	presence, exists := s.presences[clientID]
	if !exists {
		s.logger.Debug("Client not found in presence list", "client_id", clientID)
		return
	}

	delete(s.presences, clientID)
	s.logger.Info("User disconnected",
		"user_id", presence.UserID,
		"client_id", clientID,
		"remaining_users", len(s.presences))

	s.publishPresenceUpdate()
}

// GetPresence returns the current presence status for a user
func (s *Service) GetPresence(userID string) (Presence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	presence, exists := s.presences[userID]
	return presence, exists
}

// GetOnlineUsers returns a list of currently online user IDs
func (s *Service) GetOnlineUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make(map[string]struct{})
	for _, p := range s.presences {
		users[p.UserID] = struct{}{}
	}

	result := make([]string, 0, len(users))
	for userID := range users {
		result = append(result, userID)
	}

	s.logger.Debug("Retrieved online users",
		"unique_users", len(result),
		"total_connections", len(s.presences))

	return result
}

// publishPresenceUpdate publishes the current list of online users
func (s *Service) publishPresenceUpdate() {
	payload := s.getCurrentPresence()

	s.logger.Debug("Publishing presence update",
		"topic", TopicUserStatusUpdate.Name(),
		"user_count", len(s.presences),
		"payload_size", len(payload))

	msg := pubsub.Message{
		Topic:   TopicUserStatusUpdate.Name(),
		Payload: payload,
	}
	err := s.publisher.Publish(context.Background(), msg)
	if err != nil {
		s.logger.Error("Failed to publish presence update",
			"error", err,
			"topic", TopicUserStatusUpdate.Name())
	}
}

func (s *Service) getCurrentPresence() []byte {
	onlineUsers := s.GetOnlineUsers()

	// Create a map for the update message
	update := struct {
		Type  string   `json:"type"`
		Users []string `json:"users"`
	}{
		Type:  "presence_update",
		Users: onlineUsers,
	}

	// Marshal the update to JSON
	payload, err := json.Marshal(update)
	if err != nil {
		s.logger.Error("Failed to marshal presence update", "error", err)
		return nil
	}

	return payload
}

// SubscribeToPresence subscribes to presence updates for all users
func (s *Service) SubscribeToPresence(ctx context.Context, handler func(Presence) error, subscriber pubsub.Subscriber) error {
	return subscriber.Subscribe(ctx, TopicUserStatusUpdate.Name(), func(ctx context.Context, msg pubsub.Message) error {
		var presence Presence
		if err := json.Unmarshal(msg.Payload, &presence); err != nil {
			s.logger.Error("Failed to unmarshal presence update", "error", err)
			return nil
		}
		return handler(presence)
	})
}
