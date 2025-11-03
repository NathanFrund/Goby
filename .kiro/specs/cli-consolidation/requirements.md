# Requirements Document

## Introduction

This feature consolidates the standalone `cmd/topics/main.go` CLI tool into the existing `goby-cli` application as a subcommand group. This will provide developers with a unified CLI experience for all Goby framework inspection and management tasks.

## Glossary

- **Goby CLI**: The main command-line interface tool for the Goby framework
- **Topics CLI**: The standalone CLI tool for topic registry inspection
- **Topic Manager**: The internal service that manages topic registration and validation
- **Cobra**: The Go CLI library used by goby-cli for command structure
- **Subcommand Group**: A collection of related commands under a parent command

## Requirements

### Requirement 1

**User Story:** As a developer using the Goby framework, I want a single CLI tool for all framework operations, so that I don't need to remember multiple command names and can discover all available functionality in one place.

#### Acceptance Criteria

1. THE Goby CLI SHALL provide a `topics` subcommand group that includes all functionality from the standalone topics CLI
2. THE Goby CLI SHALL maintain backward compatibility for existing `goby-cli` commands
3. THE Goby CLI SHALL use consistent command patterns and flag naming across all subcommands
4. THE Goby CLI SHALL provide unified help documentation that shows all available commands

### Requirement 2

**User Story:** As a developer inspecting topic registrations, I want to list all topics with filtering options, so that I can understand what messaging topics are available in my application.

#### Acceptance Criteria

1. WHEN I run `goby-cli topics list`, THE Goby CLI SHALL display all registered topics in a formatted table
2. WHEN I run `goby-cli topics list --module=<name>`, THE Goby CLI SHALL display only topics from the specified module
3. WHEN I run `goby-cli topics list --scope=<scope>`, THE Goby CLI SHALL display only topics matching the specified scope (framework or module)
4. THE Goby CLI SHALL support JSON output format for programmatic consumption via `--format json` flag

### Requirement 3

**User Story:** As a developer debugging topic issues, I want to get detailed information about specific topics, so that I can understand their configuration and usage patterns.

#### Acceptance Criteria

1. WHEN I run `goby-cli topics get <topic-name>`, THE Goby CLI SHALL display detailed information about the specified topic
2. THE Goby CLI SHALL show topic name, scope, module, description, pattern, example, and metadata
3. THE Goby CLI SHALL display file locations where the topic is declared
4. IF the topic does not exist, THEN THE Goby CLI SHALL display an appropriate error message

### Requirement 4

**User Story:** As a developer ensuring topic compliance, I want to validate topic definitions, so that I can catch configuration errors early in development.

#### Acceptance Criteria

1. WHEN I run `goby-cli topics validate <topic-name>`, THE Goby CLI SHALL validate the topic name format and definition
2. THE Goby CLI SHALL display validation success with checkmark and topic details
3. IF validation fails, THEN THE Goby CLI SHALL display specific error messages with cross mark
4. THE Goby CLI SHALL validate both topic name format and topic definition completeness

### Requirement 5

**User Story:** As a developer migrating from the standalone topics CLI, I want the same initialization and error handling behavior, so that my existing workflows continue to work reliably.

#### Acceptance Criteria

1. THE Goby CLI SHALL initialize topic dependencies with minimal configuration setup
2. THE Goby CLI SHALL suppress logging output to maintain clean CLI output
3. THE Goby CLI SHALL handle module registration errors gracefully during topic discovery
4. THE Goby CLI SHALL load environment configuration from .env files when available
