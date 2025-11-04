# Implementation Plan

- [x] 1. Analyze current template structure and patterns

  - Review existing module.go and handler.go templates
  - Examine chat and wargame modules for best practices to extract
  - Identify areas for improvement in current generated code
  - Document current dependency injection patterns
  - _Requirements: 1.1, 5.1_

- [x] 2. Create enhanced module template with pubsub integration

  - [x] 2.1 Update module.go template with pubsub dependencies

    - Add Publisher and Subscriber to Dependencies struct
    - Include TopicMgr dependency for topic registration
    - Add proper error handling with wrapped errors
    - Include structured logging with slog
    - _Requirements: 1.1, 1.5_

  - [x] 2.2 Enhance Register method with topic management

    - Add registerTopics() helper method
    - Add registerHandlers() helper method for message subscriptions
    - Include proper error handling and logging
    - _Requirements: 2.1, 2.3, 2.5_

  - [x] 2.3 Improve Boot method with background service startup

    - Add subscriber service initialization
    - Include proper goroutine management with context
    - Add HTTP route registration with better patterns
    - _Requirements: 4.1, 4.4_

  - [x] 2.4 Add Shutdown method with graceful cleanup
    - Implement proper context-based cancellation
    - Add timeout handling for graceful shutdown
    - Include logging for shutdown lifecycle
    - _Requirements: 4.3, 4.4_

- [x] 3. Create subscriber template for background message processing

  - [x] 3.1 Design subscriber service structure

    - Create subscriber.go template with proper lifecycle management
    - Add context-aware Start method
    - Include message handler registration pattern
    - _Requirements: 1.2, 4.1, 4.2_

  - [x] 3.2 Implement message processing patterns

    - Add example message handlers with different patterns
    - Include proper error handling and recovery
    - Add structured logging for message processing
    - _Requirements: 1.4, 5.2_

  - [x] 3.3 Add graceful shutdown handling
    - Implement proper context cancellation
    - Add cleanup for message subscriptions
    - Include timeout handling for pending messages
    - _Requirements: 4.2, 4.3_

- [x] 4. Create topics package template

  - [x] 4.1 Design topic definition structure

    - Create topics/topics.go template
    - Define standard topic naming conventions
    - Add topic validation helpers
    - _Requirements: 2.2, 2.4_

  - [x] 4.2 Implement topic registration helpers
    - Add RegisterTopics() function
    - Include proper error handling for topic registration
    - Add logging for topic registration events
    - _Requirements: 2.1, 2.3, 2.5_

- [x] 5. Enhance handler template with better patterns

  - [x] 5.1 Improve HTTP handler patterns

    - Add better error handling with proper HTTP status codes
    - Include request validation examples
    - Add user context extraction helpers
    - _Requirements: 5.4_

  - [x] 5.2 Add pubsub integration to handlers
    - Include examples of publishing messages from HTTP endpoints
    - Add proper message formatting and validation
    - Include error handling for publish operations
    - _Requirements: 1.4, 5.4_

- [x] 6. Add CLI flag support for template variations

  - [x] 6.1 Implement --minimal flag support

    - Add flag parsing to new_module command
    - Create minimal template variant with only Renderer dependency
    - Update dependency injection logic for minimal mode
    - _Requirements: 3.2_

  - [x] 6.2 Update default template behavior
    - Make pubsub and topic management the default
    - Include all communication dependencies by default
    - Add proper flag documentation
    - _Requirements: 3.1_

- [x] 7. Add commented examples for advanced integrations

  - [x] 7.1 Add database integration examples

    - Include commented Database dependency in Dependencies struct
    - Add examples of both store pattern and raw database access
    - Include transaction handling examples
    - _Requirements: 3.4, 5.5_

  - [x] 7.2 Add script engine integration examples

    - Include commented ScriptEngine dependency
    - Add examples of script execution in handlers
    - Include script configuration patterns
    - _Requirements: 3.4, 5.5_

  - [x] 7.3 Add presence service integration examples
    - Include commented PresenceService dependency
    - Add examples of presence tracking
    - Include presence event handling patterns
    - _Requirements: 3.5, 5.5_

- [x] 8. Update dependency injection and registration

  - [x] 8.1 Update dependencies.go template generation

    - Modify updateDependenciesFile to include new dependencies
    - Add proper import statements for pubsub and topicmgr
    - Include error handling for dependency injection
    - _Requirements: 1.1, 2.1_

  - [x] 8.2 Update modules.go registration
    - Ensure proper module registration with new dependencies
    - Add error handling for module registration
    - Include logging for module registration events
    - _Requirements: 1.1, 2.1_

- [x] 9. Add comprehensive documentation and examples

  - [x] 9.1 Create module README template

    - Add module-specific documentation template
    - Include usage examples and common patterns
    - Add troubleshooting section
    - _Requirements: 5.1, 5.3_

  - [x] 9.2 Enhance inline code documentation

    - Add comprehensive comments explaining pubsub patterns
    - Include TODO comments for common customization points
    - Add references to existing modules for examples
    - _Requirements: 5.1, 5.3, 5.5_

  - [x] 9.3 Update CLI help and success messages
    - Update command help text to reflect new capabilities
    - Enhance success messages with next steps
    - Add examples of generated code in output
    - _Requirements: 5.3_

- [ ]\* 10. Add testing templates and utilities

  - [ ]\* 10.1 Create module test template

    - Add module_test.go with lifecycle testing
    - Include mock dependencies for testing
    - Add integration test examples
    - _Requirements: 5.2_

  - [ ]\* 10.2 Create subscriber test template

    - Add subscriber_test.go with message processing tests
    - Include mock pubsub for testing
    - Add error handling test cases
    - _Requirements: 5.2_

  - [ ]\* 10.3 Create handler test template
    - Add handler_test.go with HTTP endpoint tests
    - Include table-driven test examples
    - Add request/response validation tests
    - _Requirements: 5.2_
