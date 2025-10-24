# Requirements Document

## Introduction

This feature adds a general purpose embedded scripting engine to the Goby framework that supports both Tengo and Zygomys scripting languages. The system allows modules to execute custom scripts in response to messages from the bus, endpoint handlers, and other events. Scripts are embedded in the binary by default but can be overridden by domain experts through external files that are hot-loaded by the system.

## Glossary

- **Scripting_Engine**: The core component that manages script execution and language support
- **Tengo**: A small, dynamic, fast, secure script language for Go
- **Zygomys**: A Lisp interpreter written in Go
- **Module_Script**: A script file associated with a specific module for custom behavior
- **Hot_Loading**: The ability to reload scripts from disk without restarting the application
- **Script_Registry**: Component that manages script discovery, loading, and execution
- **Binary_Embedded_Scripts**: Default scripts compiled into the application binary
- **External_Script_Files**: User-modified scripts loaded from the filesystem

## Requirements

### Requirement 1

**User Story:** As a domain expert, I want to be able to choose the language that best fits my preference for customizable behavior without recomompiling the application.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL support both Tengo and Zygomys script languages
2. WHEN a module requests script execution, THE Scripting_Engine SHALL execute the script with the specified engine type
3. THE Scripting_Engine SHALL provide a unified interface for script execution regardless of the underlying language
4. THE Scripting_Engine SHALL pass context and data to scripts during execution
5. THE Scripting_Engine SHALL return script execution results to the calling module

### Requirement 2

**User Story:** As a module developer, I want to embed default scripts in my module, so that the system works out of the box with sensible defaults.

#### Acceptance Criteria

1. THE Script_Registry SHALL discover Binary_Embedded_Scripts at application startup
2. THE Script_Registry SHALL organize scripts by module name in the embedded filesystem
3. WHEN no External_Script_Files exist, THE Script_Registry SHALL use Binary_Embedded_Scripts
4. THE Script_Registry SHALL maintain a mapping between module names and their associated scripts
5. THE Binary_Embedded_Scripts SHALL be accessible through the same interface as External_Script_Files

### Requirement 3

**User Story:** As a domain expert, I want to be able to override default scripts with custom implementations, so that I can customize application behavior without modifying source code.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL support loading External_Script_Files from the filesystem
2. THE Scripting_Engine SHALL support replacement External_Script_Files that are not in the language of the embedded default scripts.
3. WHEN External_Script_Files exist, THE Script_Registry SHALL prioritize them over Binary_Embedded_Scripts
4. THE Script_Registry SHALL organize External_Script_Files in a `scripts/<module_name>/` directory structure
5. THE Script_Registry SHALL validate External_Script_Files before loading them
6. IF External_Script_Files are invalid, THEN THE Script_Registry SHALL fall back to Binary_Embedded_Scripts

### Requirement 4

**User Story:** As a domain expert, I want the system to extract default scripts to disk, so that I can use them as a starting point for customization.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL provide a command to extract Binary_Embedded_Scripts to disk
2. THE Scripting_Engine SHALL create the directory structure `scripts/<module_name>/` when extracting
3. THE Scripting_Engine SHALL preserve script file names and organization during extraction
4. THE Scripting_Engine SHALL not overwrite existing External_Script_Files during extraction
5. THE Scripting_Engine SHALL provide feedback about the extraction process

### Requirement 5

**User Story:** As a domain expert, I want scripts to be hot-loaded from disk, so that I can see their changes immediately without restarting the application.

#### Acceptance Criteria

1. THE Script_Registry SHALL monitor External_Script_Files for changes
2. WHEN External_Script_Files are modified, THE Script_Registry SHALL reload them automatically
3. THE Script_Registry SHALL validate reloaded scripts before making them active
4. IF reloaded scripts are invalid, THEN THE Script_Registry SHALL keep the previous valid version
5. THE Script_Registry SHALL provide notifications about script reload status
6. IF an exported script is deleted, THEN the default script for that functionality will be used instead.

### Requirement 6

**User Story:** As a module developer, I want scripts to execute in response to user input over module http endpoints and bus messages, so that I can provide event-driven customizable behavior.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL integrate with the existing message bus system
2. WHEN a bus message is received, THE Scripting_Engine SHALL execute associated Module_Scripts
3. THE Scripting_Engine SHALL pass message data to scripts as execution context
4. THE Scripting_Engine SHALL handle script execution errors without crashing the module
5. THE Scripting_Engine SHALL support asynchronous script execution for non-blocking operations

### Requirement 7

**User Story:** As a system administrator, I want visibility into script execution status, so that I can troubleshoot issues and monitor system behavior.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL log script execution events with appropriate detail levels
2. THE Scripting_Engine SHALL track script execution metrics including success/failure rates
3. THE Scripting_Engine SHALL provide status information about loaded scripts
4. THE Scripting_Engine SHALL report script compilation and runtime errors clearly
5. THE Scripting_Engine SHALL integrate with the existing logging system

### Requirement 8

**User Story:** As a system administrator, I want scripts to run within defined security and resource boundaries, so that untrusted code cannot harm the host application or consume excessive resources.

#### Acceptance Criteria

1. THE Scripting_Engine SHALL enforce strict whitelisting of host application functions and packages exposed to scripts
2. THE Scripting_Engine SHALL prevent script access to dangerous packages and system calls
3. THE Scripting_Engine SHALL enforce a CPU time limit on script execution to prevent infinite loops
4. THE Scripting_Engine SHALL enforce a memory limit on each script instance to prevent excessive memory consumption
5. THE Scripting_Engine SHALL provide standardized error types when scripts violate security or resource boundaries
