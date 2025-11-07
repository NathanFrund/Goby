# SurrealDB Go SDK v1.0.0 Live Query API Research

## Overview

This document contains research findings on the SurrealDB Go SDK v1.0.0 live query API based on examining the SDK documentation and current implementation.

## Key API Methods

### 1. DB.LiveNotifications()

```go
func (db *DB) LiveNotifications(liveQueryID string) (chan connection.Notification, error)
```

- **Purpose**: Returns a channel that receives live query notifications for a specific live query ID
- **Parameters**:
  - `liveQueryID`: The UUID string returned from executing a LIVE SELECT query
- **Returns**:
  - A channel of `connection.Notification` structs
  - An error if the live query ID is invalid or the channel cannot be created

### 2. DB.CloseLiveNotifications()

```go
func (db *DB) CloseLiveNotifications(liveQueryID string) error
```

- **Purpose**: Closes the notification channel for a specific live query
- **Parameters**:
  - `liveQueryID`: The UUID string of the live query to close
- **Returns**:
  - An error if the operation fails

### 3. surrealdb.Query()

```go
func Query[TResult any](ctx context.Context, db *DB, sql string, vars map[string]any) (*[]QueryResult[TResult], error)

type QueryResult[T any] struct {
    Status string      `json:"status"`
    Time   string      `json:"time"`
    Result T           `json:"result"`
    Error  *QueryError `json:"-"`
}
```

- **Purpose**: Executes SurrealQL queries including LIVE SELECT
- **Usage for Live Queries**: Execute `LIVE SELECT * FROM table` to establish a live query
- **Returns**: Slice of QueryResult where Result field contains the live query UUID as a string

## Notification Structure

### connection.Notification

```go
type Notification struct {
    ID     *models.UUID `json:"id,omitempty"`
    Action Action       `json:"action"`
    Result interface{}  `json:"result"`
}
```

**Fields**:

- `ID`: Optional UUID identifying the notification
- `Action`: The type of change (CREATE, UPDATE, DELETE, CLOSE)
- `Result`: The actual data payload (record that changed)

### connection.Action

```go
type Action string

const (
    CreateAction Action = "CREATE"
    UpdateAction Action = "UPDATE"
    DeleteAction Action = "DELETE"
    CloseAction  Action = "CLOSE"
)
```

## Live Query Workflow

### Step 1: Execute LIVE SELECT Query

```go
// QueryResult structure:
// type QueryResult[T any] struct {
//     Status string      `json:"status"`
//     Time   string      `json:"time"`
//     Result T           `json:"result"`
//     Error  *QueryError `json:"-"`
// }

results, err := surrealdb.Query[string](ctx, db, "LIVE SELECT * FROM table", nil)
if err != nil {
    return err
}

// Extract the live query UUID from the result
var liveQueryID string
if len(*results) > 0 {
    result := (*results)[0]
    if result.Status == "OK" {
        // The Result field contains the live query UUID as a string
        liveQueryID = result.Result
    }
}
```

### Step 2: Get Notification Channel

```go
notificationChan, err := db.LiveNotifications(liveQueryID)
if err != nil {
    return err
}
```

### Step 3: Listen for Notifications

```go
for notification := range notificationChan {
    switch notification.Action {
    case connection.CreateAction:
        // Handle CREATE
    case connection.UpdateAction:
        // Handle UPDATE
    case connection.DeleteAction:
        // Handle DELETE
    case connection.CloseAction:
        // Handle CLOSE (live query terminated)
        return
    }

    // Process notification.Result (the actual data)
}
```

### Step 4: Cleanup

```go
// When done, close the notification channel
err := db.CloseLiveNotifications(liveQueryID)
if err != nil {
    log.Printf("Failed to close live notifications: %v", err)
}

// Kill the live query on the database
_, err = surrealdb.Query[interface{}](ctx, db, fmt.Sprintf("KILL %s", liveQueryID), nil)
```

## Key Findings

1. **Live Query UUID**: The LIVE SELECT query returns a UUID string that identifies the live query session
2. **Notification Channel**: Use `DB.LiveNotifications(liveQueryID)` to get a channel for receiving updates
3. **Automatic WebSocket**: The SDK handles WebSocket connection internally when using LiveNotifications
4. **Structured Notifications**: Notifications come as typed structs with Action and Result fields
5. **Cleanup Required**: Both `CloseLiveNotifications()` and `KILL` query should be called for proper cleanup
6. **Channel-Based**: The API is channel-based, making it idiomatic Go for concurrent notification handling

## Current Implementation Issues

Based on the existing `internal/database/live_query.go`:

1. **Missing LiveNotifications Call**: The current code doesn't call `db.LiveNotifications(liveQueryID)`
2. **No Notification Channel**: There's no actual channel receiving notifications from SurrealDB
3. **Placeholder Logic**: The `listenForLiveQueryNotifications` function is a placeholder with no real implementation
4. **UUID Extraction**: The code attempts to extract the live query ID but doesn't use it correctly

## Correct Implementation Pattern

```go
// 1. Execute LIVE SELECT and get UUID
results, err := surrealdb.Query[string](ctx, dbConn, query, params)
if err != nil {
    return err
}

// Extract UUID from QueryResult
var liveQueryID string
if len(*results) > 0 && (*results)[0].Status == "OK" {
    liveQueryID = (*results)[0].Result // Result is the UUID string
}

// 2. Get notification channel from SDK
notificationChan, err := dbConn.LiveNotifications(liveQueryID)
if err != nil {
    return err
}

// 3. Listen on the channel
go func() {
    for notification := range notificationChan {
        // Process notification
        processNotification(notification)
    }
}()

// 4. Cleanup when done
defer func() {
    dbConn.CloseLiveNotifications(liveQueryID)
    surrealdb.Query[interface{}](ctx, dbConn, fmt.Sprintf("KILL %s", liveQueryID), nil)
}()
```

## References

- SurrealDB Go SDK: github.com/surrealdb/surrealdb.go v1.0.0
- Connection Package: github.com/surrealdb/surrealdb.go/pkg/connection
- Models Package: github.com/surrealdb/surrealdb.go/pkg/models
