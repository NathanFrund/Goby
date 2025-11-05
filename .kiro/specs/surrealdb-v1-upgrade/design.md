# Design Document

## Overview

This design outlines the approach for upgrading the Goby application from SurrealDB Go SDK v0.9.0 to v1.0.0. The upgrade involves handling breaking changes in the SDK API while maintaining backward compatibility in the application's public interfaces. The design focuses on minimizing disruption to existing functionality while leveraging improvements in the new SDK version.

## Architecture

The upgrade affects three main architectural layers:

1. **Dependency Layer**: Go module dependencies and import statements
2. **Database Layer**: Connection management, query execution, and model serialization
3. **Domain Layer**: Domain models that use SurrealDB types

The upgrade strategy follows a phased approach:

- Phase 1: Update dependencies and resolve compilation errors
- Phase 2: Update connection management and authentication
- Phase 3: Update query execution patterns
- Phase 4: Validate and test all functionality

## Components and Interfaces

### 1. Dependency Management

**Current State:**

- Uses `github.com/surrealdb/surrealdb.go v0.9.0`
- Imports `surrealmodels "github.com/surrealdb/surrealdb.go/pkg/models"`

**Target State:**

- Upgrade to `github.com/surrealdb/surrealdb.go v1.0.0`
- Update import paths to match v1.0.0 structure
- Resolve any package reorganization changes

**Key Changes:**

- Update go.mod dependency version
- Review and update import statements across all files
- Handle any package path changes in v1.0.0

### 2. Connection Management (internal/database/connection.go)

**Current Implementation:**

```go
conn, err := surrealdb.FromEndpointURLString(ctx, c.cfg.GetDBURL())
authData := &surrealdb.Auth{
    Username: c.cfg.GetDBUser(),
    Password: c.cfg.GetDBPass(),
}
conn.SignIn(ctx, authData)
conn.Use(ctx, c.cfg.GetDBNs(), c.cfg.GetDBDb())
```

**Design Changes:**

- Research v1.0.0 connection creation patterns
- Update authentication method calls if changed
- Maintain the same public interface for Connection struct
- Preserve health monitoring and reconnection logic
- Update version check method if API changed

**Interface Preservation:**

- `WithConnection(ctx context.Context, fn func(*surrealdb.DB) error) error`
- `Connect(ctx context.Context) error`
- `Close(ctx context.Context) error`
- `IsHealthy() bool`

### 3. Query Execution (internal/database/executor.go)

**Current Implementation:**

```go
results, err := surrealdb.Query[[]T](ctx, db, query, params)
```

**Design Changes:**

- Update to v1.0.0 query execution patterns
- Maintain generic type support for query results
- Preserve error handling and logging
- Keep the same QueryExecutor interface contract

**Interface Preservation:**

- `Query(ctx context.Context, query string, params map[string]any) ([]T, error)`
- `QueryOne(ctx context.Context, query string, params map[string]any) (*T, error)`
- `Execute(ctx context.Context, query string, params map[string]any) error`

### 4. Domain Models

**Current Usage:**

- `surrealmodels.RecordID` for entity IDs
- `surrealmodels.CustomDateTime` for timestamps

**Design Changes:**

- Map v0.9.0 model types to v1.0.0 equivalents
- Ensure JSON serialization/deserialization remains consistent
- Maintain struct field tags and validation
- Preserve domain model interfaces (UserRepository, FileRepository)

## Data Models

### Model Type Mapping

The design includes a mapping strategy for SurrealDB model types:

1. **RecordID Handling:**

   - Verify v1.0.0 RecordID structure and methods
   - Ensure String() method compatibility
   - Maintain JSON marshaling behavior

2. **DateTime Handling:**

   - Map CustomDateTime to v1.0.0 equivalent
   - Preserve timezone and formatting behavior
   - Maintain compatibility with existing data

3. **Query Result Structures:**
   - Ensure query result parsing remains consistent
   - Handle any changes in result wrapper types
   - Maintain error response formats

## Error Handling

### Error Type Compatibility

**Current Error Handling:**

- Connection errors detected via `isConnectionError()`
- Database errors wrapped with `NewDBError()`
- Context cancellation and timeout handling

**Design Approach:**

- Map v0.9.0 error types to v1.0.0 equivalents
- Maintain error detection logic for reconnection
- Preserve error wrapping and logging patterns
- Ensure test assertions continue to work

### Reconnection Logic

The design preserves the existing reconnection strategy:

- Health check using `conn.Version(ctx)`
- Connection error detection patterns
- Automatic reconnection on failure
- Monitoring goroutine behavior

## Testing Strategy

### Test Compatibility

**Existing Test Patterns:**

- Integration tests with real SurrealDB connections
- Connection failure simulation
- Query execution validation
- Timeout behavior testing

**Upgrade Validation:**

1. **Compilation Tests:** Ensure all code compiles with v1.0.0
2. **Unit Tests:** Verify individual component behavior
3. **Integration Tests:** Validate end-to-end database operations
4. **Regression Tests:** Ensure existing functionality works identically

### Test Data Compatibility

- Verify existing test data structures work with v1.0.0
- Ensure TestUser and domain models serialize correctly
- Validate query parameter handling
- Test error scenarios and edge cases

## Migration Strategy

### Phase 1: Dependency Update

1. Update go.mod to SurrealDB v1.0.0
2. Run `go mod tidy` to resolve dependencies
3. Identify compilation errors from breaking changes

### Phase 2: Import and Type Updates

1. Update import statements to v1.0.0 paths
2. Map model types (RecordID, CustomDateTime) to v1.0.0 equivalents
3. Resolve type compatibility issues

### Phase 3: API Method Updates

1. Update connection creation methods
2. Update authentication patterns
3. Update query execution calls
4. Update health check methods

### Phase 4: Testing and Validation

1. Run existing test suite
2. Fix any test failures
3. Validate integration test behavior
4. Perform manual testing of key workflows

## Risk Mitigation

### Breaking Changes Approach

Since Goby is pre-v1.0, we can embrace fundamental reorganization changes to align with SurrealDB v1.0.0 best practices while maintaining the store pattern architecture.

**Approach:** Leverage v1.0.0 improvements for better code organization
**Constraint:** Preserve existing store pattern interfaces and domain model contracts

**Risk:** Data serialization changes
**Mitigation:** Validate JSON marshaling/unmarshaling with existing data

**Risk:** Connection behavior changes
**Mitigation:** Extensive testing of connection lifecycle and error scenarios

### Rollback Strategy

- Maintain ability to revert to v0.9.0 if critical issues arise
- Document all changes made during upgrade
- Test rollback procedure in development environment

## Performance Considerations

### Expected Improvements

- Leverage v1.0.0 performance optimizations
- Maintain existing connection pooling behavior
- Preserve query execution efficiency
- Monitor memory usage patterns

### Monitoring

- Maintain existing logging and metrics
- Monitor connection health check performance
- Track query execution times
- Validate reconnection behavior under load
