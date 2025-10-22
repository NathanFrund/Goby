# WebSocket Topic Broadcasting Flow

## Overview

This document explains how WebSocket topic-based broadcasting works with the new topic management system. The key insight is that **the new system enhances the existing WebSocket functionality without breaking it** - it adds compile-time safety and better documentation while maintaining the same broadcasting behavior.

## ✅ **Yes, WebSocket Topic Broadcasting Still Works!**

The new topic management system **enhances** the existing WebSocket subscription system without breaking it. Here's how the complete message flow works:

## Complete Message Flow: Chat Example

Let's trace a chat message from HTTP request to WebSocket delivery:

### **Step 1: User Sends Message via HTTP**

```go
// internal/modules/chat/handler.go
func (h *Handler) MessagePost(c echo.Context) error {
    user := c.Get(middleware.UserContextKey).(*domain.User)
    content := c.FormValue("content")

    // Create structured message
    msg := struct {
        Content string `json:"content"`
        User    string `json:"user"`
    }{
        Content: content,
        User:    user.Email,
    }

    payload, _ := json.Marshal(msg)

    // 🎯 Publish using typed topic - compile-time safe!
    h.publisher.Publish(c.Request().Context(), pubsub.Message{
        Topic:   chat.TopicMessages.Name(), // "chat.messages" - no magic strings!
        UserID:  user.Email,
        Payload: payload,
    })

    return c.NoContent(http.StatusOK)
}
```

**What happens here:**

- User submits chat form via HTTP POST
- Handler uses **typed topic** `chat.TopicMessages.Name()` instead of magic string
- Message published to pub/sub system with topic name `"chat.messages"`

### **Step 2: Chat Subscriber Processes and Renders**

```go
// internal/modules/chat/subscriber.go
func (cs *ChatSubscriber) Start(ctx context.Context) {
    // 🎯 Subscribe using typed topic - compile-time safe!
    go func() {
        err := cs.subscriber.Subscribe(ctx, chat.TopicMessages.Name(), cs.handleChatMessage)
        if err != nil && err != context.Canceled {
            slog.Error("Chat message subscriber stopped with error", "error", err)
        }
    }()
}

func (cs *ChatSubscriber) handleChatMessage(ctx context.Context, msg pubsub.Message) error {
    // Parse message payload
    var payload struct {
        Content string `json:"content"`
        User    string `json:"user"`
    }
    json.Unmarshal(msg.Payload, &payload)

    // 🎨 Render message to HTML using templ component
    messageComponent := components.ChatMessage(payload.User, payload.Content, time.Now())
    renderedHTML, err := cs.renderer.RenderComponent(ctx, messageComponent)
    if err != nil {
        return err
    }

    // 🚀 Publish rendered HTML to WebSocket broadcast topic
    return cs.publisher.Publish(ctx, pubsub.Message{
        Topic:   wsTopics.TopicHTMLBroadcast.Name(), // "ws.html.broadcast"
        Payload: renderedHTML, // Rendered HTML ready for browser
    })
}
```

**What happens here:**

- Chat subscriber listens on `"chat.messages"` topic using **typed topic reference**
- When message arrives, it renders the message to HTML using templ components
- Publishes rendered HTML to WebSocket broadcast topic `"ws.html.broadcast"`

### **Step 3: WebSocket Bridge Receives and Broadcasts**

```go
// internal/websocket/bridge.go
func (b *Bridge) Start(ctx context.Context) error {
    // 🎯 Subscribe to broadcast topic using typed topic
    broadcastTopic := wsTopics.TopicHTMLBroadcast // Typed topic reference!

    err := b.subscriber.Subscribe(ctx, broadcastTopic.Name(), b.handleBroadcast)
    if err != nil {
        return fmt.Errorf("failed to subscribe to broadcast topic: %w", err)
    }

    return nil
}

func (b *Bridge) handleBroadcast(ctx context.Context, msg pubsub.Message) error {
    // 📡 Get all connected WebSocket clients
    clients := b.clients.GetAll()

    // 🚀 Send message to each client's WebSocket connection
    for _, client := range clients {
        client.SendMessage(msg.Payload) // Broadcasts to WebSocket!
    }
    return nil
}
```

**What happens here:**

- WebSocket bridge subscribes to `"ws.html.broadcast"` using **typed topic**
- When rendered HTML arrives, it broadcasts to **all connected WebSocket clients**
- Each client receives the HTML via their WebSocket connection

### **Step 4: Browser Receives and Displays**

```javascript
// Client-side JavaScript
websocket.onmessage = function (event) {
  // 🎉 Rendered HTML arrives at browser
  document.getElementById("chat-messages").innerHTML += event.data;
};
```

**What happens here:**

- Browser receives rendered HTML via WebSocket
- HTML is directly inserted into the DOM
- User sees the new chat message appear instantly

## WebSocket Client Subscription Flow

The WebSocket subscription mechanism remains **completely unchanged**:

### **Client-Side Subscription**

```javascript
// 1. Client connects to WebSocket
const websocket = new WebSocket("ws://localhost:8080/ws/html");

// 2. Client subscribes to topics (unchanged API)
websocket.send(
  JSON.stringify({
    action: "subscribe",
    topic: "chat.messages", // Same topic name as defined in typed topics
  })
);

// 3. Client can publish messages (unchanged API)
websocket.send(
  JSON.stringify({
    action: "client.chat.message.new",
    topic: "client.chat.message.new",
    payload: {
      content: "Hello from WebSocket!",
    },
  })
);
```

### **Bridge Subscription Management**

```go
// internal/websocket/bridge.go

// Bridge maintains subscription map (unchanged logic)
type topicManager struct {
    sync.RWMutex
    subscriptions map[string]map[string]struct{} // topic -> clientID -> struct{}
}

// Handle subscription messages (unchanged)
func (b *Bridge) handleSubscription(client *Client, msg SubscribeMessage) {
    topic := msg.Topic

    switch msg.Action {
    case "subscribe":
        b.subscribeClient(client.ID, topic)
        slog.Info("Client subscribed to topic", "clientID", client.ID, "topic", topic)

    case "unsubscribe":
        b.unsubscribeClient(client.ID, topic)
        slog.Info("Client unsubscribed from topic", "clientID", client.ID, "topic", topic)
    }
}

// Track client subscriptions (unchanged)
func (b *Bridge) subscribeClient(clientID, topic string) {
    b.topics.Lock()
    defer b.topics.Unlock()

    if _, exists := b.topics.subscriptions[topic]; !exists {
        b.topics.subscriptions[topic] = make(map[string]struct{})
    }
    b.topics.subscriptions[topic][clientID] = struct{}{}
}
```

## Topic Name Mapping: The Key Connection

The **magic** is in how topic names connect the type-safe system to WebSocket subscriptions:

```go
// 🎯 Typed topic definition (compile-time safe)
var TopicMessages = topicmgr.DefineModule(topicmgr.TopicConfig{
    Name: "chat.messages", // ← This string is what WebSocket clients subscribe to
    Module: "chat",
    Description: "Broadcasts a rendered chat message to all clients",
    // ...
})

// 🎯 Server-side usage (compile-time safe)
publisher.Publish(ctx, pubsub.Message{
    Topic: TopicMessages.Name(), // ← Returns "chat.messages"
    Payload: data,
})

// 🎯 WebSocket client subscription (same string, centrally managed)
websocket.send(JSON.stringify({
    action: "subscribe",
    topic: "chat.messages" // ← Same string, but now type-safe on server!
}));
```

## Message Broadcasting Types

The system supports different types of WebSocket broadcasting:

### **1. Broadcast Messages (All Clients)**

```go
// Publish to all HTML WebSocket clients
publisher.Publish(ctx, pubsub.Message{
    Topic:   wsTopics.TopicHTMLBroadcast.Name(), // "ws.html.broadcast"
    Payload: renderedHTML,
})

// Publish to all Data WebSocket clients
publisher.Publish(ctx, pubsub.Message{
    Topic:   wsTopics.TopicDataBroadcast.Name(), // "ws.data.broadcast"
    Payload: jsonData,
})
```

### **2. Direct Messages (Specific Client)**

```go
// Send to specific HTML client
publisher.Publish(ctx, pubsub.Message{
    Topic:   wsTopics.TopicHTMLDirect.Name(), // "ws.html.direct"
    Payload: renderedHTML,
    Metadata: map[string]string{
        "recipient_id": "user123", // Target specific user
    },
})
```

### **3. Topic-Based Messages (Subscribed Clients Only)**

```go
// Only clients subscribed to "chat.messages" receive this
publisher.Publish(ctx, pubsub.Message{
    Topic:   "chat.messages", // Custom topic
    Payload: data,
})
```

## Benefits of the New System

### **✅ Same Functionality, Enhanced Safety**

| Aspect                  | Old System        | New System      |
| ----------------------- | ----------------- | --------------- |
| **WebSocket API**       | Unchanged         | ✅ Unchanged    |
| **Broadcasting**        | Works             | ✅ Works        |
| **Topic Subscription**  | Works             | ✅ Works        |
| **Compile-time Safety** | ❌ Magic strings  | ✅ Typed topics |
| **Documentation**       | ❌ Scattered      | ✅ Centralized  |
| **Discoverability**     | ❌ Hard to find   | ✅ CLI tools    |
| **Validation**          | ❌ Runtime errors | ✅ Compile-time |

### **✅ Enhanced Developer Experience**

```go
// ❌ Old way - error-prone magic strings
publisher.Publish(ctx, pubsub.Message{
    Topic: "chat.mesages", // Typo! Runtime error
    Payload: data,
})

// ✅ New way - compile-time safe
publisher.Publish(ctx, pubsub.Message{
    Topic: chat.TopicMessages.Name(), // Typo impossible, IDE autocomplete
    Payload: data,
})
```

### **✅ Better Debugging and Monitoring**

```bash
# List all topics with metadata
$ go run cmd/topics/main.go list

NAME                     SCOPE      MODULE    DESCRIPTION
ws.html.broadcast        framework            Broadcast HTML to all clients
chat.messages           module     chat      Rendered chat messages
wargame.event.damage    module     wargame   Damage events in game

# Get detailed topic information
$ go run cmd/topics/main.go get chat.messages

Name:        chat.messages
Scope:       module
Module:      chat
Description: Broadcasts a rendered chat message to all clients
Pattern:     chat.messages
Example:     chat.messages
Metadata:
  routing_type: broadcast
  content_type: rendered_html
```

## Summary

The new topic management system **preserves all existing WebSocket functionality** while adding:

- **🛡️ Compile-time Safety**: No more typos in topic names
- **📚 Rich Documentation**: Every topic has description, examples, metadata
- **🔍 Discoverability**: CLI tools to explore and debug topics
- **🏗️ Better Organization**: Framework vs module scoping
- **🚀 Enhanced DX**: IDE autocompletion and refactoring support

**The WebSocket broadcasting flow works exactly the same** - clients subscribe to topics, messages are published to those topics, and the WebSocket bridge broadcasts to subscribed clients. The only difference is that server-side code now uses strongly-typed topic references instead of magic strings, making the system much more maintainable and reliable.
