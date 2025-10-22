# Topic Management System Guide

## Why We Built This System

### The Problem with Magic Strings

Before the new topic management system, Goby used magic strings for topic-based messaging:

```go
// ❌ Old way - prone to typos and runtime errors
publisher.Publish(ctx, pubsub.Message{
    Topic: "chat.messages", // Magic string - no compile-time safety
    Payload: data,
})

subscriber.Subscribe(ctx, "chat.mesages", handler) // Typo! Runtime error
```

**Problems with the old approach:**

- **Runtime Errors**: Typos in topic names only discovered at runtime
- **No Discoverability**: Hard to find what topics exist in the system
- **Poor Documentation**: No centralized place to understand topic purposes
- **Inconsistent Usage**: Developers could bypass the registry entirely
- **No Validation**: No way to enforce naming conventions or validate usage

### The Solution: Strongly-Typed Topics

The new system provides compile-time safety and rich metadata:

```go
// ✅ New way - compile-time safe and self-documenting
publisher.Publish(ctx, pubsub.Message{
    Topic: chat.TopicMessages.Name(), // Strongly typed - typos caught at compile time
    Payload: data,
})

subscriber.Subscribe(ctx, chat.TopicMessages.Name(), handler) // Same topic guaranteed
```

## Core Concepts

### 1. Framework vs Module Scoping

**Framework Topics** - Core system functionality:

- WebSocket routing (`ws.html.broadcast`, `ws.data.direct`)
- Presence tracking (`presence.user.online`, `presence.user.offline`)
- Server lifecycle events

**Module Topics** - Application-specific features:

- Chat messages (`chat.messages`, `client.chat.message.new`)
- Game events (`wargame.event.damage`, `wargame.state.update`)
- Business logic events

### 2. Rich Metadata

Every topic includes comprehensive information:

- **Name**: Unique identifier
- **Description**: Human-readable purpose
- **Module**: Owning module (for module topics)
- **Scope**: Framework or module level
- **Pattern**: Routing pattern with placeholders
- **Example**: Sample usage or payload
- **Metadata**: Additional structured information

## How to Use the System

### Defining Framework Topics

Framework services define their topics in their package:

```go
// internal/websocket/topics.go
package websocket

import "github.com/nfrund/goby/internal/topicmgr"

var (
    TopicHTMLBroadcast = topicmgr.DefineFramework(topicmgr.TopicConfig{
        Name:        "ws.html.broadcast",
        Description: "Broadcast HTML content to all connected HTML WebSocket clients",
        Pattern:     "ws.html.broadcast",
        Example:     "ws.html.broadcast",
        Metadata: map[string]interface{}{
            "endpoint_type": "html",
            "routing_type":  "broadcast",
        },
    })
)

// Register all framework topics
func RegisterTopics() error {
    manager := topicmgr.Default()
    return manager.Register(TopicHTMLBroadcast)
}
```

### Defining Module Topics

Modules define their topics in a dedicated `topics` subpackage:

```go
// internal/modules/chat/topics/topics.go
package topics

import "github.com/nfrund/goby/internal/topicmgr"

var (
    TopicNewMessage = topicmgr.DefineModule(topicmgr.TopicConfig{
        Name:        "client.chat.message.new",
        Module:      "chat",
        Description: "A new chat message sent by a client",
        Pattern:     "client.chat.message.new",
        Example:     `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
        Metadata: map[string]interface{}{
            "source":       "client",
            "message_type": "new",
            "payload_fields": []string{"content", "user"},
        },
    })
)

func RegisterTopics() error {
    manager := topicmgr.Default()
    return manager.Register(TopicNewMessage)
}
```

### Using Topics in Publishers

```go
// internal/modules/chat/handler.go
func (h *Handler) MessagePost(c echo.Context) error {
    // ... process message ...

    // Use the strongly-typed topic
    h.publisher.Publish(c.Request().Context(), pubsub.Message{
        Topic:   topics.TopicMessages.Name(), // Compile-time safe!
        UserID:  user.Email,
        Payload: payload,
    })

    return c.NoContent(http.StatusOK)
}
```

### Using Topics in Subscribers

```go
// internal/modules/chat/subscriber.go
func (cs *ChatSubscriber) Start(ctx context.Context) {
    // Subscribe using typed topics
    go func() {
        err := cs.subscriber.Subscribe(ctx, topics.TopicNewMessage.Name(), cs.handleChatMessage)
        if err != nil && err != context.Canceled {
            slog.Error("Chat message subscriber stopped with error", "error", err)
        }
    }()
}
```

### Module Registration

Register topics during module initialization:

```go
// internal/modules/chat/module.go
func (m *ChatModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
    // Register chat module topics
    if err := topics.RegisterTopics(); err != nil {
        return err
    }

    // ... rest of module setup ...
}
```

## Advanced Features

### Topic Discovery

```go
manager := topicmgr.Default()

// List all topics
allTopics := manager.List()

// Find topics by module
chatTopics := manager.ListByModule("chat")

// Find framework topics only
frameworkTopics := manager.ListFrameworkTopics()

// Search by pattern
wsTopics := manager.FindTopics("ws.*")

// Check if topic exists
exists := manager.CheckTopicExists("chat.messages")
```

### Validation

```go
manager := topicmgr.Default()

// Validate topic name
err := manager.ValidateTopicName("invalid..name") // Returns error

// Validate topic configuration
config := topicmgr.TopicConfig{
    Name: "test.topic",
    Module: "test",
    Description: "Test topic",
    // ... other fields
}
err = manager.ValidateConfiguration(config)

// Validate topic access
err = manager.ValidateTopicAccess("chat.messages", "chat", "subscriber")
```

### Statistics and Monitoring

```go
manager := topicmgr.Default()

// Get comprehensive statistics
stats := manager.GetStats()
fmt.Printf("Total topics: %d\n", stats.RegistryStats.TotalTopics)
fmt.Printf("Framework topics: %d\n", stats.RegistryStats.FrameworkTopics)
fmt.Printf("Module breakdown: %+v\n", stats.RegistryStats.ModuleBreakdown)

// Check manager status
if manager.IsStarted() {
    fmt.Printf("Manager uptime: %v\n", stats.Uptime)
}
```

## CLI Tools

### List All Topics

```bash
go run cmd/topics/main.go list
```

Output:

```
NAME                     SCOPE      MODULE    DESCRIPTION                              EXAMPLE
ws.html.broadcast        framework            Broadcast HTML content to all clients   ws.html.broadcast
chat.messages           module     chat      Broadcasts rendered chat messages       chat.messages
wargame.event.damage    module     wargame   Damage event in wargame                 {"targetUnit":"Tank-01",...}
```

### Get Topic Details

```bash
go run cmd/topics/main.go get chat.messages
```

Output:

```
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

## Migration from Old System

### Before (Magic Strings)

```go
// ❌ Old way
var ClientMessageNew = topics.MustRegister(
    "client.chat.message.new",
    "A new chat message sent by a client.",
    "client.chat.message.new",
    `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
)

// Usage
publisher.Publish(ctx, pubsub.Message{
    Topic: "client.chat.message.new", // Magic string
    Payload: data,
})
```

### After (Strongly Typed)

```go
// ✅ New way
var TopicNewMessage = topicmgr.DefineModule(topicmgr.TopicConfig{
    Name:        "client.chat.message.new",
    Module:      "chat",
    Description: "A new chat message sent by a client",
    Pattern:     "client.chat.message.new",
    Example:     `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
    Metadata: map[string]interface{}{
        "source": "client",
        "payload_fields": []string{"content", "user"},
    },
})

// Usage
publisher.Publish(ctx, pubsub.Message{
    Topic: TopicNewMessage.Name(), // Compile-time safe
    Payload: data,
})
```

## Best Practices

### 1. Naming Conventions

**Framework Topics:**

- Use service prefix: `ws.`, `presence.`, `auth.`
- Follow pattern: `service.type.action`
- Examples: `ws.html.broadcast`, `presence.user.online`

**Module Topics:**

- Include module name: `chat.`, `wargame.`, `inventory.`
- Follow pattern: `module.entity.action` or `client.module.entity.action`
- Examples: `chat.messages`, `client.chat.message.new`

### 2. Rich Metadata

Always include comprehensive metadata:

```go
TopicUserAction = topicmgr.DefineModule(topicmgr.TopicConfig{
    Name:        "wargame.action",
    Module:      "wargame",
    Description: "Player action in wargame such as move, attack, or special abilities",
    Pattern:     "wargame.action",
    Example:     `{"playerID":"player123","action":"move","unitID":"tank-01"}`,
    Metadata: map[string]interface{}{
        "event_type": "player_action",
        "payload_fields": []string{"playerID", "action", "unitID", "target"},
        "valid_actions": []string{"move", "attack", "defend", "special"},
    },
})
```

### 3. Module Organization

Organize topics in dedicated packages:

```
internal/modules/chat/
├── topics/
│   └── topics.go          # All chat topics
├── handler.go             # Uses chat topics
├── subscriber.go          # Uses chat topics
└── module.go             # Registers topics
```

### 4. Registration Patterns

Register topics during module boot:

```go
func (m *ChatModule) Boot(ctx context.Context, g *echo.Group, reg *registry.Registry) error {
    // Register topics first
    if err := topics.RegisterTopics(); err != nil {
        return fmt.Errorf("failed to register chat topics: %w", err)
    }

    // Then set up subscribers and handlers
    // ...
}
```

### 5. Error Handling

Always handle topic registration errors:

```go
func RegisterTopics() error {
    manager := topicmgr.Default()

    topics := []topicmgr.Topic{
        TopicNewMessage,
        TopicMessages,
        TopicDirectMessage,
    }

    for _, topic := range topics {
        if err := manager.Register(topic); err != nil {
            return fmt.Errorf("failed to register topic %s: %w", topic.Name(), err)
        }
    }

    return nil
}
```

## Benefits Achieved

### ✅ Compile-Time Safety

- Typos in topic names caught during compilation
- IDE autocompletion for all available topics
- Refactoring support across the codebase

### ✅ Better Documentation

- Self-documenting topics with rich metadata
- Centralized topic discovery via CLI tools
- Clear examples and usage patterns

### ✅ Improved Maintainability

- Framework/module scoping prevents naming conflicts
- Consistent naming conventions enforced by validation
- Easy to find and understand topic relationships

### ✅ Enhanced Developer Experience

- No more hunting for magic strings in the codebase
- Clear separation between framework and application concerns
- Powerful discovery and debugging tools

### ✅ Production Ready

- Comprehensive validation and error handling
- Statistics and monitoring capabilities
- Graceful migration path from legacy systems

The new topic management system transforms Goby's messaging infrastructure from error-prone magic strings into a robust, type-safe, and well-documented system that scales with your application's complexity.
