package presence

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/nfrund/goby/internal/pubsub"
	"github.com/nfrund/goby/internal/topicmgr"
	wsTopics "github.com/nfrund/goby/internal/websocket"
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
	clients   map[string]string   // clientID -> userID (for disconnect lookup)
	publisher pubsub.Publisher
	logger    *slog.Logger
	
	// Rate limiting
	rateLimiter map[string]*time.Timer // userID -> last update timer
	rateMu      sync.Mutex
	
	// Cleanup mechanism
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// Now returns the current time in UTC
func Now() time.Time {
	return time.Now().UTC()
}

// checkRateLimit prevents too frequent presence updates from the same user
func (s *Service) checkRateLimit(userID string) bool {
	const rateLimitWindow = 1 * time.Second // Max 1 update per second per user
	
	s.rateMu.Lock()
	defer s.rateMu.Unlock()
	
	if timer, exists := s.rateLimiter[userID]; exists {
		// Check if rate limit window has passed
		select {
		case <-timer.C:
			// Timer expired, allow update
			delete(s.rateLimiter, userID)
			s.rateLimiter[userID] = time.NewTimer(rateLimitWindow)
			return true
		default:
			// Still within rate limit window
			return false
		}
	}
	
	// First update for this user
	s.rateLimiter[userID] = time.NewTimer(rateLimitWindow)
	return true
}

// NewService creates a new presence service with the provided dependencies.
func NewService(publisher pubsub.Publisher, subscriber pubsub.Subscriber, topicMgr *topicmgr.Manager) *Service {
	svc := &Service{
		presences:     make(map[string]Presence),
		clients:       make(map[string]string),
		publisher:     publisher,
		logger:        slog.Default().With("service", "presence"),
		rateLimiter:   make(map[string]*time.Timer),
		cleanupTicker: time.NewTicker(30 * time.Second), // Cleanup every 30 seconds
		stopCleanup:   make(chan struct{}),
	}

	// Register presence framework topics
	if err := RegisterTopics(); err != nil {
		svc.logger.Error("failed to register presence topics", "error", err)
	}

	// Subscribe to WebSocket client lifecycle events
	ctx := context.Background()

	svc.logger.Info("Subscribing to WebSocket events",
		"ready_topic", wsTopics.TopicClientReady.Name(),
		"disconnect_topic", wsTopics.TopicClientDisconnected.Name())

	// Debug: Let's also check what the bridge is publishing to
	svc.logger.Info("WebSocket bridge should publish to", "ready_topic", wsTopics.TopicClientReady.Name())

	// Listen for WebSocket client ready events (when clients connect)
	if err := subscriber.Subscribe(ctx, wsTopics.TopicClientReady.Name(), svc.handleClientConnected); err != nil {
		svc.logger.Error("failed to subscribe to WebSocket client ready events", "error", err)
	} else {
		svc.logger.Info("Successfully subscribed to WebSocket client ready events")
	}

	// Listen for WebSocket client disconnected events
	if err := subscriber.Subscribe(ctx, wsTopics.TopicClientDisconnected.Name(), svc.handleClientDisconnected); err != nil {
		svc.logger.Error("failed to subscribe to WebSocket client disconnected events", "error", err)
	} else {
		svc.logger.Info("Successfully subscribed to WebSocket client disconnected events")
	}

	// Start cleanup goroutine
	go svc.startCleanup()

	svc.logger.Info("Presence service initialized")
	return svc
}

func (s *Service) handleClientConnected(ctx context.Context, msg pubsub.Message) error {
	s.logger.Info("Received client connected event",
		"message", string(msg.Payload),
		"topic", msg.Topic,
	)

	// WebSocket client ready event structure
	var event struct {
		UserID   string `json:"userID"`
		Endpoint string `json:"endpoint"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		s.logger.Error("Failed to unmarshal client ready event", "error", err)
		return err
	}

	s.logger.Info("Processing client connection", "userID", event.UserID, "endpoint", event.Endpoint)

	// Use userID as clientID for now (can be enhanced later)
	s.addPresence(event.UserID, event.UserID, "")

	return nil
}

func (s *Service) addPresence(userID, clientID, userAgent string) {
	// Rate limiting check
	if !s.checkRateLimit(userID) {
		s.logger.Debug("Rate limit exceeded for user", "user_id", userID)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track client to user mapping for disconnection
	s.clients[clientID] = userID

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

	// Get current users while we have the lock
	users := make(map[string]struct{})
	for _, p := range s.presences {
		users[p.UserID] = struct{}{}
	}
	onlineUsers := make([]string, 0, len(users))
	for uid := range users {
		onlineUsers = append(onlineUsers, uid)
	}

	s.logger.Debug("Current online users", "count", len(onlineUsers), "users", onlineUsers)

	// Release lock before publishing to avoid deadlock
	s.mu.Unlock()
	s.publishPresenceUpdateWithUsers(onlineUsers)
	s.mu.Lock() // Re-acquire for defer
}

func (s *Service) handleClientDisconnected(ctx context.Context, msg pubsub.Message) error {
	s.logger.Info("Received client disconnected event",
		"message", string(msg.Payload),
		"topic", msg.Topic,
	)

	// WebSocket client disconnected event structure
	var event struct {
		UserID   string `json:"userID"`
		Endpoint string `json:"endpoint"`
		Reason   string `json:"reason"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		s.logger.Error("Failed to unmarshal client disconnected event", "error", err)
		return err
	}

	s.logger.Info("Processing client disconnection", "userID", event.UserID, "endpoint", event.Endpoint, "reason", event.Reason)

	s.removePresence(event.UserID)

	return nil
}

func (s *Service) removePresence(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	presence, exists := s.presences[userID]
	if !exists {
		s.logger.Debug("User not found in presence list", "user_id", userID)
		return
	}

	// Remove from both maps
	delete(s.presences, userID)
	delete(s.clients, presence.ClientID)

	s.logger.Info("User disconnected",
		"user_id", presence.UserID,
		"client_id", presence.ClientID,
		"remaining_users", len(s.presences))

	// Get current users while we have the lock
	users := make(map[string]struct{})
	for _, p := range s.presences {
		users[p.UserID] = struct{}{}
	}
	onlineUsers := make([]string, 0, len(users))
	for uid := range users {
		onlineUsers = append(onlineUsers, uid)
	}

	// Release lock before publishing to avoid deadlock
	s.mu.Unlock()
	s.publishPresenceUpdateWithUsers(onlineUsers)
	s.mu.Lock() // Re-acquire for defer
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
	
	return s.getOnlineUsersUnsafe()
}

// getOnlineUsersUnsafe returns online users without acquiring lock (internal use)
func (s *Service) getOnlineUsersUnsafe() []string {
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



// publishPresenceUpdateWithUsers publishes presence update with provided user list (avoids deadlock)
func (s *Service) publishPresenceUpdateWithUsers(onlineUsers []string) {
	s.logger.Info("Publishing presence update", "user_count", len(onlineUsers))
	
	jsonPayload := s.getCurrentPresenceWithUsers(onlineUsers)
	jsonMsg := pubsub.Message{
		Topic:   TopicUserStatusUpdate.Name(),
		Payload: jsonPayload,
	}
	err := s.publisher.Publish(context.Background(), jsonMsg)
	if err != nil {
		s.logger.Error("Failed to publish presence update",
			"error", err,
			"topic", TopicUserStatusUpdate.Name())
	} else {
		s.logger.Info("Successfully published presence update")
	}
}



func (s *Service) getCurrentPresenceWithUsers(onlineUsers []string) []byte {
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

// startCleanup runs periodic cleanup of stale presences
func (s *Service) startCleanup() {
	for {
		select {
		case <-s.cleanupTicker.C:
			s.cleanupStalePresences()
		case <-s.stopCleanup:
			s.cleanupTicker.Stop()
			return
		}
	}
}

// cleanupStalePresences removes presences that haven't been updated recently
func (s *Service) cleanupStalePresences() {
	const staleThreshold = 5 * time.Minute // Consider stale after 5 minutes
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := Now()
	var staleUsers []string
	
	for userID, presence := range s.presences {
		if now.Sub(presence.Timestamp) > staleThreshold {
			staleUsers = append(staleUsers, userID)
		}
	}
	
	if len(staleUsers) > 0 {
		s.logger.Info("Cleaning up stale presences", "count", len(staleUsers), "users", staleUsers)
		
		for _, userID := range staleUsers {
			if presence, exists := s.presences[userID]; exists {
				delete(s.presences, userID)
				delete(s.clients, presence.ClientID)
			}
		}
		
		// Get updated user list and publish
		users := make(map[string]struct{})
		for _, p := range s.presences {
			users[p.UserID] = struct{}{}
		}
		onlineUsers := make([]string, 0, len(users))
		for uid := range users {
			onlineUsers = append(onlineUsers, uid)
		}
		
		// Release lock before publishing
		s.mu.Unlock()
		s.publishPresenceUpdateWithUsers(onlineUsers)
		s.mu.Lock() // Re-acquire for defer
	}
}

// Shutdown gracefully stops the presence service
func (s *Service) Shutdown() {
	close(s.stopCleanup)
}