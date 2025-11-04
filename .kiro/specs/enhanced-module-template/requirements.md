# Requirements Document

## Introduction

The goby-cli's new_module command currently generates basic module scaffolding with minimal functionality. This enhancement will create a more comprehensive and useful template that includes pubsub integration, topic management, and follows established patterns from existing modules like chat and wargame. The enhanced template should provide developers with a solid foundation that demonstrates best practices and common integration patterns.

## Glossary

- **Module Template**: The code generation template used by the goby-cli new_module command
- **PubSub System**: The publish-subscribe messaging system used for inter-module communication
- **Topic Manager**: The service responsible for registering and managing message topics
- **Module Dependencies**: The services and components that a module requires to function
- **Background Subscriber**: A goroutine-based service that listens for and processes messages
- **Script Engine**: The embedded scripting system for extending module functionality
- **Presence Service**: The service that tracks user presence and activity

## Requirements

### Requirement 1

**User Story:** As a developer, I want the new_module template to include pubsub integration, so that I can easily build modules that communicate with other parts of the system.

#### Acceptance Criteria

1. WHEN generating a new module, THE Module Template SHALL include pubsub Publisher and Subscriber dependencies
2. THE Module Template SHALL generate example message handler registration code
3. THE Module Template SHALL include a background subscriber implementation with proper lifecycle management
4. THE Module Template SHALL demonstrate both publishing and subscribing to messages
5. THE Module Template SHALL include proper error handling for pubsub operations

### Requirement 2

**User Story:** As a developer, I want the template to include topic management integration, so that my module can register its own topics and follow the established topic organization pattern.

#### Acceptance Criteria

1. THE Module Template SHALL include Topic Manager dependency injection
2. THE Module Template SHALL generate a topics subdirectory with topic definitions
3. THE Module Template SHALL include example topic registration in the Register method
4. THE Module Template SHALL follow the established topic naming convention pattern
5. THE Module Template SHALL include topic validation and error handling

### Requirement 3

**User Story:** As a developer, I want the template to provide different complexity levels, so that I can choose between a minimal setup and a more comprehensive example.

#### Acceptance Criteria

1. THE Module Template SHALL generate pubsub and topic management integration by default
2. WHEN the --minimal flag is provided, THE Module Template SHALL generate only Renderer dependency
3. WHEN no flags are provided, THE Module Template SHALL include Publisher, Subscriber, and TopicMgr dependencies
4. THE Module Template SHALL include commented examples showing how to add Script Engine integration
5. THE Module Template SHALL include commented examples showing how to add Presence Service integration

### Requirement 4

**User Story:** As a developer, I want the template to include proper background service patterns, so that my module can handle asynchronous operations correctly.

#### Acceptance Criteria

1. THE Module Template SHALL generate a subscriber service with proper goroutine management
2. THE Module Template SHALL include context-based cancellation for background services
3. THE Module Template SHALL demonstrate proper shutdown handling in the Shutdown method
4. THE Module Template SHALL include logging for background service lifecycle events
5. THE Module Template SHALL follow the established subscriber pattern from existing modules

### Requirement 5

**User Story:** As a developer, I want the template to include practical examples and clear guidance, so that I can understand how to extend and customize the generated code.

#### Acceptance Criteria

1. THE Module Template SHALL include inline code comments explaining core pubsub and topic patterns
2. THE Module Template SHALL generate example message handlers demonstrating common use cases
3. THE Module Template SHALL include TODO comments indicating where developers should add their custom logic
4. THE Module Template SHALL provide examples of both HTTP endpoints and message-driven functionality
5. THE Module Template SHALL include commented code showing how to integrate advanced services when needed
