# Implementation Plan

- [x] 1. Create core topic management interfaces and types

  - Implement the Topic interface with compile-time safety guarantees
  - Create TypedTopic struct with framework/module scoping support
  - Define TopicConfig and TopicScope types for configuration
  - Add error types for structured topic management error handling
  - _Requirements: 1.1, 1.2, 1.3, 6.5_

- [x] 2. Implement the Topic Manager with framework/module distinction

  - [x] 2.1 Create the core Manager struct with registry and validator

    - Implement Manager struct with thread-safe operations
    - Add DefineFramework and DefineModule functions for scoped topic creation
    - Implement Register, Get, List methods for topic management
    - _Requirements: 2.1, 2.2, 6.1, 6.2_

  - [x] 2.2 Add scoped topic discovery and filtering methods

    - Implement ListByModule for module-specific topic discovery
    - Implement ListByScope for framework/module filtering
    - Add ListFrameworkTopics convenience method
    - _Requirements: 2.4, 6.3_

  - [x] 2.3 Implement topic validation and error handling
    - Create Validator component for topic definition validation
    - Add Validate method for runtime topic usage validation
    - Implement structured error handling with TopicError types
    - _Requirements: 2.5, 6.5_

- [x] 3. Create backward compatibility layer with existing TopicRegistry

  - [x] 3.1 Implement compatibility wrapper for existing Topic interface

    - Create adapter that wraps new TypedTopic to satisfy old Topic interface
    - Ensure existing TopicRegistry.Get() calls continue to work
    - Maintain existing topic name resolution behavior
    - _Requirements: 5.1, 5.3_

  - [x] 3.2 Add migration utilities for gradual transition
    - Create Migrator struct for analyzing legacy topic usage
    - Implement AnalyzeLegacyUsage to scan for magic string patterns
    - Add GenerateTopicDefinitions to create typed definitions from legacy topics
    - _Requirements: 5.2, 5.4_

- [ ] 4. Integrate with existing pub/sub infrastructure

  - [ ] 4.1 Enhance Publisher interface with typed topic support

    - Add PublishToTopic method that accepts Topic interface
    - Maintain backward compatibility with existing Publish method
    - Ensure topic name consistency between old and new methods
    - _Requirements: 3.1, 3.4_

  - [ ] 4.2 Enhance Subscriber interface with typed topic support

    - Add SubscribeToTopic method that accepts Topic interface
    - Maintain backward compatibility with existing Subscribe method
    - Ensure consistent topic resolution across publisher and subscriber
    - _Requirements: 3.1, 3.4_

  - [ ] 4.3 Update WebSocket bridge to use typed topics
    - Modify Bridge to accept typed topics for HTML and Data endpoints
    - Update getEndpointTopics to use new topic management system
    - Ensure WebSocket topic routing continues to work correctly
    - _Requirements: 3.2, 3.3_

- [x] 5. Create framework topic definitions for core services

  - [x] 5.1 Define WebSocket framework topics

    - Create internal/websocket/topics.go with typed topic definitions
    - Convert TopicHTMLBroadcast, TopicHTMLDirect constants to typed topics
    - Convert TopicDataBroadcast, TopicDataDirect constants to typed topics
    - Update RegisterTopics function to use new topic management system
    - _Requirements: 6.1, 6.2_

  - [x] 5.2 Define presence service framework topics
    - Create internal/presence/topics.go for presence-related topics
    - Define UserOnline and UserOffline topics with proper metadata
    - Ensure topics follow framework naming conventions
    - _Requirements: 6.1, 6.4_

- [x] 6. Convert existing modules to use typed topics

  - [x] 6.1 Migrate chat module to typed topic system

    - Create internal/modules/chat/topics/topics.go package
    - Convert ClientMessageNew, Messages, Direct variables to typed topics
    - Update chat handlers to use typed topics instead of magic strings
    - Update chat subscriber to use typed topic system
    - _Requirements: 1.4, 3.3, 6.1_

  - [x] 6.2 Migrate wargame module to typed topic system
    - Create internal/modules/wargame/topics/topics.go package
    - Convert existing wargame topics to typed topic definitions
    - Update wargame engine and subscriber to use typed topics
    - Ensure wargame topic registration uses new system
    - _Requirements: 1.4, 3.3, 6.1_

- [ ] 7. Create CLI tools for topic management and debugging

  - [ ] 7.1 Implement topic listing and inspection commands

    - Create cmd/topics/list.go for listing all registered topics
    - Add filtering options by module, scope, and pattern matching
    - Display topic metadata including descriptions and examples
    - _Requirements: 4.1, 4.2, 4.4_

  - [ ] 7.2 Add topic validation and debugging utilities
    - Create cmd/topics/validate.go for topic validation commands
    - Add topic usage analysis and magic string detection
    - Implement topic resolution testing and debugging tools
    - _Requirements: 4.3, 4.5_

- [ ]\* 8. Add comprehensive testing for topic management system

  - [ ]\* 8.1 Create unit tests for core topic management functionality

    - Write tests for Topic interface implementations and type safety
    - Test Manager registration, lookup, and validation operations
    - Verify framework/module scoping and isolation
    - _Requirements: 1.1, 1.2, 2.1, 6.2_

  - [ ]\* 8.2 Add integration tests for pub/sub topic usage

    - Test typed topic usage with Publisher and Subscriber interfaces
    - Verify topic name consistency across publish/subscribe operations
    - Test WebSocket bridge integration with typed topics
    - _Requirements: 3.1, 3.4_

  - [ ]\* 8.3 Create migration and compatibility tests
    - Test backward compatibility with existing TopicRegistry usage
    - Verify gradual migration scenarios with mixed old/new usage
    - Test migration utilities and legacy topic analysis
    - _Requirements: 5.1, 5.2, 5.3_

- [ ] 9. Update application initialization to use new topic system

  - [ ] 9.1 Modify server startup to initialize topic management

    - Update server.go to create and configure topic Manager
    - Ensure framework topics are registered during startup
    - Initialize topic management before module registration
    - _Requirements: 2.1, 6.1_

  - [ ] 9.2 Update module registration to use typed topics
    - Modify module.Module interface to support topic registration
    - Update app/modules.go to pass topic Manager to modules
    - Ensure modules register their topics during initialization
    - _Requirements: 2.1, 6.1, 6.3_

- [ ] 10. Documentation and migration guide

  - [ ] 10.1 Create developer documentation for new topic system

    - Document how to define framework and module topics
    - Provide examples of typed topic usage in publishers and subscribers
    - Document CLI tools and debugging utilities
    - _Requirements: 2.3, 4.1, 6.4_

  - [ ] 10.2 Create migration guide for existing code
    - Document step-by-step migration process from magic strings
    - Provide examples of converting existing topics to typed definitions
    - Document backward compatibility features and limitations
    - _Requirements: 5.2, 5.4_
