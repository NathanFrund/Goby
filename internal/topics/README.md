# Topic Registry System

The topic registry provides a centralized way to define, document, and discover message topics used throughout the application. It ensures all topics are properly registered and discoverable at runtime.

## Key Features

- **Type-safe topic definitions**
- **Self-documenting topics** with descriptions and examples
- **Compile-time registration** using `MustRegister`
- **Thread-safe** topic management
- **Pattern-based** topic generation
- **Runtime discovery** of all registered topics

## Defining Topics

Each module should define its topics in a `topics.go` file using `topics.MustRegister`:

```go
package yourmodule

import "github.com/nfrund/goby/internal/topics"

var (
    // NewChatMessage is published when a new chat message is created
    NewChatMessage = topics.MustRegister(
        "chat_message",
        "Published when a new chat message is created",
        "chat.messages.{room_id}.new",
        "chat.messages.general.new",
    )
)
```

> **Note**: `MustRegister` panics if a topic with the same name is already registered, ensuring all topics are unique at startup.

## Using Topics

### Formatting Topics with Parameters

```go
// Format a topic with parameters
topic, err := chat.NewChatMessage.Format(map[string]string{
    "room_id": "general",
})
if err != nil {
    // Handle error (e.g., missing required parameters)
}
// Result: "chat.messages.general.new"
```

### Discovering Topics at Runtime

```go
// List all registered topics
for _, topic := range topics.List() {
    fmt.Printf("%s: %s\n", topic.Name(), topic.Description())
    fmt.Printf("  Pattern: %s\n", topic.Pattern())
    fmt.Printf("  Example: %s\n", topic.Example())
}

// Get a specific topic
if topic, exists := topics.Get("chat_message"); exists {
    fmt.Printf("Found topic: %s\n", topic.Description())
}
```

### WebSocket Integration

Topics can be used with the WebSocket bridge for real-time communication:

```go
// Subscribe to a topic from client-side
ws.send(JSON.stringify({
    action: "subscribe",
    topic: "chat.messages.general"
}));

// Publish to a topic from server-side
err := publisher.Publish(ctx, pubsub.Message{
    Topic:   "chat.messages.general",
    Payload: []byte(`{"user":"alice","message":"Hello!"}`),
})

## Topic Naming Conventions

1. **Names**: Use lowercase with dots (e.g., `chat.message.new`, `user.status.update`)
2. **Patterns**: Use dot notation with `{parameters}` (e.g., `chat.messages.{room_id}.new`)
3. **Descriptions**: Be clear and concise, explain when the topic is used
4. **Examples**: Show a complete, realistic example of the topic in use
5. **Segmentation**: Use consistent prefixes for related topics (e.g., `chat.*`, `user.*`)

## Best Practices

1. **Use MustRegister**: Always use `topics.MustRegister()` for static topic definitions
2. **Document Thoroughly**: Include clear descriptions and realistic examples
3. **Reuse Topics**: Check for existing topics before creating new ones
4. **Parameter Validation**: Implement `Validate()` for topics with parameters
5. **Thread Safety**: The registry is thread-safe, but be careful with topic instances
6. **Module Boundaries**: Keep topic definitions close to where they're used

## CLI Reference

The topics CLI provides a way to explore registered topics:

```bash
# List all topics
go run cmd/topics/main.go list

# Get details about a specific topic
go run cmd/topics/main.go get chat_message
```

## Example: Chat Module

Here's a complete example of the chat module's topic definitions:

```go
package chat

import "github.com/nfrund/goby/internal/topics"

var (
    // ClientMessageNew is published when a client sends a new chat message
    ClientMessageNew = topics.MustRegister(
        "client.chat.message.new",
        "A new chat message sent by a client",
        "client.chat.message.new",
        `{"action":"client.chat.message.new","payload":{"content":"Hello!"}}`,
    )

    // Messages broadcasts rendered chat messages to all clients
    Messages = topics.MustRegister(
        "chat.messages",
        "Broadcasts a rendered chat message to all clients",
        "chat.messages",
        "chat.messages",
    )

    // Direct sends a rendered direct message to a specific user
    Direct = topics.MustRegister(
        "chat.direct",
        "Sends a rendered direct message to a specific user",
        "chat.direct.*",
        "chat.direct.user123",
    )
)
```

### Using the Topics

```go
// Subscribing to a topic
err := subscriber.Subscribe(ctx, Messages.Name(), func(ctx context.Context, msg pubsub.Message) {
    // Handle incoming message
    fmt.Printf("New message: %s\n", string(msg.Payload))
})

// Publishing to a topic
err := publisher.Publish(ctx, pubsub.Message{
    Topic:   Messages.Name(),
    Payload: []byte(`{"user":"alice","message":"Hello!"}`),
})
)

var (
    // NewMessage is published when a new message is created
    NewMessage = topics.Topic{
        Name:        "chat_message",
        Description: "Published when a new chat message is created",
        Pattern:     "chat.messages.{room_id}.new",
        Example:     "chat.messages.general.new",
    }

    // UserJoined is published when a user joins a room
    UserJoined = topics.Topic{
        Name:        "chat_user_joined",
        Description: "Published when a user joins a chat room",
        Pattern:     "chat.rooms.{room_id}.user_joined",
        Example:     "chat.rooms.general.user_joined",
    }
)

func init() {
    topics.Register(NewMessage)
    topics.Register(UserJoined)
}
```
