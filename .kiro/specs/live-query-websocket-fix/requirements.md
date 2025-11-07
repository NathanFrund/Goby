# Requirements Document

## Introduction

The live query service in `internal/database/live_query.go` needs to be fixed to properly work with SurrealDB Go SDK v1.0.0 using WebSocket-based live queries. The current implementation has incomplete WebSocket notification handling and relies on placeholder code that doesn't actually receive database change notifications. The service must deliver real-time database changes to modules (like the announcer module) without polling, using proper WebSocket communication.

## Glossary

- **LiveQueryService**: The Go service that manages SurrealDB live query subscriptions and delivers change notifications to handlers
- **SurrealDB**: The database system that provides native live query support via WebSocket connections
- **WebSocket**: A persistent bidirectional communication protocol used by SurrealDB for real-time notifications
- **Subscription**: An active live query registration that monitors a table or query for changes
- **Handler**: A callback function provided by modules to process database change notifications
- **LIVE SELECT**: SurrealDB's SQL syntax for creating live queries that push changes to clients
- **Notification**: A message from SurrealDB containing information about a database change (CREATE, UPDATE, DELETE)
- **Module**: Application components (like announcer, chat) that use the LiveQueryService to react to database changes

## Requirements

### Requirement 1

**User Story:** As a module developer, I want to subscribe to database table changes, so that my module can react to data modifications in real-time

#### Acceptance Criteria

1. WHEN a module calls Subscribe with a table name and handler, THE LiveQueryService SHALL establish a WebSocket-based live query with SurrealDB
2. WHEN a record is created in the subscribed table, THE LiveQueryService SHALL invoke the handler with action "CREATE" and the record data
3. WHEN a record is updated in the subscribed table, THE LiveQueryService SHALL invoke the handler with action "UPDATE" and the record data
4. WHEN a record is deleted from the subscribed table, THE LiveQueryService SHALL invoke the handler with action "DELETE" and the record data
5. WHERE a filter is provided with WHERE clause, THE LiveQueryService SHALL only deliver notifications for records matching the filter criteria

### Requirement 2

**User Story:** As a module developer, I want to subscribe using custom SurrealQL queries, so that I can control which fields are monitored and apply complex filtering

#### Acceptance Criteria

1. WHEN a module calls SubscribeQuery with a LIVE SELECT query, THE LiveQueryService SHALL execute the query and establish the live query subscription
2. WHERE the query includes field selection (e.g., "LIVE SELECT id, email FROM user"), THE LiveQueryService SHALL deliver notifications containing only the specified fields
3. WHERE the query includes parameters, THE LiveQueryService SHALL properly bind the parameters before executing the query
4. IF the query syntax is invalid, THEN THE LiveQueryService SHALL return an error without creating a subscription

### Requirement 3

**User Story:** As a module developer, I want to unsubscribe from live queries, so that I can clean up resources when my module shuts down or no longer needs updates

#### Acceptance Criteria

1. WHEN a module calls Unsubscribe with a subscription ID, THE LiveQueryService SHALL send a KILL command to SurrealDB to terminate the live query
2. WHEN a subscription is unsubscribed, THE LiveQueryService SHALL stop invoking the handler for that subscription
3. WHEN a subscription is unsubscribed, THE LiveQueryService SHALL release all resources associated with that subscription
4. IF Unsubscribe is called with an invalid subscription ID, THEN THE LiveQueryService SHALL return without error

### Requirement 4

**User Story:** As a system operator, I want the live query service to use WebSocket connections efficiently, so that the system scales well with many subscriptions

#### Acceptance Criteria

1. THE LiveQueryService SHALL use the existing database connection's WebSocket for live query notifications
2. THE LiveQueryService SHALL NOT create separate WebSocket connections for each subscription
3. THE LiveQueryService SHALL properly multiplex multiple live query subscriptions over a single WebSocket connection
4. WHEN the WebSocket connection is lost, THE LiveQueryService SHALL leverage the Connection's REWS reconnection logic to restore subscriptions

### Requirement 5

**User Story:** As a developer debugging issues, I want proper error handling and logging, so that I can diagnose problems with live queries

#### Acceptance Criteria

1. WHEN a handler function panics, THE LiveQueryService SHALL recover from the panic and log the error without stopping other subscriptions
2. WHEN a live query fails to establish, THE LiveQueryService SHALL return a descriptive error to the caller
3. THE LiveQueryService SHALL log subscription lifecycle events (created, terminated) at INFO level
4. THE LiveQueryService SHALL log notification processing at DEBUG level to avoid log spam
5. IF SurrealDB returns an error status for a live query, THEN THE LiveQueryService SHALL return an error containing the status message

### Requirement 6

**User Story:** As a quality engineer, I want comprehensive tests for the live query service, so that I can verify it works correctly with real database operations

#### Acceptance Criteria

1. THE test suite SHALL verify that CREATE operations trigger handler invocations with correct action and data
2. THE test suite SHALL verify that UPDATE operations trigger handler invocations with correct action and data
3. THE test suite SHALL verify that DELETE operations trigger handler invocations with correct action and data
4. THE test suite SHALL verify that filtered subscriptions only receive matching notifications
5. THE test suite SHALL verify that unsubscribed handlers no longer receive notifications
6. THE test suite SHALL verify that multiple concurrent subscriptions work independently
7. THE test suite SHALL use real SurrealDB instances, not mocks, to validate WebSocket behavior
