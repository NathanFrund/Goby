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

const (
	// DefaultWebSocketPingInterval is the expected interval between WebSocket pings.
	// This should ideally be kept in sync with the `pingPeriod` in the websocket bridge.
	DefaultWebSocketPingInterval = 54 * time.Second

	// StaleThresholdMultiplier determines how many missed pings to tolerate before considering a client stale.
	StaleThresholdMultiplier = 2

	// DefaultStaleThreshold is the default time after which a presence is considered stale.
	DefaultStaleThreshold = DefaultWebSocketPingInterval * StaleThresholdMultiplier
	
	// OfflineDebounceDelay is the time to wait before marking a user as offline after their last connection closes.
	// This handles page reloads, double-clicks, and slow network conditions gracefully.
	// Can be overridden using WithOfflineDebounce() option.
	// Recommended: 3-10 seconds depending on your network conditions and browser quirks.
	OfflineDebounceDelay = 5 * time.Second
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
	presences map[string]map[string]Presence // userID -> clientID -> Presence
	clients   map[string]string              // clientID -> userID (for disconnect lookup)
	publisher pubsub.Publisher
	logger    *slog.Logger

	// Rate limiting
	rateLimiter map[string]*time.Timer // userID -> last update timer
	rateMu      sync.Mutex

	// Cleanup mechanism
	cleanupTicker  *time.Ticker
	stopCleanup    chan struct{}
	staleThreshold time.Duration
	
	// Debouncing for offline events (to handle page reloads gracefully)
	offlineDebounce      map[string]*time.Timer // userID -> debounce timer
	offlineDebounceDelay time.Duration          // configurable delay
	debounceMu           sync.Mutex
}

// Option is a function that configures a Service.
type Option func(*Service)

// WithStaleThreshold sets a custom stale threshold for the presence service.
func WithStaleThreshold(d time.Duration) Option {
	return func(s *Service) {
		s.staleThreshold = d
	}
}

// WithOfflineDebounce sets a custom debounce delay for offline events.
// This is useful for handling different network conditions or browser behaviors.
// Set to 0 to disable debouncing (useful for testing).
func WithOfflineDebounce(d time.Duration) Option {
	return func(s *Service) {
		s.offlineDebounceDelay = d
	}
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
func NewService(publisher pubsub.Publisher, subscriber pubsub.Subscriber, topicMgr *topicmgr.Manager, opts ...Option) *Service {
	svc := &Service{
		presences:            make(map[string]map[string]Presence),
		clients:              make(map[string]string),
		publisher:            publisher,
		logger:               slog.Default().With("service", "presence"),
		rateLimiter:          make(map[string]*time.Timer),
		cleanupTicker:        time.NewTicker(30 * time.Second), // Cleanup every 30 seconds
		stopCleanup:          make(chan struct{}),
		staleThreshold:       DefaultStaleThreshold,
		offlineDebounce:      make(map[string]*time.Timer),
		offlineDebounceDelay: OfflineDebounceDelay,
	}

	// Apply functional options
	for _, opt := range opts {
		opt(svc)
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
		ClientID string `json:"clientID"`
		Endpoint string `json:"endpoint"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		s.logger.Error("Failed to unmarshal client ready event", "error", err)
		return err
	}

	s.logger.Info("Processing client connection", 
		"userID", event.UserID, 
		"clientID", event.ClientID,
		"endpoint", event.Endpoint)

	// Use the actual clientID from the WebSocket bridge
	s.addPresence(event.UserID, event.ClientID, "")

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

	// Cancel any pending offline debounce for this user
	s.debounceMu.Lock()
	if timer, exists := s.offlineDebounce[userID]; exists {
		timer.Stop()
		delete(s.offlineDebounce, userID)
		s.logger.Info("Cancelled offline debounce due to reconnection",
			"user_id", userID,
			"client_id", clientID)
	}
	s.debounceMu.Unlock()

	// Initialize user's presence map if needed
	if s.presences[userID] == nil {
		s.presences[userID] = make(map[string]Presence)
		s.logger.Info("User came online",
			"user_id", userID,
			"client_id", clientID,
			"user_agent", userAgent)
	} else {
		s.logger.Debug("Adding additional connection for user",
			"user_id", userID,
			"client_id", clientID,
			"existing_connections", len(s.presences[userID]),
			"user_agent", userAgent)
	}

	// Add this specific client's presence
	s.presences[userID][clientID] = Presence{
		UserID:    userID,
		Status:    StatusOnline,
		ClientID:  clientID,
		Timestamp: Now(),
		UserAgent: userAgent,
	}

	// Get current users while we have the lock
	onlineUsers := s.getOnlineUsersUnsafe()

	s.logger.Debug("Current online users",
		"count", len(onlineUsers),
		"total_connections", s.getTotalConnectionsUnsafe())

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
		ClientID string `json:"clientID"`
		Endpoint string `json:"endpoint"`
		Reason   string `json:"reason"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		s.logger.Error("Failed to unmarshal client disconnected event", "error", err)
		return err
	}

	s.logger.Info("Processing client disconnection", 
		"userID", event.UserID, 
		"clientID", event.ClientID,
		"endpoint", event.Endpoint, 
		"reason", event.Reason)

	s.removePresenceForClient(event.UserID, event.ClientID)

	return nil
}

// removePresenceForClient removes a specific client connection for a user
func (s *Service) removePresenceForClient(userID, clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientPresences, exists := s.presences[userID]
	if !exists {
		s.logger.Debug("User not found in presence list", "user_id", userID, "client_id", clientID)
		return
	}

	// Remove this specific client connection
	if _, clientExists := clientPresences[clientID]; clientExists {
		delete(clientPresences, clientID)
		delete(s.clients, clientID)
		
		s.logger.Info("Client disconnected",
			"user_id", userID,
			"client_id", clientID,
			"remaining_connections", len(clientPresences))
	}

	// If no more clients for this user, debounce the offline event
	if len(clientPresences) == 0 {
		// If debounce is disabled (0), mark offline immediately
		if s.offlineDebounceDelay == 0 {
			delete(s.presences, userID)
			delete(s.rateLimiter, userID)
			s.logger.Info("User went offline immediately (debounce disabled)",
				"user_id", userID)
			
			onlineUsers := s.getOnlineUsersUnsafe()
			s.mu.Unlock()
			s.publishPresenceUpdateWithUsers(onlineUsers)
			s.mu.Lock()
			return
		}
		
		s.logger.Info("User has no more connections, scheduling offline event",
			"user_id", userID,
			"debounce_delay", s.offlineDebounceDelay)
		
		// Cancel any existing debounce timer for this user
		s.debounceMu.Lock()
		if timer, exists := s.offlineDebounce[userID]; exists {
			timer.Stop()
		}
		
		// Schedule offline event after a delay (to handle page reloads, double-clicks, etc.)
		s.offlineDebounce[userID] = time.AfterFunc(s.offlineDebounceDelay, func() {
			s.handleDebouncedOffline(userID)
		})
		s.debounceMu.Unlock()
		
		// Don't publish update yet - wait for debounce
		return
	}

	// User still has other connections, publish update immediately
	onlineUsers := s.getOnlineUsersUnsafe()
	s.mu.Unlock()
	s.publishPresenceUpdateWithUsers(onlineUsers)
	s.mu.Lock() // Re-acquire for defer
}

// handleDebouncedOffline is called after the debounce period to mark a user as offline
func (s *Service) handleDebouncedOffline(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Check if user reconnected during debounce period
	clientPresences, exists := s.presences[userID]
	if !exists || len(clientPresences) == 0 {
		// User is still offline, remove them
		delete(s.presences, userID)
		delete(s.rateLimiter, userID)
		
		s.logger.Info("User went offline after debounce period",
			"user_id", userID)
		
		// Clean up debounce timer
		s.debounceMu.Lock()
		delete(s.offlineDebounce, userID)
		s.debounceMu.Unlock()
		
		// Publish update
		onlineUsers := s.getOnlineUsersUnsafe()
		s.mu.Unlock()
		s.publishPresenceUpdateWithUsers(onlineUsers)
		s.mu.Lock() // Re-acquire for defer
	} else {
		// User reconnected, cancel offline event
		s.logger.Info("User reconnected during debounce period, staying online",
			"user_id", userID,
			"connections", len(clientPresences))
		
		// Clean up debounce timer
		s.debounceMu.Lock()
		delete(s.offlineDebounce, userID)
		s.debounceMu.Unlock()
	}
}

// removePresence removes all client connections for a user
func (s *Service) removePresence(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	clientPresences, exists := s.presences[userID]
	if !exists {
		s.logger.Debug("User not found in presence list", "user_id", userID)
		return
	}

	// Remove all client connections for this user
	for clientID := range clientPresences {
		delete(s.clients, clientID)
	}

	// Remove user's presence map
	delete(s.presences, userID)

	s.logger.Info("User disconnected",
		"user_id", userID,
		"connections_removed", len(clientPresences),
		"remaining_users", len(s.presences))

	// Get current users while we have the lock
	onlineUsers := s.getOnlineUsersUnsafe()

	// Release lock before publishing to avoid deadlock
	s.mu.Unlock()
	s.publishPresenceUpdateWithUsers(onlineUsers)
	s.mu.Lock() // Re-acquire for defer
}

// GetPresence returns the current presence status for a user
// If the user has multiple connections, returns the most recent one
func (s *Service) GetPresence(userID string) (Presence, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clientPresences, exists := s.presences[userID]
	if !exists || len(clientPresences) == 0 {
		return Presence{}, false
	}

	// Return the most recent presence
	var mostRecent Presence
	for _, p := range clientPresences {
		if mostRecent.Timestamp.IsZero() || p.Timestamp.After(mostRecent.Timestamp) {
			mostRecent = p
		}
	}

	return mostRecent, true
}

// GetOnlineUsers returns a list of currently online user IDs
func (s *Service) GetOnlineUsers() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.getOnlineUsersUnsafe()
}

// getOnlineUsersUnsafe returns online users without acquiring lock (internal use)
func (s *Service) getOnlineUsersUnsafe() []string {
	// A user is online if they have at least one active client
	result := make([]string, 0, len(s.presences))
	for userID, clientPresences := range s.presences {
		if len(clientPresences) > 0 {
			result = append(result, userID)
		}
	}

	return result
}

// getTotalConnectionsUnsafe returns total number of connections without acquiring lock
func (s *Service) getTotalConnectionsUnsafe() int {
	total := 0
	for _, clientPresences := range s.presences {
		total += len(clientPresences)
	}
	return total
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
	s.mu.Lock()
	defer s.mu.Unlock()

	threshold := Now().Add(-s.staleThreshold)
	var staleUsers []string
	totalStaleConnections := 0

	// Find and remove stale connections
	for userID, clientPresences := range s.presences {
		for clientID, presence := range clientPresences {
			if presence.Timestamp.Before(threshold) {
				delete(clientPresences, clientID)
				delete(s.clients, clientID)
				totalStaleConnections++
			}
		}

		// Remove user if no clients remain
		if len(clientPresences) == 0 {
			delete(s.presences, userID)
			delete(s.rateLimiter, userID)
			staleUsers = append(staleUsers, userID)
		}
	}

	if len(staleUsers) == 0 && totalStaleConnections == 0 {
		return
	}

	s.logger.Info("Cleaned up stale presences",
		"users_removed", len(staleUsers),
		"connections_removed", totalStaleConnections,
		"users", staleUsers)

	// Publish update asynchronously after getting the new list of online users
	onlineUsers := s.getOnlineUsersUnsafe()
	go s.publishPresenceUpdateWithUsers(onlineUsers)
}

// Shutdown gracefully stops the presence service
func (s *Service) Shutdown() {
	close(s.stopCleanup)
}
