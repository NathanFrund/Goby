# Design Document

## Overview

The service registry listing feature extends the existing Goby CLI tool with a new `list-services` command that provides visibility into registered services within the Goby framework's registry. This feature helps developers discover available dependencies and understand the service ecosystem when building modules.

## Architecture

The feature follows the existing Goby CLI architecture pattern using Cobra commands. It will integrate with the Goby application's registry system to extract and display service information.

### Key Components

1. **CLI Command Handler**: A new Cobra command that handles the `list-services` subcommand
2. **Registry Inspector**: A utility that can introspect the registry to extract service information
3. **Service Metadata Extractor**: Logic to extract meaningful information about registered services
4. **Output Formatter**: Utilities to format service information for display

## Components and Interfaces

### CLI Command Structure

The new command will follow the existing pattern established by `new-module` and `version` commands:

```go
// listServicesCmd represents the list-services command
var listServicesCmd = &cobra.Command{
    Use:   "list-services",
    Short: "List all registered services in the Goby registry",
    Long:  `Displays all services currently registered in the Goby application registry...`,
    Run:   listServicesHandler,
}
```

### Registry Integration

**Low-Intrusion Approach**: Instead of modifying the registry system, use static code analysis to discover services. This approach:

1. **Static Analysis**: Parse Go source files to find `registry.Key` declarations and `registry.Set` calls
2. **Code Pattern Recognition**: Extract service information from existing code patterns without runtime introspection
3. **Zero Framework Changes**: No modifications needed to the registry, application startup, or existing service registrations
4. **Development Tool Only**: Operates entirely as a development-time tool analyzing source code

**Alternative High-Intrusion Approach** (if static analysis proves insufficient):

1. **Registry Extension**: Add minimal metadata export capability
2. **Introspection Mode**: Add a CLI flag to the main application for service discovery
3. **Service Registration Enhancement**: Optionally enhance service registrations with metadata

### Service Information Structure

```go
type ServiceInfo struct {
    Key         string
    Type        string
    Description string
    Module      string
}

type RegistrySnapshot struct {
    Services []ServiceInfo
    Count    int
}
```

### Registry Enhancement

Add methods to the registry package to support service discovery:

```go
// ServiceMetadata represents information about a registered service
type ServiceMetadata struct {
    Key         string
    TypeName    string
    Description string
}

// GetAllServices returns metadata for all registered services
func (r *Registry) GetAllServices() []ServiceMetadata

// RegisterServiceMetadata allows services to register descriptive metadata
func (r *Registry) RegisterServiceMetadata(key string, metadata ServiceMetadata)
```

## Data Models

### Service Registry Keys

Based on the codebase analysis, services are registered using type-safe keys like:

- `"core.database.Connection"` - Database connection service
- `"core.presence.Service"` - Presence service
- `"wargame.Engine"` - Wargame engine service
- `"core.script.Engine"` - Script engine service

### Service Categories

Services can be categorized as:

- **Core Services**: Essential framework services (database, presence, script engine)
- **Module Services**: Services provided by specific modules (wargame engine)
- **Test Services**: Services used in testing environments

## Error Handling

### Registry Access Errors

- Handle cases where the registry cannot be accessed
- Provide meaningful error messages when service discovery fails

### Application State Errors

- Handle scenarios where the application is not properly initialized
- Gracefully handle missing or incomplete service registrations

### CLI Argument Errors

- Validate command arguments and flags
- Provide helpful usage information for invalid inputs

## Testing Strategy

### Unit Tests

- Test service metadata extraction logic
- Test output formatting functions
- Test error handling scenarios

### Integration Tests

- Test the complete CLI command execution
- Test registry integration with actual services
- Test output formatting with real service data

### Manual Testing

- Verify command works with existing Goby applications
- Test with different service configurations
- Validate output readability and usefulness

## Implementation Approach

### Phase 1: Static Analysis Implementation (Low Intrusion)

1. Create Go AST parser to find `registry.Key` declarations
2. Implement pattern matching for `registry.Set` calls
3. Extract service information from source code comments and patterns

### Phase 2: CLI Command Implementation

1. Create the `list-services` command structure
2. Implement source code analysis logic
3. Add output formatting and display logic

### Phase 3: Enhancement (Optional)

1. Add more sophisticated code analysis for better service descriptions
2. Support for analyzing external modules/packages
3. Integration with Go modules for dependency analysis

**Intrusion Level**: Minimal - only adds new CLI command, no changes to existing Goby framework code

## Cost-Benefit Analysis

### Benefits

- **Developer Experience**: Significantly improves module development by providing service discovery
- **Documentation**: Acts as living documentation of available services
- **Onboarding**: Helps new developers understand the service ecosystem
- **Debugging**: Assists in troubleshooting dependency issues

### Costs (Static Analysis Approach)

- **Implementation Time**: ~1-2 days for basic functionality
- **Maintenance**: Low - only needs updates if registry patterns change
- **Framework Impact**: Zero - no changes to existing Goby code
- **Performance**: No runtime impact on Goby applications

### Alternative: Manual Documentation

Instead of this feature, services could be documented manually in README files or wiki pages, but this approach:

- Requires manual maintenance and often becomes outdated
- Doesn't provide the same level of detail or searchability
- Lacks integration with the development workflow

**Recommendation**: The static analysis approach provides high value with minimal cost and zero framework intrusion.

## Command Interface Design

### Basic Usage

```bash
# Development usage
go run ./cmd/goby-cli list-services                    # List all services
go run ./cmd/goby-cli list-services --format table    # Table format (default)
go run ./cmd/goby-cli list-services --format json     # JSON format
go run ./cmd/goby-cli list-services --category core   # Filter by category
go run ./cmd/goby-cli list-services service-name      # Show detailed info for specific service

# If built and installed
goby list-services                    # List all services (when CLI is built/installed)
```

### Output Formats

**Table Format (Default):**

```
SERVICE KEY                    TYPE                    MODULE      DESCRIPTION
core.database.Connection       *database.Connection    core        Database connection manager
core.presence.Service          *presence.Service       core        User presence tracking
wargame.Engine                 *Engine                 wargame     Game engine for wargame module
core.script.Engine             ScriptEngine            core        Script execution engine
```

**JSON Format:**

```json
{
  "services": [
    {
      "key": "core.database.Connection",
      "type": "*database.Connection",
      "module": "core",
      "description": "Database connection manager"
    }
  ],
  "count": 4
}
```

## Security Considerations

- This feature is designed for development environments only
- Service listing should not expose sensitive configuration data or runtime values
- Only expose service metadata (keys, types, descriptions) that are safe for development use
- The CLI should detect if it's running against a production environment and either warn or restrict functionality
- Consider adding a `--allow-production` flag for cases where this might be needed in production debugging

## Performance Considerations

- Service discovery should be efficient and not impact application startup
- Registry inspection should be read-only and thread-safe
- Output formatting should handle large numbers of services gracefully
