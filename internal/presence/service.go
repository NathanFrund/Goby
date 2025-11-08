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

// ConnectionEvent represents a connection lifecycle event for learning
type ConnectionEvent struct {
	ClientID  string
	UserID    string
	EventType string // "connect", "disconnect", "ping", "reconnect"
	Timestamp time.Time
	Duration  *time.Duration // For disconnect events
	Reason    string         // Disconnect reason
}

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
	UserID     string    `json:"user_id"`
	Status     Status    `json:"status"`
	ClientID   string    `json:"client_id,omitempty"`
	ClientType string    `json:"client_type,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	UserAgent  string    `json:"user_agent,omitempty"`
}

type ConnectionState struct {
	ClientID            string
	UserID              string
	ConnectedAt         time.Time
	LastSeen            time.Time
	DisconnectTime      *time.Time
	Status              ConnectionStatus
	ReconnectCount      int
	TotalUptime         time.Duration
	AveragePingInterval time.Duration
	PingCount           int
}

type ConnectionStatus int

const (
	ConnectionActive       ConnectionStatus = iota
	ConnectionSuspected                     // No ping for > pingInterval
	ConnectionStale                         // No ping for > staleThreshold
	ConnectionReconnecting                  // Within reconnection window
	ConnectionOffline
)

type UserActivityPattern struct {
	UserID                 string
	TypicalSessionLength   time.Duration
	AverageReconnectTime   time.Duration
	PreferredCleanupDelay  time.Duration
	LastActivity           time.Time
	TotalConnections       int
	SuccessfulReconnects   int
	BackgroundTabTolerance time.Duration
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

	// Publishing channel to avoid lock contention during pubsub operations
	publishCh chan publishRequest

	// Connection intelligence and learning
	connectionStates  map[string]*ConnectionState     // clientID -> state
	userPatterns      map[string]*UserActivityPattern // userID -> patterns
	connectionHistory map[string][]ConnectionEvent    // userID -> history
	learningMu        sync.RWMutex

	// Metrics for monitoring presence tracking
	metrics struct {
		totalConnections int64
		totalUsers       int64
		disconnections   int64
		reconnections    int64
		staleCleanups    int64
		rateLimitHits    int64
		debounceTimeouts int64
		publishErrors    int64
		adaptiveCleanups int64 // Connections kept alive due to adaptive logic
	}
}

type publishRequest struct {
	onlineUsers []string
	done        chan struct{}
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
			s.metrics.rateLimitHits++
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
		cleanupTicker:        time.NewTicker(120 * time.Second), // Conservative: cleanup every 2 minutes
		stopCleanup:          make(chan struct{}),
		staleThreshold:       180 * time.Second, // Conservative: 3 minute timeout
		offlineDebounce:      make(map[string]*time.Timer),
		offlineDebounceDelay: OfflineDebounceDelay,
		publishCh:            make(chan publishRequest, 100), // Buffered channel for publishing
		connectionStates:     make(map[string]*ConnectionState),
		userPatterns:         make(map[string]*UserActivityPattern),
		connectionHistory:    make(map[string][]ConnectionEvent),
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

	// Start publishing goroutine
	go svc.startPublishing()

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
		UserID     string `json:"userID"`
		ClientID   string `json:"clientID"`
		ClientType string `json:"clientType"`
		Endpoint   string `json:"endpoint"`
	}

	if err := json.Unmarshal(msg.Payload, &event); err != nil {
		s.logger.Error("Failed to unmarshal client ready event", "error", err)
		return err
	}

	s.logger.Info("Processing client connection",
		"userID", event.UserID,
		"clientID", event.ClientID,
		"clientType", event.ClientType,
		"endpoint", event.Endpoint)

	// Use the actual clientID from the WebSocket bridge
	s.addPresenceWithClientType(event.UserID, event.ClientID, "", event.ClientType)

	// Update timestamp for this client to prevent premature cleanup
	s.updateClientActivity(event.UserID, event.ClientID)

	return nil
}

func (s *Service) addPresence(userID, clientID, userAgent string) {
	s.addPresenceWithClientType(userID, clientID, userAgent, "")
}

// addPresenceWithClientType adds a presence entry with client type information
func (s *Service) addPresenceWithClientType(userID, clientID, userAgent, clientType string) {
	// Rate limiting check
	if !s.checkRateLimit(userID) {
		s.logger.Debug("Rate limit exceeded for user", "user_id", userID)
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Track client to user mapping for disconnection
	s.clients[clientID] = userID

	// Track connection state for intelligence
	s.trackConnectionEvent(userID, clientID, "connect", "")

	// Cancel any pending offline debounce for this user
	s.debounceMu.Lock()
	if timer, exists := s.offlineDebounce[userID]; exists {
		timer.Stop()
		delete(s.offlineDebounce, userID)
		s.logger.Info("Cancelled offline debounce due to reconnection",
			"user_id", userID,
			"client_id", clientID)
		s.metrics.reconnections++
	}
	s.debounceMu.Unlock()

	// Initialize user's presence map if needed
	isNewUser := s.presences[userID] == nil
	if isNewUser {
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
		UserID:     userID,
		Status:     StatusOnline,
		ClientID:   clientID,
		ClientType: clientType,
		Timestamp:  Now(),
		UserAgent:  userAgent,
	}
	s.metrics.totalConnections++
	s.metrics.totalUsers = int64(len(s.presences))

	// Update connection state and learn user patterns
	s.learningMu.Lock()
	if connState := s.connectionStates[clientID]; connState != nil {
		// This is a reconnection - update patterns
		connState.Status = ConnectionActive
		connState.LastSeen = Now()
		connState.ReconnectCount++

		// Learn from reconnection behavior
		s.updateUserPatterns(userID, connState)
	} else {
		// New connection
		s.connectionStates[clientID] = &ConnectionState{
			ClientID:    clientID,
			UserID:      userID,
			ConnectedAt: Now(),
			LastSeen:    Now(),
			Status:      ConnectionActive,
		}
	}
	s.learningMu.Unlock()

	// Get current users while we have the lock
	onlineUsers := s.getOnlineUsersUnsafe()

	s.logger.Debug("Current online users",
		"count", len(onlineUsers),
		"total_connections", s.getTotalConnectionsUnsafe())

	// Publish asynchronously to avoid lock contention
	s.publishAsync(onlineUsers)
}

func (s *Service) handleClientDisconnected(ctx context.Context, msg pubsub.Message) error {
	s.logger.Info("Received client disconnected event",
		"message", string(msg.Payload),
		"topic", msg.Topic,
	)

	// WebSocket client disconnected event structure
	var event struct {
		UserID     string `json:"userID"`
		ClientID   string `json:"clientID"`
		ClientType string `json:"clientType"`
		Endpoint   string `json:"endpoint"`
		Reason     string `json:"reason"`
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
		s.metrics.disconnections++
		s.metrics.totalConnections--

		// Update connection state - must be done while holding the main lock
		// to avoid race conditions with concurrent access
		if connState := s.connectionStates[clientID]; connState != nil {
			now := Now()
			connState.DisconnectTime = &now
			connState.Status = ConnectionOffline
			connState.TotalUptime += now.Sub(connState.ConnectedAt)
		}

		// Track disconnect event
		s.trackConnectionEvent(userID, clientID, "disconnect", "client_disconnect")

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
			// Clean up rate limiter timer for this user
			if timer, exists := s.rateLimiter[userID]; exists {
				timer.Stop()
				delete(s.rateLimiter, userID)
			}
			s.logger.Info("User went offline immediately (debounce disabled)",
				"user_id", userID)

			onlineUsers := s.getOnlineUsersUnsafe()
			s.publishAsync(onlineUsers)
			return
		}

		s.logger.Info("User has no more connections, scheduling offline event",
			"user_id", userID,
			"debounce_delay", s.offlineDebounceDelay)

		// Cancel any existing debounce timer for this user
		s.debounceMu.Lock()
		if timer, exists := s.offlineDebounce[userID]; exists {
			timer.Stop()
			delete(s.offlineDebounce, userID) // Clean up immediately
		}

		// Use adaptive debounce delay based on user's reconnection patterns
		debounceDelay := s.offlineDebounceDelay
		if predictedReconnect := s.predictReconnectionTime(userID); predictedReconnect > 0 {
			// If user typically reconnects quickly, use a shorter debounce
			// but not shorter than 1 second to avoid being too aggressive
			if predictedReconnect < debounceDelay && predictedReconnect > time.Second {
				debounceDelay = predictedReconnect
				s.logger.Debug("Using adaptive debounce delay",
					"user_id", userID,
					"predicted_reconnect", predictedReconnect,
					"adaptive_delay", debounceDelay)
			}
		}

		// Schedule offline event after a delay (to handle page reloads, double-clicks, etc.)
		s.offlineDebounce[userID] = time.AfterFunc(debounceDelay, func() {
			s.debounceMu.Lock()
			defer s.debounceMu.Unlock()
			// Double-check the timer still exists and user is still offline
			if timer, exists := s.offlineDebounce[userID]; exists && timer.Stop() {
				delete(s.offlineDebounce, userID)
				s.handleDebouncedOffline(userID)
			}
		})
		s.debounceMu.Unlock()

		// Don't publish update yet - wait for debounce
		return
	}

	// User still has other connections, publish update immediately
	onlineUsers := s.getOnlineUsersUnsafe()
	s.publishAsync(onlineUsers)
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
		// Clean up rate limiter timer for this user
		if timer, exists := s.rateLimiter[userID]; exists {
			timer.Stop()
			delete(s.rateLimiter, userID)
		}
		s.metrics.debounceTimeouts++

		s.logger.Info("User went offline after debounce period",
			"user_id", userID)

		// Publish update asynchronously
		onlineUsers := s.getOnlineUsersUnsafe()
		s.publishAsync(onlineUsers)
	} else {
		// User reconnected, cancel offline event
		s.logger.Info("User reconnected during debounce period, staying online",
			"user_id", userID,
			"connections", len(clientPresences))
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
		s.metrics.totalConnections--
	}

	// Clean up rate limiter timer for this user
	if timer, exists := s.rateLimiter[userID]; exists {
		timer.Stop()
		delete(s.rateLimiter, userID)
	}

	// Remove user's presence map
	delete(s.presences, userID)

	s.logger.Info("User disconnected",
		"user_id", userID,
		"connections_removed", len(clientPresences),
		"remaining_users", len(s.presences))

	// Get current users while we have the lock
	onlineUsers := s.getOnlineUsersUnsafe()

	// Publish asynchronously to avoid lock contention
	s.publishAsync(onlineUsers)
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

// updateClientActivity updates the timestamp of a client's presence to show recent activity
func (s *Service) updateClientActivity(userID, clientID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the user's presence and update the specific client's timestamp
	if clientPresences, exists := s.presences[userID]; exists {
		if presence, clientExists := clientPresences[clientID]; clientExists {
			presence.Timestamp = Now()
			clientPresences[clientID] = presence

			s.logger.Debug("Updated client activity timestamp",
				"user_id", userID,
				"client_id", clientID,
				"new_timestamp", presence.Timestamp)
		}
	}
}

// publishAsync sends a publish request to the background publishing goroutine
func (s *Service) publishAsync(onlineUsers []string) {
	req := publishRequest{
		onlineUsers: append([]string(nil), onlineUsers...), // Copy slice
		done:        make(chan struct{}),
	}
	select {
	case s.publishCh <- req:
		// Request sent, wait for completion if needed
		<-req.done
	default:
		s.logger.Warn("Publish channel full, dropping presence update")
	}
}

// updateUserPatterns learns from user connection behavior
func (s *Service) updateUserPatterns(userID string, connState *ConnectionState) {
	// Initialize pattern if it doesn't exist
	if s.userPatterns[userID] == nil {
		s.userPatterns[userID] = &UserActivityPattern{
			UserID:               userID,
			LastActivity:         Now(),
			TotalConnections:     1,
			SuccessfulReconnects: 0,
		}
	}

	pattern := s.userPatterns[userID]
	pattern.LastActivity = Now()
	pattern.TotalConnections++

	// Learn reconnection patterns
	if connState.ReconnectCount > 0 {
		pattern.SuccessfulReconnects++

		// Calculate average reconnect time
		history := s.connectionHistory[userID]
		if len(history) >= 2 {
			var reconnectTimes []time.Duration
			for i := 1; i < len(history); i++ {
				if history[i].EventType == "connect" && history[i-1].EventType == "disconnect" {
					reconnectTime := history[i].Timestamp.Sub(history[i-1].Timestamp)
					if reconnectTime > 0 && reconnectTime < 10*time.Minute {
						reconnectTimes = append(reconnectTimes, reconnectTime)
					}
				}
			}

			if len(reconnectTimes) > 0 {
				var total time.Duration
				for _, t := range reconnectTimes {
					total += t
				}
				pattern.AverageReconnectTime = total / time.Duration(len(reconnectTimes))
			}
		}
	}
}

// predictReconnectionTime estimates when a user might reconnect based on their patterns
func (s *Service) predictReconnectionTime(userID string) time.Duration {
	s.learningMu.RLock()
	pattern := s.userPatterns[userID]
	history := s.connectionHistory[userID]
	s.learningMu.RUnlock()

	if pattern == nil || pattern.AverageReconnectTime == 0 {
		return 30 * time.Second // Default fallback
	}

	// Look at recent disconnect patterns
	var recentDisconnects []time.Time
	for _, event := range history {
		if event.EventType == "disconnect" && Now().Sub(event.Timestamp) < 24*time.Hour {
			recentDisconnects = append(recentDisconnects, event.Timestamp)
		}
	}

	// If user has disconnected recently and typically reconnects quickly, predict reconnection
	if len(recentDisconnects) > 0 {
		timeSinceLastDisconnect := Now().Sub(recentDisconnects[len(recentDisconnects)-1])
		if timeSinceLastDisconnect < pattern.AverageReconnectTime*2 {
			// User might reconnect soon
			return pattern.AverageReconnectTime - timeSinceLastDisconnect
		}
	}

	return pattern.AverageReconnectTime
}

// trackConnectionEvent records connection lifecycle events for learning
func (s *Service) trackConnectionEvent(userID, clientID, eventType, reason string) {
	s.learningMu.Lock()
	defer s.learningMu.Unlock()

	event := ConnectionEvent{
		ClientID:  clientID,
		UserID:    userID,
		EventType: eventType,
		Timestamp: Now(),
		Reason:    reason,
	}

	// Keep only recent history (last 100 events per user)
	history := s.connectionHistory[userID]
	if len(history) >= 100 {
		// Remove oldest events, keep most recent 99
		copy(history, history[1:])
		history = history[:99]
	}
	history = append(history, event)
	s.connectionHistory[userID] = history
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
		s.metrics.publishErrors++
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

// startPublishing handles publishing presence updates asynchronously to avoid lock contention
func (s *Service) startPublishing() {
	for req := range s.publishCh {
		s.publishPresenceUpdateWithUsers(req.onlineUsers)
		close(req.done)
	}
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

// calculateAdaptiveThreshold determines the appropriate cleanup threshold for a user based on their behavior
// Conservative approach: Use fixed threshold, avoid aggressive adaptive logic
// NOTE: Currently unused but kept for future adaptive cleanup features
// nolint:unused // Will be used when implementing adaptive cleanup based on user behavior patterns
func (s *Service) calculateAdaptiveThreshold(_ string) time.Duration {
	// Fixed conservative threshold to prevent false cleanups
	return s.staleThreshold // Always 3 minutes
}

// cleanupStalePresences removes presences that haven't been updated recently
func (s *Service) cleanupStalePresences() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var staleUsers []string
	totalStaleConnections := 0

	// Find and remove stale connections (conservative server-side approach)
	for userID, clientPresences := range s.presences {
		for clientID, presence := range clientPresences {
			timeSinceLastSeen := Now().Sub(presence.Timestamp)

			// Conservative: Only remove if significantly past threshold (3 minutes + 30 second buffer)
			if timeSinceLastSeen > s.staleThreshold+(30*time.Second) {
				delete(clientPresences, clientID)
				delete(s.clients, clientID)
				totalStaleConnections++
				s.metrics.staleCleanups++
				s.metrics.totalConnections--

				// Update connection state
				s.learningMu.Lock()
				if connState := s.connectionStates[clientID]; connState != nil {
					connState.Status = ConnectionOffline
				}
				s.learningMu.Unlock()

				s.trackConnectionEvent(userID, clientID, "disconnect", "stale_cleanup")

				s.logger.Info("Removed stale connection (conservative cleanup)",
					"user_id", userID,
					"client_id", clientID,
					"last_seen", presence.Timestamp,
					"time_since_last_seen", timeSinceLastSeen,
					"threshold", s.staleThreshold)
			}
		}

		// Remove user if no clients remain
		if len(clientPresences) == 0 {
			delete(s.presences, userID)
			// Clean up rate limiter timer for this user
			if timer, exists := s.rateLimiter[userID]; exists {
				timer.Stop()
				delete(s.rateLimiter, userID)
			}
			staleUsers = append(staleUsers, userID)
		}
	}

	if len(staleUsers) == 0 && totalStaleConnections == 0 {
		return
	}

	s.logger.Info("Cleaned up stale presences (conservative approach)",
		"users_removed", len(staleUsers),
		"connections_removed", totalStaleConnections,
		"users", staleUsers,
	)

	// Get the new list of online users while still holding the lock
	onlineUsers := s.getOnlineUsersUnsafe()

	// Publish update asynchronously (no lock contention)
	s.publishAsync(onlineUsers)
}

// GetPresenceProbability calculates the probability that a user is actually present
// based on their connection patterns and current state
func (s *Service) GetPresenceProbability(userID string) float64 {
	s.mu.RLock()
	clientPresences, exists := s.presences[userID]
	s.mu.RUnlock()

	if !exists || len(clientPresences) == 0 {
		return 0.0 // Definitely offline
	}

	s.learningMu.RLock()
	pattern := s.userPatterns[userID]
	history := s.connectionHistory[userID]
	s.learningMu.RUnlock()

	// Base probability from active connections
	baseProbability := 0.8 // High confidence with active connections

	// Adjust based on connection patterns
	if pattern != nil && len(history) > 5 {
		// Users with frequent reconnections are more likely to be present
		if pattern.SuccessfulReconnects > 3 {
			baseProbability += 0.1
		}

		// Users with short session lengths might be more transient
		if pattern.TypicalSessionLength < 5*time.Minute {
			baseProbability -= 0.1
		}

		// Recent activity increases confidence
		if Now().Sub(pattern.LastActivity) < 30*time.Second {
			baseProbability += 0.05
		}
	}

	// Cap between 0 and 1
	if baseProbability > 1.0 {
		baseProbability = 1.0
	} else if baseProbability < 0.0 {
		baseProbability = 0.0
	}

	return baseProbability
}

// GetMetrics returns current presence service metrics
func (s *Service) GetMetrics() map[string]int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]int64{
		"total_connections": s.metrics.totalConnections,
		"total_users":       s.metrics.totalUsers,
		"disconnections":    s.metrics.disconnections,
		"reconnections":     s.metrics.reconnections,
		"stale_cleanups":    s.metrics.staleCleanups,
		"rate_limit_hits":   s.metrics.rateLimitHits,
		"debounce_timeouts": s.metrics.debounceTimeouts,
		"publish_errors":    s.metrics.publishErrors,
		"adaptive_cleanups": s.metrics.adaptiveCleanups,
	}
}

// Shutdown gracefully stops the presence service
func (s *Service) Shutdown() {
	close(s.stopCleanup)
	close(s.publishCh) // Stop the publishing goroutine
}
