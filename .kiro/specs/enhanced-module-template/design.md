# Enhanced Module Template Design

## Overview

The enhanced module template will transform the goby-cli's new_module command from generating basic scaffolding to creating production-ready module foundations. The design focuses on providing developers with pubsub integration, topic management, and background service patterns while maintaining simplicity and following Go best practices.

The template will generate modules that demonstrate real-world patterns for inter-module communication, proper lifecycle management, and extensibility points for advanced features.

## Architecture

### Template Generation Strategy

The enhanced template uses a layered approach:

1. **Core Layer**: Always includes basic module structure, HTTP handlers, and renderer
2. **Communication Layer**: Adds pubsub and topic management (default behavior)
3. **Minimal Mode**: Strips back to just renderer dependency (--minimal flag)
4. **Extension Examples**: Commented code showing advanced integrations

### Module Structure

Generated modules will follow this directory structure:

```
internal/modules/{name}/
├── module.go           # Main module implementation
├── handler.go          # HTTP request handlers
├── subscriber.go       # Background message processing
├── topics/
│   └── topics.go       # Topic definitions and registration
└── README.md           # Module-specific documentation
```

### Dependency Injection Pattern

The template will use explicit dependency injection with clear separation of concerns:

```go
type Dependencies struct {
    // Core dependencies (always present)
    Renderer rendering.Renderer

    // Communication dependencies (default mode)
    Publisher  pubsub.Publisher
    Subscriber pubsub.Subscriber
    TopicMgr   *topicmgr.Manager

    // Data access dependencies (commented examples)
    // Database database.Database  // Raw database access
    // UserStore database.UserStore // Specific store example

    // Advanced dependencies (commented examples)
    // ScriptEngine script.ScriptEngine
    // PresenceService *presence.Service
}
```

## Components and Interfaces

### Module Component

The main module component implements the standard module.Module interface:

- **Register()**: Sets up topics and message handlers
- **Boot()**: Starts HTTP routes and background services
- **Shutdown()**: Gracefully stops background services
- **Name()**: Returns module identifier

Key improvements over current template:

- Proper error handling with wrapped errors
- Structured logging with context
- Clear separation of registration vs. runtime concerns

### Subscriber Component

A dedicated subscriber component handles background message processing:

- Context-aware goroutine management
- Graceful shutdown with timeout
- Error recovery and logging
- Message handler registration pattern

This addresses the current template's lack of background service examples.

### Topics Component

A separate topics package provides:

- Centralized topic definitions
- Registration helper functions
- Topic validation
- Clear naming conventions

### Handler Component

Enhanced HTTP handlers demonstrate:

- Proper error handling patterns
- User context extraction
- Request validation
- Response formatting

### Database Integration Component

The template includes commented examples for database access patterns:

- **Store Pattern**: Using typed stores (UserStore, FileStore) for domain-specific operations
- **Raw Database Access**: Direct database client for complex queries
- **Transaction Management**: Examples of proper transaction handling
- **Error Handling**: Database-specific error handling and logging

This provides developers with both the recommended store pattern and the flexibility of raw database access when needed.

## Data Models

### Topic Definitions

Topics follow a consistent naming pattern:

```go
var (
    TopicModuleEvent = topics.Topic{
        Name: "{module}.event",
        Description: "Events from the {module} module",
    }

    TopicModuleCommand = topics.Topic{
        Name: "{module}.command",
        Description: "Commands to the {module} module",
    }
)
```

### Message Structures

Standard message envelope pattern:

```go
type ModuleMessage struct {
    Type      string                 `json:"type"`
    Payload   map[string]interface{} `json:"payload"`
    Timestamp time.Time              `json:"timestamp"`
    UserID    string                 `json:"user_id,omitempty"`
}
```

## Error Handling

### Structured Error Handling

The template demonstrates proper Go error handling:

- Wrapped errors with context using `fmt.Errorf`
- Structured logging for error conditions
- Graceful degradation for non-critical failures
- Clear error messages for debugging

### Background Service Resilience

Subscriber services include:

- Panic recovery with logging
- Automatic reconnection logic
- Circuit breaker pattern for failing handlers
- Metrics collection for monitoring

## Testing Strategy

### Generated Test Structure

The template will include basic test files:

```
internal/modules/{name}/
├── module_test.go      # Module lifecycle tests
├── handler_test.go     # HTTP handler tests
├── subscriber_test.go  # Message processing tests
└── topics/
    └── topics_test.go  # Topic registration tests
```

### Test Patterns

- Table-driven tests for handlers
- Mock dependencies for unit testing
- Integration test examples
- Benchmark tests for message processing

### Testing Utilities

Helper functions for common testing scenarios:

- Mock pubsub publisher/subscriber
- Test topic manager
- HTTP request builders
- Message assertion helpers

## Implementation Phases

### Phase 1: Core Template Enhancement

1. Update module.go template with pubsub integration
2. Add subscriber.go template with background service pattern
3. Create topics/ subdirectory template
4. Enhance handler.go with better patterns

### Phase 2: CLI Flag Support

1. Add --minimal flag for basic template
2. Update dependency injection logic
3. Conditional template generation
4. Update help documentation

### Phase 3: Advanced Examples

1. Add commented script engine integration
2. Include presence service examples
3. Create comprehensive README template
4. Add testing utilities

### Phase 4: Documentation and Polish

1. Update CLI help text
2. Create migration guide from old template
3. Add troubleshooting documentation
4. Performance optimization

## Migration Strategy

### Backward Compatibility

- Existing generated modules continue to work unchanged
- New template generates different structure but same interfaces
- Clear migration path for upgrading existing modules

### Developer Experience

- Improved error messages during generation
- Better validation of module names
- Helpful output showing next steps
- Links to documentation and examples

## Security Considerations

### Input Validation

- Module name sanitization
- Path traversal prevention
- Template injection protection

### Generated Code Security

- Proper input validation in handlers
- SQL injection prevention examples
- XSS protection in templates
- Rate limiting examples

## Performance Considerations

### Template Generation

- Efficient file I/O operations
- Minimal memory allocation
- Fast AST manipulation
- Parallel file generation where possible

### Generated Code Performance

- Efficient message routing
- Connection pooling examples
- Caching patterns
- Resource cleanup

## Monitoring and Observability

### Logging Standards

Generated modules include:

- Structured logging with slog
- Consistent log levels
- Request tracing
- Performance metrics

### Health Checks

- Module health endpoints
- Dependency health checks
- Background service monitoring
- Graceful degradation indicators
