# Implementation Plan

- [x] 1. Research SurrealDB Go SDK v1.0.0 live query API

  - Examine the surrealdb.go package to identify the correct methods for live queries
  - Determine the exact API for receiving notifications (DB.Live(), channels, etc.)
  - Document the notification message structure from SurrealDB
  - Verify how live query UUIDs are returned from LIVE SELECT queries
  - _Requirements: 1.1, 2.1, 4.1_

- [x] 2. Implement core subscription logic
- [x] 2.1 Refactor subscribeQuery method to use proper SurrealDB v1.0.0 API

  - Remove placeholder WebSocket listener code
  - Execute LIVE SELECT query using surrealdb.Query to get live query UUID
  - Call DB.Live() method to obtain notification channel
  - Store live query UUID in subscriptionState
  - _Requirements: 1.1, 2.1_

- [x] 2.2 Implement notification listener goroutine

  - Create goroutine that blocks on notification channel
  - Parse notification structure to extract action and data
  - Map SurrealDB action strings to LiveQueryAction constants
  - Invoke handler with parsed action and data
  - Handle channel closure and context cancellation
  - _Requirements: 1.2, 1.3, 1.4_

- [x] 2.3 Add panic recovery for handler invocations

  - Wrap handler calls in defer/recover
  - Log panics with subscription ID and error details
  - Ensure panic in one handler doesn't affect other subscriptions
  - _Requirements: 5.1_

- [x] 3. Implement subscription cleanup
- [x] 3.1 Update Unsubscribe method to properly kill live queries

  - Execute KILL command with live query UUID
  - Cancel subscription context to stop listener goroutine
  - Remove subscription from sync.Map
  - Handle errors from KILL command gracefully (log warning only)
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 3.2 Add cleanup on context cancellation

  - Ensure KILL command runs when subscription context is cancelled
  - Use separate timeout context for cleanup to avoid cancellation issues
  - Log cleanup operations at appropriate level
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 4. Update Subscribe and SubscribeQuery methods
- [x] 4.1 Refactor Subscribe to build correct LIVE SELECT queries

  - Build query with field selection if filter.Fields provided
  - Add WHERE clause if filter.Where provided
  - Properly bind filter.Params to query
  - Call subscribeQuery with built query
  - _Requirements: 1.1, 1.5, 2.2_

- [x] 4.2 Update SubscribeQuery to validate query syntax

  - Check that query starts with "LIVE SELECT"
  - Return descriptive error for invalid queries
  - Pass query and params to subscribeQuery
  - _Requirements: 2.1, 2.3, 2.4_

- [x] 5. Enhance error handling and logging
- [x] 5.1 Add structured logging for subscription lifecycle

  - Log subscription creation at INFO level with subscription ID and table
  - Log subscription termination at INFO level
  - Log notification processing at DEBUG level
  - _Requirements: 5.3, 5.4_

- [x] 5.2 Improve error messages for subscription failures

  - Include SurrealDB status in error messages
  - Add context about which query failed
  - Return wrapped errors with NewDBError where appropriate
  - _Requirements: 5.2, 5.5_

- [x] 6. Update integration tests
- [x] 6.1 Fix TestSubscribeToTableChanges to verify real-time notifications

  - Remove artificial delays where possible
  - Use channels to synchronize handler invocations with test assertions
  - Verify CREATE action and data correctness
  - Add UPDATE and DELETE test cases
  - _Requirements: 6.1, 6.2, 6.3_

- [x] 6.2 Enhance TestSubscribeQueryWithCustomQuery

  - Verify field selection works (only specified fields in notification)
  - Test parameter binding with WHERE clause
  - Verify filtered notifications (matching vs non-matching records)
  - _Requirements: 6.4, 2.2, 2.3_

- [x] 6.3 Add TestMultipleConcurrentSubscriptions

  - Create 3 subscriptions to different tables
  - Trigger changes in all tables concurrently
  - Verify each handler receives only its table's notifications
  - Verify no cross-contamination between subscriptions
  - _Requirements: 6.6_

- [x] 6.4 Add TestHandlerPanic

  - Create subscription with handler that panics
  - Trigger notification
  - Verify service logs panic and continues
  - Create second subscription and verify it still works
  - _Requirements: 5.1_

- [x] 6.5 Update TestUnsubscribeRemovesSubscription

  - Verify KILL command is executed
  - Verify handler not invoked after unsubscribe
  - Test double unsubscribe doesn't error
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 6.5_

- [ ] 7. Remove obsolete code
- [x] 7.1 Delete unused helper methods

  - Remove listenForWebSocketLiveQueryNotifications (old implementation)
  - Remove registerResponseChannelWithDB (not needed)
  - Remove unregisterResponseChannel (not needed)
  - Remove processLiveQueryNotificationFromDB (not needed)
  - Remove checkForLiveQueryUpdates if it exists (polling code)
  - _Requirements: 1.1, 4.2_

- [x] 7.2 Clean up documentation
  - Remove references to polling fallback
  - Update comments to reflect WebSocket-only operation
  - Update live_query_fix_documentation.md with final solution
  - _Requirements: 5.3_
