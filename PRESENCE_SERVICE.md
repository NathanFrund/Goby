# Presence Service

The Presence Service tracks user online/offline status in real-time across multiple connections and devices. It integrates with the WebSocket bridge to provide accurate presence information with graceful handling of network conditions and browser quirks.

## Features

- **Multi-Connection Support**: Tracks multiple connections per user (multiple tabs, devices)
- **Real-Time Updates**: Broadcasts presence changes via WebSocket
- **Graceful Reconnection**: Handles page reloads and network interruptions smoothly
- **Automatic Cleanup**: Removes stale connections based on configurable thresholds
- **Rate Limiting**: Prevents excessive presence update broadcasts
- **Thread-Safe**: Safe for concurrent access from multiple goroutines

## Architecture

### Connection Tracking

The presence service maintains a nested map structure:
```
userID -> clientID -> Presence
```

This allows a single user to have multiple active connections (e.g., multiple browser tabs, mobile app + web browser).

### Lifecycle

1. **Connection**: When a WebSocket connects, the service receives a `ws.client.ready` event
2. **Tracking**: The service adds the connection to the user's presence map
3. **Heartbeat**: WebSocket pings keep the connection alive (every ~54 seconds)
4. **Disconnection**: When a connection closes, the service receives a `ws.client.disconnected` event
5. **Debouncing**: If it's the user's last connection, the service waits before marking them offline
6. **Cleanup**: A background process removes stale connections that haven't sent heartbeats

## Configuration

### Environment Variables

#### `PRESENCE_OFFLINE_DEBOUNCE`

**Type**: Duration (e.g., `5s`, `10s`, `500ms`)  
**Default**: `5s`  
**Description**: Time to wait before marking a user as offline after their last connection closes.

This debounce period handles:
- Page reloads (old connection closes, new one opens)
- Double-clicks on reload button
- Slow network conditions
- Browser-specific reconnection behaviors

**Recommended Values**:
- **3-5 seconds**: Good for fast, reliable networks
- **5-8 seconds**: Recommended for production (handles most edge cases)
- **8-10 seconds**: For slower networks or problematic browsers (Opera, etc.)
- **0**: Disables debouncing (useful for testing, not recommended for production)

**Example Usage**:

```bash
# In .env file or environment
PRESENCE_OFFLINE_DEBOUNCE=8s
```

```go
// In main.go
debounceDelay := 5 * time.Second
if delay := os.Getenv("PRESENCE_OFFLINE_DEBOUNCE"); delay != "" {
    if d, err := time.ParseDuration(delay); err == nil {
        debounceDelay = d
    }
}

presenceService := presence.NewService(ps, ps, topicManager,
    presence.WithOfflineDebounce(debounceDelay),
)
```

### Programmatic Configuration

You can also configure the presence service programmatically using options:

```go
presenceService := presence.NewService(publisher, subscriber, topicManager,
    // Set custom offline debounce delay
    presence.WithOfflineDebounce(8 * time.Second),
    
    // Set custom stale threshold (how long before a connection is considered dead)
    presence.WithStaleThreshold(3 * time.Minute),
)
```

## How It Works

### Offline Debouncing

When a user's last connection closes, the service doesn't immediately mark them as offline. Instead:

1. **Schedule**: A timer is set for the configured debounce delay (default: 5 seconds)
2. **Wait**: The service waits for the timer to expire
3. **Check**: When the timer fires, the service checks if the user reconnected
4. **Result**:
   - If the user reconnected → Cancel offline event, user stays online
   - If still no connections → Mark user as offline, broadcast update

This prevents the "flickering" effect where users appear to go offline and come back online during page reloads.

### Example Timeline

**Without Debouncing** (problematic):
```
0.0s: User clicks reload
0.1s: Old WebSocket disconnects → User marked OFFLINE
0.2s: Page loads, requests presence → Shows "Loading..."
0.5s: New WebSocket connects → User marked ONLINE
Result: User briefly appears offline, UI shows "Loading..."
```

**With Debouncing** (smooth):
```
0.0s: User clicks reload
0.1s: Old WebSocket disconnects → Debounce timer starts (5s)
0.2s: Page loads, requests presence → User still shows as ONLINE
0.5s: New WebSocket connects → Debounce timer cancelled
Result: User stays online throughout, no UI flicker
```

## API Reference

### Creating the Service

```go
func NewService(
    publisher pubsub.Publisher,
    subscriber pubsub.Subscriber,
    topicMgr *topicmgr.Manager,
    opts ...Option,
) *Service
```

### Configuration Options

```go
// WithOfflineDebounce sets a custom debounce delay for offline events
func WithOfflineDebounce(d time.Duration) Option

// WithStaleThreshold sets a custom stale threshold for the presence service
func WithStaleThreshold(d time.Duration) Option
```

### Public Methods

```go
// GetOnlineUsers returns a list of currently online user IDs
func (s *Service) GetOnlineUsers() []string

// GetPresence returns the presence information for a specific user
func (s *Service) GetPresence(userID string) (Presence, bool)

// IsUserOnline checks if a user has any active connections
func (s *Service) IsUserOnline(userID string) bool

// Shutdown gracefully stops the presence service
func (s *Service) Shutdown()
```

## Integration

### Module Integration

To use the presence service in your module:

```go
// In your module's Dependencies struct
type Dependencies struct {
    Publisher       pubsub.Publisher
    Subscriber      pubsub.Subscriber
    Renderer        rendering.Renderer
    PresenceService *presence.Service  // Add this
}

// In your handler
func (h *Handler) GetPresence(c echo.Context) error {
    onlineUsers := h.presenceService.GetOnlineUsers()
    
    // Render presence component
    component := components.OnlineUsers(onlineUsers)
    html, err := h.renderer.RenderComponent(c.Request().Context(), component)
    if err != nil {
        return err
    }
    
    return c.HTMLBlob(http.StatusOK, html)
}
```

### Real-Time Updates

The presence service automatically publishes updates to the `presence.user.online` topic. Subscribe to receive real-time presence changes:

```go
// In your subscriber
func (s *Subscriber) Start(ctx context.Context) {
    err := s.subscriber.Subscribe(ctx, presence.TopicUserStatusUpdate.Name(), 
        s.handlePresenceUpdate)
    if err != nil {
        log.Error("Failed to subscribe to presence updates", "error", err)
    }
}

func (s *Subscriber) handlePresenceUpdate(ctx context.Context, msg pubsub.Message) error {
    var update struct {
        Type  string   `json:"type"`
        Users []string `json:"users"`
    }
    
    if err := json.Unmarshal(msg.Payload, &update); err != nil {
        return err
    }
    
    // Render and broadcast the updated presence list
    // ...
    
    return nil
}
```

## Monitoring

The presence service logs important events at different levels:

- **INFO**: User connections, disconnections, and presence updates
- **DEBUG**: Rate limiting, connection counts, and detailed state
- **WARN**: Unexpected conditions or errors

Example log output:
```
level=INFO msg="User came online" service=presence user_id=user@example.com client_id=abc123
level=INFO msg="Client disconnected" service=presence user_id=user@example.com client_id=abc123 remaining_connections=1
level=INFO msg="User has no more connections, scheduling offline event" service=presence user_id=user@example.com debounce_delay=5s
level=INFO msg="User reconnected during debounce period, staying online" service=presence user_id=user@example.com connections=1
```

## Best Practices

1. **Set appropriate debounce delays**: 
   - Development: 3-5 seconds
   - Production: 5-8 seconds
   - Slow networks: 8-10 seconds

2. **Monitor presence updates**: Watch logs for patterns that might indicate issues

3. **Test edge cases**: 
   - Multiple tabs
   - Page reloads
   - Network interruptions
   - Browser-specific behaviors

4. **Use environment variables**: Make configuration easy to adjust per environment

5. **Handle offline gracefully**: Design your UI to handle users going offline smoothly

## Troubleshooting

### Users appear offline during page reload

**Cause**: Debounce delay is too short for your network conditions  
**Solution**: Increase `PRESENCE_OFFLINE_DEBOUNCE` to 8-10 seconds

### Users stay online after disconnecting

**Cause**: Stale threshold is too long, or cleanup isn't running  
**Solution**: Check logs for cleanup events, adjust stale threshold if needed

### Presence updates are slow

**Cause**: Rate limiting is preventing updates  
**Solution**: This is by design to prevent excessive broadcasts (1 update per second per user)

### Different behavior in different browsers

**Cause**: Browsers handle WebSocket reconnection differently  
**Solution**: Increase debounce delay to accommodate slower browsers

## Performance Considerations

- **Memory**: The service maintains one `Presence` struct per active connection
- **CPU**: Background cleanup runs every 30 seconds
- **Network**: Presence updates are broadcast to all connected clients
- **Rate Limiting**: Maximum 1 presence update per second per user

For most applications, the presence service has minimal overhead. With 1000 concurrent users and 2 connections each:
- Memory: ~200KB (approximate)
- CPU: Negligible
- Network: Only when users connect/disconnect

## Future Enhancements

Potential improvements for future versions:

- [ ] Configurable cleanup interval
- [ ] Metrics/observability integration
- [ ] Per-user presence metadata (status message, activity, etc.)
- [ ] Presence history/analytics
- [ ] Custom presence states (away, busy, etc.)
