# Analysis of Current Template Structure and Patterns

## Current Template Analysis

### Generated Template Structure

The current `new_module` command generates a minimal module structure with:

**Files Generated:**

- `internal/modules/{name}/module.go` - Main module implementation
- `internal/modules/{name}/handler.go` - HTTP request handlers

**Automatic Updates:**

- `internal/app/dependencies.go` - Adds dependency injection function
- `internal/app/modules.go` - Registers module in application

### Current Template Patterns

#### 1. Module.go Template Analysis

**Current Dependencies (Generated):**

```go
type Dependencies struct {
    Publisher  pubsub.Publisher
    Subscriber pubsub.Subscriber
    Renderer   rendering.Renderer
    TopicMgr   *topicmgr.Manager
}
```

**Current Module Structure:**

- Implements `module.Module` interface
- Has `Register()`, `Boot()`, and `Name()` methods
- Includes placeholder methods for topic registration and message handlers
- Basic error handling with wrapped errors
- Structured logging with slog

**Issues Identified:**

1. **Incomplete Implementation**: Generated `registerTopics()` and `registerHandlers()` are empty placeholders
2. **No Background Services**: No subscriber service implementation
3. **No Shutdown Method**: Missing graceful shutdown handling
4. **No Topics Package**: No separate topics directory structure
5. **Limited Examples**: Minimal guidance for developers

#### 2. Handler.go Template Analysis

**Current Handler Patterns:**

- Basic HTTP handler structure with renderer dependency
- User authentication context extraction
- Two example routes: protected and public
- Basic error handling for unauthorized access
- Simple template rendering with templ components

**Good Patterns:**

- Proper user context extraction with `getCurrentUser()`
- User display name helper function
- Error handling for authentication
- Integration with flash message system

**Areas for Improvement:**

- No pubsub integration examples
- No message publishing from HTTP endpoints
- Limited error handling patterns
- No request validation examples

#### 3. Dependency Injection Analysis

**Current Pattern:**

```go
func {name}Deps(deps Dependencies) {name}.Dependencies {
    return {name}.Dependencies{
        Renderer: deps.Renderer,
    }
}
```

**Issues:**

- Only includes Renderer dependency (minimal)
- Doesn't match the actual generated Dependencies struct
- Missing pubsub, topicmgr, and other dependencies

## Best Practices from Existing Modules

### Chat Module Patterns

**Excellent Patterns to Extract:**

1. **Complete Dependency Injection:**

```go
type Dependencies struct {
    Publisher       pubsub.Publisher
    Subscriber      pubsub.Subscriber
    Renderer        rendering.Renderer
    TopicMgr        *topicmgr.Manager
    PresenceService *presence.Service
}
```

2. **Proper Background Service Management:**

- Dedicated subscriber service with `Start(ctx)` method
- Context-aware goroutine management
- Multiple subscription handlers in separate goroutines
- Proper error handling and logging

3. **Topic Organization:**

- Separate `topics/topics.go` package
- Comprehensive topic definitions with metadata
- `RegisterTopics()` function for centralized registration
- Clear topic naming conventions

4. **Message Processing Patterns:**

- Structured message handling with JSON unmarshaling
- Component rendering for HTML output
- Proper error recovery (don't stop subscriber for bad messages)
- Message routing based on content (direct vs broadcast)

5. **HTTP Integration:**

- Publishing messages from HTTP endpoints
- Proper request validation and error handling
- Integration with presence service

### Wargame Module Patterns

**Advanced Patterns to Extract:**

1. **Script Engine Integration:**

- Script helper for module-specific scripts
- Embedded script provider pattern
- Message handler script execution
- HTTP endpoint script execution
- Exposed functions for script access

2. **Complex Background Processing:**

- Multiple event types with different handlers
- Script-enhanced message processing
- Chain reaction detection and processing
- Fallback behavior when scripts fail

3. **Service Registration:**

- Registry pattern for exposing module services
- Type-safe service keys
- Service lifecycle management

4. **Advanced Error Handling:**

- Script execution error handling
- Graceful degradation patterns
- Comprehensive logging with context

## Areas for Improvement in Current Template

### 1. Incomplete Pubsub Integration

**Current Issues:**

- Generated template includes pubsub dependencies but no implementation
- Empty placeholder methods for topic registration and message handlers
- No background subscriber service
- No examples of publishing or subscribing to messages

**Required Improvements:**

- Complete subscriber service implementation
- Working examples of message handlers
- Topic registration with actual topics
- Publishing examples from HTTP handlers

### 2. Missing Background Service Patterns

**Current Issues:**

- No subscriber service generated
- No goroutine management examples
- No graceful shutdown handling
- No context-based cancellation

**Required Improvements:**

- Dedicated subscriber.go template
- Proper lifecycle management (Start/Stop)
- Context-aware background services
- Graceful shutdown with timeout handling

### 3. Inadequate Topic Management

**Current Issues:**

- No topics/ subdirectory generated
- No topic definitions or registration
- No examples of topic usage
- No validation or error handling

**Required Improvements:**

- Generate topics/topics.go with examples
- Topic registration in Register() method
- Clear naming conventions and documentation
- Validation and error handling patterns

### 4. Limited Dependency Injection

**Current Issues:**

- Mismatch between generated Dependencies struct and actual injection
- Only Renderer dependency in injection function
- No support for optional dependencies
- No examples of advanced service integration

**Required Improvements:**

- Complete dependency injection matching generated struct
- Support for minimal vs full dependency modes
- Commented examples for optional services
- Proper error handling in dependency injection

### 5. Insufficient Documentation and Examples

**Current Issues:**

- Minimal inline comments
- No module-specific README
- Limited examples of common patterns
- No guidance for customization

**Required Improvements:**

- Comprehensive inline documentation
- Generated README template
- Multiple usage examples
- Clear customization points with TODO comments

## Recommended Template Enhancements

### 1. Enhanced Module Structure

```
internal/modules/{name}/
├── module.go           # Enhanced with complete lifecycle
├── handler.go          # Enhanced with pubsub integration
├── subscriber.go       # NEW: Background message processing
├── topics/
│   └── topics.go       # NEW: Topic definitions and registration
└── README.md           # NEW: Module-specific documentation
```

### 2. Improved Dependency Patterns

**Default Mode (Full Integration):**

```go
type Dependencies struct {
    // Core dependencies
    Renderer rendering.Renderer

    // Communication dependencies
    Publisher  pubsub.Publisher
    Subscriber pubsub.Subscriber
    TopicMgr   *topicmgr.Manager

    // Optional advanced dependencies (commented)
    // Database database.Database
    // ScriptEngine script.ScriptEngine
    // PresenceService *presence.Service
}
```

**Minimal Mode (--minimal flag):**

```go
type Dependencies struct {
    Renderer rendering.Renderer
}
```

### 3. Complete Lifecycle Implementation

- **Register()**: Topic registration and message handler setup
- **Boot()**: HTTP routes and background service startup
- **Shutdown()**: Graceful cleanup with context and timeout
- **Background Services**: Proper goroutine management with context

### 4. Enhanced Error Handling

- Wrapped errors with context
- Structured logging throughout
- Graceful degradation for non-critical failures
- Recovery patterns for background services

### 5. Comprehensive Examples

- Message publishing from HTTP endpoints
- Background message processing
- Topic registration and validation
- User context extraction and validation
- Request/response patterns
- Error handling and recovery

## Implementation Priority

1. **High Priority**: Complete pubsub integration and background services
2. **High Priority**: Topic management and registration
3. **Medium Priority**: Enhanced dependency injection with minimal mode
4. **Medium Priority**: Comprehensive documentation and examples
5. **Low Priority**: Advanced service integration examples (script engine, presence)

This analysis provides the foundation for implementing the enhanced module template that addresses all identified gaps and incorporates best practices from existing modules.
