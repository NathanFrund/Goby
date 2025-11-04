# Goby CLI

A command-line interface tool for the Goby framework that helps developers discover and manage services within the Goby registry system.

## Installation

Build the CLI tool from the project root:

```bash
go build -o goby-cli ./cmd/goby-cli
```

## Commands

### list-services

Discover and list registered services in the Goby registry using static code analysis.

#### Basic Usage

```bash
# List all services
./goby-cli list-services

# List services in JSON format
./goby-cli list-services --format json

# Show detailed information for a specific service
./goby-cli list-services core.database.Connection
```

#### Filtering Options

```bash
# Filter by category
./goby-cli list-services --category core
./goby-cli list-services --category module

# Filter by module
./goby-cli list-services --module wargame

# Show only registered services
./goby-cli list-services --registered-only

# Show only declared but unregistered services
./goby-cli list-services --declared-only
```

#### Output Formats

- **table** (default): Human-readable table format with summary
- **json**: Machine-readable JSON format for programmatic use

#### Service Categories

- **core**: Essential framework services (database, presence, script engine)
- **module**: Services provided by specific modules (wargame engine)
- **test**: Services used in testing environments
- **command**: Services from command-line applications

## How It Works

The `list-services` command uses static analysis to discover services by:

1. Parsing Go source files for `registry.Key[T]` declarations
2. Finding corresponding `registry.Set` function calls
3. Extracting service metadata from code comments and patterns
4. Matching declarations with registrations to show service status

This approach is safe and read-only - it doesn't execute any code or modify the registry.

## Requirements

- Must be run from the root directory of a Go project
- Project must have a `go.mod` file
- Works best with Goby framework projects that use the registry pattern

## Safety

The tool includes environment detection and will warn when running in production environments. Use the `--allow-production` flag if you need to run analysis in production (though this is generally not recommended).

## Examples

```bash
# Quick overview of all services
./goby-cli list-services

# Detailed view of a specific service with usage examples
./goby-cli list-services "core.script.Engine"

# Export service information for external tools
./goby-cli list-services --format json > services.json

# Find all core services
./goby-cli list-services --category core

# Check for unregistered services
./goby-cli list-services --declared-only
```
