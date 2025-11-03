# Design Document

## Overview

This design consolidates the standalone topics CLI into the existing goby-cli application by creating a new `topics` subcommand group. The implementation will extract the core functionality from `cmd/topics/main.go` and restructure it as Cobra commands while maintaining all existing behavior and adding consistent CLI patterns.

## Architecture

### Command Structure

```
goby-cli
├── list-services (existing)
├── new-module (existing)
├── version (existing)
└── topics (new subcommand group)
    ├── list
    ├── get
    └── validate
```

### Component Integration

- **Existing goby-cli**: Maintains current Cobra-based architecture
- **Topics functionality**: Extracted into separate command files following goby-cli patterns
- **Shared initialization**: Topics initialization logic becomes reusable utility
- **Consistent output**: All commands use similar table formatting and JSON output patterns

## Components and Interfaces

### New Command Files

1. **`cmd/goby-cli/cmd/topics.go`**: Parent command for topics subcommand group
2. **`cmd/goby-cli/cmd/topics_list.go`**: List topics with filtering
3. **`cmd/goby-cli/cmd/topics_get.go`**: Get detailed topic information
4. **`cmd/goby-cli/cmd/topics_validate.go`**: Validate topic definitions

### Shared Utilities

1. **`cmd/goby-cli/internal/topics/initializer.go`**: Topic initialization logic
2. **`cmd/goby-cli/internal/topics/formatter.go`**: Output formatting utilities

### Command Patterns

Each topics subcommand follows the established goby-cli pattern:

- Cobra command definition with proper flags
- Handler function with error handling
- Consistent output formatting (table/JSON)
- Proper help documentation

## Data Models

### Topic Display Model

```go
type TopicDisplay struct {
    Name        string
    Scope       string
    Module      string
    Description string
    Example     string
    Pattern     string
    Metadata    map[string]interface{}
}
```

### Command Configuration

```go
type TopicsConfig struct {
    OutputFormat string // "table" or "json"
    ModuleFilter string
    ScopeFilter  string
}
```

## Error Handling

### Initialization Errors

- Graceful handling of missing .env files
- Module registration errors ignored for topic discovery
- Clear error messages for configuration issues

### Command Errors

- Topic not found: User-friendly error with suggestions
- Validation failures: Specific error details with context
- Invalid filters: Clear guidance on valid options

### Output Consistency

- All errors written to stderr
- Exit codes: 0 for success, 1 for errors
- Consistent error message formatting

## Testing Strategy

### Unit Tests

- Topic initialization logic
- Output formatting functions
- Filter and validation logic
- Error handling scenarios

### Integration Tests

- Full command execution with real topic data
- Flag parsing and validation
- Output format verification
- Error condition testing

### Migration Validation

- Verify all existing topics CLI functionality works
- Compare output formats between old and new implementations
- Test all command combinations and flags

## Implementation Approach

### Phase 1: Extract Core Logic

1. Move topic initialization to shared utility
2. Extract formatting functions to reusable components
3. Create base topic command structure

### Phase 2: Implement Subcommands

1. Create `topics list` command with filtering
2. Implement `topics get` for detailed views
3. Add `topics validate` functionality

### Phase 3: Integration & Cleanup

1. Update root command help text
2. Add topics commands to command registration
3. Remove standalone topics CLI
4. Update documentation

### Migration Strategy

- Keep standalone CLI during development for comparison
- Implement feature parity before removal
- Provide migration guide for users
- Update build scripts and documentation
