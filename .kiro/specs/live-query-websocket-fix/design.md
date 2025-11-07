# Live Query Service Design

## Overview

This design refactors the `SurrealLiveQueryService` to properly implement WebSocket-based live queries using the SurrealDB Go SDK v1.0.0. The key insight is that SurrealDB v1.0.0 uses a different API for live queries compared to earlier versions. The service will:

1. Execute `LIVE SELECT` queries to establish live query subscriptions
2. Use the database connection's WebSocket to receive notifications
3. Parse and route notifications to the appropriate handler functions
4. Properly clean up subscriptions using `KILL` commands

## Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────┐
│                         Module                               │
│  (announcer, chat, etc.)                                     │
└────────────────┬────────────────────────────────────────────┘
                 │ Subscribe()
                 │ handler callback
                 ▼
┌─────────────────────────────────────────────────────────────┐
│              SurrealLiveQueryService                         │
│  ┌──────────────────────────────────────────────────────┐   │
│  │  subscriptions map[string]*subscriptionState         │   │
│  │  - id, table, handler, liveQueryID, cancel           │   │
│  └──────────────────────────────────────────────────────┘   │
└────────────────┬────────────────────────────────────────────┘
                 │ LIVE SELECT query
                 │ Listen for notifications
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                    DBConnection                              │
│  (manages WebSocket connection via REWS)                     │
└────────────────┬────────────────────────────────────────────┘
                 │ WebSocket
                 ▼
┌─────────────────────────────────────────────────────────────┐
│                      SurrealDB                               │
│  (sends live query notifications)                            │
└─────────────────────────────────────────────────────────────┘
```

### Key Design Decisions

1. **Use surrealdb.Query for LIVE SELECT**: Execute live queries using the standard `surrealdb.Query` function, which returns a live query UUID
2. **Leverage DB.Live() method**: Use the `DB.Live()` method from SurrealDB Go SDK v1.0.0 to receive notifications on a channel
3. **Single goroutine per subscription**: Each subscription runs a dedicated goroutine that listens to its notification channel
4. **Graceful cleanup**: Use context cancellation and KILL commands to properly terminate live queries

## Components and Interfaces

### LiveQueryService Interface

```go
type LiveQueryService interface {
    Subscribe(ctx context.Context, table string, filter *LiveQueryFilter, handler LiveQueryHandler) (*Subscription, error)
    SubscribeQuery(ctx context.Context, query string, params map[string]interface{}, handler LiveQueryHandler) (*Subscription, error)
    Unsubscribe(subID string) error
}
```

No changes to the public interface - maintains backward compatibility.

### SurrealLiveQueryService Implementation

```go
type SurrealLiveQueryService struct {
    db            DBConnection
    subscriptions sync.Map // map[string]*subscriptionState
}

type subscriptionState struct {
    id          string
    table       string
    handler     LiveQueryHandler
    active      bool
    cancel      context.CancelFunc
    query       string
    params      map[string]interface{}
    liveQueryID uuid.UUID // SurrealDB live query UUID
}
```

### Notification Flow

1. **Subscription Creation**:

   - Generate unique subscription ID
   - Build LIVE SELECT query
   - Execute query via `surrealdb.Query` to get live query UUID
   - Call `db.Live(liveQueryID)` to get notification channel
   - Start goroutine to listen on channel
   - Store subscription state

2. **Notification Processing**:

   - Goroutine blocks on channel receive
   - Parse notification structure from SurrealDB
   - Extract action (CREATE/UPDATE/DELETE) and data
   - Invoke handler in separate goroutine (with panic recovery)
   - Continue listening until context cancelled

3. **Cleanup**:
   - Cancel subscription context
   - Execute `KILL $liveQueryID` query
   - Remove from subscriptions map
   - Notification channel closes automatically

## Data Models

### SurrealDB Live Query Response

When executing a LIVE SELECT query, SurrealDB returns:

```go
type QueryResult struct {
    Status string
    Result uuid.UUID // The live query ID
}
```

### SurrealDB Notification Format

Notifications received on the channel have this structure:

```go
type LiveNotification struct {
    Action string      // "CREATE", "UPDATE", "DELETE", "CLOSE"
    Result interface{} // The record data
}
```

### Internal Subscription State

```go
type subscriptionState struct {
    id          string                 // Our internal subscription ID
    table       string                 // Table being monitored
    handler     LiveQueryHandler       // Callback function
    active      bool                   // Whether subscription is active
    cancel      context.CancelFunc     // Cancel function for cleanup
    query       string                 // Original LIVE SELECT query
    params      map[string]interface{} // Query parameters
    liveQueryID uuid.UUID              // SurrealDB's live query UUID
}
```

## Error Handling

### Error Categories

1. **Subscription Errors**:

   - Invalid handler (nil) → Return error immediately
   - Query execution failure → Return error with context
   - Live query establishment failure → Return error with SurrealDB status

2. **Runtime Errors**:

   - Handler panic → Recover, log error, continue processing other notifications
   - Channel closed unexpectedly → Log warning, mark subscription inactive
   - Context cancellation → Clean shutdown, no error logged

3. **Cleanup Errors**:
   - KILL command failure → Log warning but don't fail (query may already be dead)
   - Double unsubscribe → No-op, no error

### Error Logging Strategy

- **ERROR level**: Subscription creation failures, unrecoverable handler panics
- **WARN level**: KILL command failures, unexpected channel closures
- **INFO level**: Subscription lifecycle events (created, terminated)
- **DEBUG level**: Individual notifications received

## Testing Strategy

### Unit Tests

Not applicable - this service requires real database interaction for WebSocket behavior.

### Integration Tests

1. **TestSubscribeToTableChanges**:

   - Create subscription
   - Insert record
   - Verify handler receives CREATE notification
   - Update record
   - Verify handler receives UPDATE notification
   - Delete record
   - Verify handler receives DELETE notification

2. **TestSubscribeWithFilter**:

   - Create subscription with WHERE clause
   - Insert matching record → Verify notification received
   - Insert non-matching record → Verify no notification
   - Update record to match filter → Verify notification received

3. **TestSubscribeQuery**:

   - Create subscription with field selection
   - Insert record
   - Verify notification contains only selected fields

4. **TestMultipleConcurrentSubscriptions**:

   - Create 3 subscriptions to different tables
   - Trigger changes in all tables
   - Verify each handler receives only its table's notifications

5. **TestUnsubscribe**:

   - Create subscription
   - Unsubscribe
   - Trigger database change
   - Verify handler not invoked

6. **TestHandlerPanic**:

   - Create subscription with handler that panics
   - Trigger notification
   - Verify service recovers and continues
   - Verify other subscriptions unaffected

7. **TestNilHandler**:
   - Attempt to subscribe with nil handler
   - Verify error returned

### Test Infrastructure

- Use existing `setupTestDB` helper for database connection
- Use `testify/suite` for test organization
- Set reasonable timeouts (5-10 seconds) for notification delivery
- Use channels to synchronize between test and handler goroutines

## Implementation Notes

### SurrealDB Go SDK v1.0.0 API

Based on the SDK structure, the implementation will use:

```go
// Execute LIVE SELECT to get live query ID
results, err := surrealdb.Query[uuid.UUID](ctx, db, "LIVE SELECT * FROM user", nil)
liveQueryID := results[0].Result

// Get notification channel
notificationChan := db.Live(liveQueryID)

// Listen for notifications
for notification := range notificationChan {
    // Process notification
}
```

### WebSocket Connection Reuse

The service leverages the existing `DBConnection` which already manages:

- WebSocket connection via REWS
- Automatic reconnection
- Session restoration

The live query service doesn't need to manage WebSocket connections directly.

### Concurrency Considerations

- `sync.Map` for thread-safe subscription storage
- Each subscription has its own goroutine and context
- Handler invocations run in separate goroutines to prevent blocking
- Panic recovery in handler goroutines to prevent cascade failures

### Performance Characteristics

- **Memory**: O(n) where n = number of active subscriptions
- **Goroutines**: 1 per subscription + 1 per handler invocation (short-lived)
- **Network**: Single WebSocket connection shared across all subscriptions
- **Latency**: Near real-time (WebSocket push, no polling)

## Migration Path

The current implementation has placeholder code that doesn't work. The migration is:

1. Remove all polling-related code
2. Remove placeholder WebSocket listener functions
3. Implement proper `DB.Live()` channel listening
4. Update tests to verify real-time behavior
5. Update documentation to reflect WebSocket-only operation

No changes to the public API, so modules using the service don't need updates.
