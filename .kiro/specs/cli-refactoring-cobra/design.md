# Design Document

## Overview

This design outlines the refactoring of Goby's command-line interface using the Cobra framework to create a unified, professional CLI experience. The design prioritizes backward compatibility while establishing a foundation for future CLI feature expansion.

## Architecture

### Command Structure

The new CLI will follow a hierarchical command structure:

```
goby                          # Root command (defaults to server)
├── serve                     # Explicit server command
├── scripts                   # Script management commands
│   ├── extract <directory>   # Extract embedded scripts
│   ├── list                  # List available scripts
│   └── validate              # Validate script syntax
├── topics                    # Topic management commands
│   ├── list                  # List all topics
│   └── get <name>           # Get topic details
└── version                   # Version information
```

### Package Organization

```
cmd/
├── goby/                     # New unified CLI
│   ├── main.go              # Entry point and root command
│   ├── cmd/                 # Command implementations
│   │   ├── root.go          # Root command setup
│   │   ├── serve.go         # Server command
│   │   ├── scripts/         # Script commands
│   │   │   ├── scripts.go   # Scripts parent command
│   │   │   ├── extract.go   # Extract command
│   │   │   ├── list.go      # List command
│   │   │   └── validate.go  # Validate command
│   │   ├── topics/          # Topic commands
│   │   │   ├── topics.go    # Topics parent command
│   │   │   ├── list.go      # List command
│   │   │   └── get.go       # Get command
│   │   └── version.go       # Version command
│   └── internal/            # CLI-specific utilities
│       ├── config.go        # CLI configuration helpers
│       └── output.go        # Output formatting utilities
├── server/                   # Legacy server (deprecated)
└── topics/                   # Legacy topics (deprecated)
```

## Components and Interfaces

### Root Command

The root command serves as the entry point and handles:

- Default behavior (server startup)
- Global flags and configuration
- Command registration and routing
- Version and help information

```go
type RootCommand struct {
    cobra.Command
    globalConfig *config.Config
}
```

### Command Categories

#### Server Command

- Encapsulates all current server functionality
- Maintains existing flag compatibility
- Handles graceful shutdown and signal management

#### Scripts Commands

- **Extract**: Migrates current script extraction logic
- **List**: New functionality to display available scripts
- **Validate**: New functionality for script validation

#### Topics Commands

- **List**: Migrates current topic listing functionality
- **Get**: Migrates current topic detail functionality

### Configuration Management

The CLI will use a unified configuration approach:

- Environment variables (highest priority)
- Configuration files
- Command-line flags
- Default values (lowest priority)

### Output Formatting

Standardized output formatting across all commands:

- Consistent error messaging
- Structured output for machine consumption (JSON/YAML options)
- Human-readable formatting for interactive use
- Progress indicators for long-running operations

## Data Models

### Command Context

```go
type CommandContext struct {
    Config      *config.Config
    Logger      *slog.Logger
    OutputFormat string
    Verbose     bool
}
```

### Script Metadata

```go
type ScriptInfo struct {
    Module      string
    Name        string
    Language    string
    Source      string
    Description string
    Size        int64
}
```

## Error Handling

### Error Categories

1. **Configuration Errors**: Invalid flags, missing required values
2. **Runtime Errors**: Service failures, network issues
3. **User Errors**: Invalid input, missing files
4. **System Errors**: Permissions, disk space

### Error Response Format

- Consistent error message structure
- Appropriate exit codes following Unix conventions
- Helpful suggestions for common errors
- Debug information available with verbose flags

## Testing Strategy

### Unit Testing

- Individual command logic testing
- Flag parsing and validation
- Output formatting verification
- Error handling scenarios

### Integration Testing

- End-to-end command execution
- Backward compatibility verification
- Configuration loading and precedence
- Cross-platform compatibility

### Compatibility Testing

- Legacy command syntax support
- Environment variable handling
- Exit code consistency
- Output format preservation

## Migration Strategy

### Phase 1: Foundation

- Implement Cobra framework structure
- Create root command with server default
- Migrate server functionality to `serve` command
- Maintain full backward compatibility

### Phase 2: Command Migration

- Implement `topics` subcommands
- Migrate script extraction to `scripts extract`
- Add deprecation warnings for legacy usage
- Comprehensive testing of migrated functionality

### Phase 3: Enhancement

- Add new `scripts list` and `scripts validate` commands
- Implement enhanced help and completion
- Add structured output options
- Performance optimization

### Phase 4: Cleanup

- Remove legacy command binaries
- Update documentation and examples
- Final compatibility verification
- Release preparation

## Security Considerations

### Input Validation

- Sanitize all user input and file paths
- Validate command arguments and flags
- Prevent path traversal in file operations

### Privilege Management

- Run with minimal required privileges
- Secure handling of configuration files
- Safe temporary file creation

### Error Information Disclosure

- Avoid exposing sensitive information in error messages
- Sanitize stack traces in production
- Secure logging of command execution

## Performance Considerations

### Startup Time

- Lazy loading of heavy dependencies
- Efficient command registration
- Minimal initialization for help/version commands

### Memory Usage

- Stream processing for large operations
- Efficient data structures
- Proper resource cleanup

### Scalability

- Support for large script collections
- Efficient topic registry operations
- Responsive command completion

## Deployment Considerations

### Binary Distribution

- Single binary deployment
- Cross-platform compatibility
- Minimal external dependencies

### Backward Compatibility

- Symlink support for legacy binary names
- Environment variable preservation
- Configuration file compatibility

### Documentation

- Migration guide for existing users
- Updated CLI reference documentation
- Example usage scenarios
