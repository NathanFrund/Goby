# Implementation Plan

- [x] 1. Set up CLI command structure

  - Create new `list-services` command file following existing patterns
  - Add command registration to root command
  - Implement basic command structure with help text and flags
  - _Requirements: 3.1, 3.2, 3.3_

- [x] 2. Implement static analysis approach (primary)

  - [x] 2.1 Create Go AST parser for registry key discovery

    - Write parser to find `registry.Key[T]` declarations in source files
    - Extract key names, types, and associated comments
    - Handle different file patterns and import paths
    - _Requirements: 1.1, 1.2_

  - [x] 2.2 Implement registry.Set call detection

    - Parse source files to find `registry.Set` function calls
    - Match Set calls with their corresponding Key declarations
    - Extract service registration locations and context
    - _Requirements: 1.1, 1.2_

  - [x] 2.3 Build service information aggregator

    - Combine data from key declarations and Set calls
    - Create service metadata structure with key, type, module, description
    - Handle edge cases and missing information gracefully
    - _Requirements: 1.2, 2.2_

  - [ ]\* 2.4 Write unit tests for static analysis
    - Test AST parsing with sample Go files
    - Test service information extraction accuracy
    - Test error handling for malformed code
    - _Requirements: 1.1, 1.2_

- [ ] 3. Implement fallback intrusive approach (if static analysis fails)

  - [ ] 3.1 Add registry service metadata tracking

    - Extend registry package with metadata collection methods
    - Add ServiceMetadata struct and storage mechanism
    - Implement GetAllServices method for registry export
    - _Requirements: 1.1, 1.2_

  - [ ] 3.2 Create application introspection mode

    - Add CLI flag to main Goby application for service discovery
    - Implement startup sequence that registers services and exports registry
    - Add graceful shutdown after registry export
    - _Requirements: 1.1, 2.1_

  - [ ] 3.3 Update existing service registrations
    - Modify core service registrations to include metadata
    - Update module service registrations with descriptions
    - Ensure backward compatibility with existing code
    - _Requirements: 1.2, 2.2_

- [x] 4. Implement output formatting and display

  - [x] 4.1 Create service information display logic

    - Implement table format output with columns for key, type, module, description
    - Add JSON format output option for programmatic use
    - Handle empty registry case with appropriate messaging
    - _Requirements: 1.3, 1.4_

  - [x] 4.2 Add command-line options and filtering

    - Implement --format flag for output format selection
    - Add --category flag for filtering services by type
    - Support specific service lookup by name argument
    - _Requirements: 2.1, 2.3, 2.4, 3.4_

  - [x] 4.3 Implement detailed service information display
    - Show comprehensive service details when specific service is requested
    - Include interface information and usage examples when available
    - Format output for readability and developer use
    - _Requirements: 2.2, 2.3_

- [ ]\* 5. Add comprehensive testing

  - [ ]\* 5.1 Write integration tests for CLI command

    - Test complete command execution with real Goby project
    - Verify output formatting and accuracy
    - Test error handling and edge cases
    - _Requirements: 1.1, 1.2, 1.3, 1.4_

  - [ ]\* 5.2 Add end-to-end testing
    - Test with different Goby project configurations
    - Verify service discovery accuracy across different modules
    - Test performance with large numbers of services
    - _Requirements: 1.1, 2.1_

- [-] 6. Finalize and integrate

  - [x] 6.1 Add environment detection and safety checks

    - Implement development vs production environment detection
    - Add appropriate warnings or restrictions for production use
    - Include --allow-production flag for production debugging if needed
    - _Requirements: 3.4_

  - [x] 6.2 Update CLI documentation and help text

    - Write comprehensive help text for the new command
    - Add usage examples and common use cases
    - Update main CLI help to include new command
    - _Requirements: 3.2, 3.3_

  - [x] 6.3 Validate implementation against requirements
    - Test all acceptance criteria from requirements document
    - Ensure proper error handling and user feedback
    - Verify integration with existing CLI workflow
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 3.4_
