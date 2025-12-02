# Educational Example Modules

This directory contains **educational reference implementations** that demonstrate various features and patterns of the Goby framework. These modules are meant for learning and as starting points for your own implementations, not for direct production use.

## Example Modules

### [chat/](file:///Users/Nathan/Documents/Development/EarlyPrototyping/Go/Goby/internal/modules/examples/chat)
Demonstrates real-time WebSocket communication and pub/sub patterns.

**Key Features:**
- WebSocket-based real-time messaging
- Pub/sub event system
- Topic-based message routing
- Presence tracking integration
- HTMX for dynamic UI updates

**Route:** `/app/chat`

---

### [profile/](file:///Users/Nathan/Documents/Development/EarlyPrototyping/Go/Goby/internal/modules/examples/profile)
Demonstrates file upload handling and user authentication patterns.

**Key Features:**
- File upload with `FileRepository`
- User authentication and context handling
- Template rendering with layouts
- HTMX integration for dynamic updates
- Profile picture management

**Route:** `/app/profile`

---

### [wargame/](file:///Users/Nathan/Documents/Development/EarlyPrototyping/Go/Goby/internal/modules/examples/wargame)
Demonstrates complex state management and scripting integration.

**Key Features:**
- Tengo scripting engine integration
- Complex game state management
- Event-driven architecture
- Real-time state updates via WebSocket
- Topic-based game events

**Route:** `/app/wargame`

---

## Using These Examples

These modules are designed to be:
1. **Reference implementations** - Study the code to understand framework patterns
2. **Starting templates** - Copy and modify for your own features
3. **Learning resources** - See how different framework components work together

## Production Modules

For production-ready modules, see the parent [modules/](file:///Users/Nathan/Documents/Development/EarlyPrototyping/Go/Goby/internal/modules) directory. Currently includes:
- `announcer/` - Production module for live query announcements
