# WebSocket Bridge Migration Guide

This guide helps you migrate from the deprecated `websocket_bridge.go` to the new `bridge.go` implementation.

## Key Changes

1. **Dual Endpoint Support**
   - Old: Single WebSocket endpoint (`/ws`)
   - New: Two distinct endpoints:
     - `/ws/html` - For HTMX fragments
     - `/ws/data` - For structured data

2. **Message Structure**
   - Old: Custom message structure
   - New: Standardized message format with topic-based routing

3. **Client Management**
   - Old: Manual client management
   - New: Automatic connection handling with type safety

## Migration Steps

### 1. Update Dependencies

Remove the old WebSocket dependencies and ensure you have the new ones:

```go
// Remove these if no longer needed
// "github.com/your/old/websocket"

// Add these if not present
github.com/ThreeDotsLabs/watermill
```

### 2. Update WebSocket Connection

#### Old Way
```go
// Old connection setup
bridge := websocket.NewWebsocketBridge(publisher)
router.GET("/ws", bridge.ServeEcho)
```

#### New Way
```go
// New connection setup
bridge := websocket.NewBridge(publisher)
bridge.RegisterEchoRoutes(router) // Registers both /ws/html and /ws/data
```

### 3. Sending Messages

#### Old Way
```go
// Sending to a specific client
bridge.WriteToClient(userID, message)

// Broadcasting to all
bridge.BroadcastToAll(message)
```

#### New Way
```go
// Sending to a specific client
bridge.Send(clientID, message)

// Broadcasting to all HTML clients
bridge.Broadcast(message, websocket.ConnectionTypeHTML)

// Broadcasting to all data clients
bridge.Broadcast(message, websocket.ConnectionTypeData)

// Broadcasting to all connected clients
bridge.Broadcast(message, websocket.ConnectionTypeHTML, websocket.ConnectionTypeData)
```

### 4. Handling Incoming Messages

#### Old Way
```go
// Old message handling
func (b *WebsocketBridge) HandleIncomingMessage(client *Client, message []byte) {
    // Handle message
}
```

#### New Way
```go
// New message handling
for msg := range bridge.Incoming() {
    // msg contains ClientID, Topic, and Payload
    switch msg.Topic {
    case "chat.message":
        // Handle chat message
    default:
        slog.Warn("Unknown message topic", "topic", msg.Topic)
    }
}
```

## Migration Status

- [x] Add deprecation notice to old implementation
- [ ] Update chat module
- [ ] Update wargame module
- [ ] Update other modules
- [ ] Remove old implementation

## Help

For questions or issues, please open an issue in the repository.
