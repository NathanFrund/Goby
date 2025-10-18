# Topic Registry System

The topic registry provides a centralized way to define, document, and discover message topics used throughout the application.

## Key Features

- **Type-safe topic definitions**
- **Self-documenting topics**
- **IDE discoverability**
- **Runtime inspection**
- **Pattern-based topic generation**

## Defining Topics

Each module should define its topics in a `topics.go` file:

```go
package yourmodule

import "github.com/nfrund/goby/internal/topics"

var (
    // NewChatMessage is published when a new chat message is created
    NewChatMessage = topics.Topic{
        Name:        "chat_message",
        Description: "Published when a new chat message is created",
        Pattern:     "chat.messages.{room_id}.new",
        Example:     "chat.messages.general.new",
    }
)

func init() {
    topics.Register(NewChatMessage)
}
```

## Using Topics

### Formatting Topics with Parameters

```go
topic := chat.NewChatMessage.Format(map[string]string{
    "room_id": "general",
})
// Result: "chat.messages.general.new"
```

### Listing All Topics

```go
allTopics := topics.List()
for _, topic := range allTopics {
    fmt.Printf("%s: %s\n", topic.Name, topic.Description)
}
```

### Looking Up a Topic by Name

```go
if topic, exists := topics.Get("chat_message"); exists {
    fmt.Printf("Found topic: %s\n", topic.Description)
}
```

## Topic Naming Conventions

1. Use lowercase with underscores for topic names (e.g., `chat_message`, `user_status_update`)
2. Use dot notation for patterns (e.g., `chat.messages.{room_id}.new`)
3. Make descriptions clear and concise
4. Include examples that show typical usage

## Best Practices

1. **Register Early**: Register all topics during application startup
2. **Document Thoroughly**: Always include a description and example
3. **Reuse Existing Topics**: Check for existing topics before creating new ones
4. **Keep Patterns Simple**: Avoid overly complex topic patterns
5. **Use Parameters Sparingly**: Keep the number of parameters in patterns minimal

## CLI Reference

The topics CLI provides a way to explore registered topics:

```bash
# List all topics
go run cmd/topics/main.go list

# Get details about a specific topic
go run cmd/topics/main.go get chat_message
```

## Example: Chat Module

Here's a complete example of how the chat module might use topics:

```go
package chat

import (
    "github.com/nfrund/goby/internal/topics"
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
