# Implementation Plan

- [x] 1. Create shared topic utilities

  - Extract topic initialization logic from standalone CLI into reusable utility
  - Create output formatting functions that match existing goby-cli patterns
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 1.1 Create topic initializer utility

  - Write `cmd/goby-cli/internal/topics/initializer.go` with topic setup logic
  - Extract initialization code from `cmd/topics/main.go`
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 1.2 Create topic formatting utilities

  - Write `cmd/goby-cli/internal/topics/formatter.go` with table and JSON output functions
  - Extract and adapt formatting code from standalone CLI
  - _Requirements: 2.4, 3.2, 3.3_

- [x] 2. Implement topics parent command

  - Create base topics command structure following goby-cli patterns
  - Set up subcommand registration and help documentation
  - _Requirements: 1.1, 1.4_

- [x] 2.1 Create topics parent command file

  - Write `cmd/goby-cli/cmd/topics.go` with parent command definition
  - Add topics subcommand group to root command registration
  - _Requirements: 1.1, 1.4_

- [x] 3. Implement topics list subcommand

  - Create list command with filtering options and output formatting
  - Support module and scope filtering with proper flag handling
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 3.1 Create topics list command file

  - Write `cmd/goby-cli/cmd/topics_list.go` with list functionality
  - Implement filtering logic for module and scope options
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 4. Implement topics get subcommand

  - Create get command for detailed topic information display
  - Handle topic lookup and error cases appropriately
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 4.1 Create topics get command file

  - Write `cmd/goby-cli/cmd/topics_get.go` with detailed topic display
  - Implement topic lookup and detailed formatting
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 5. Implement topics validate subcommand

  - Create validate command for topic definition validation
  - Provide clear success and error feedback with proper formatting
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 5.1 Create topics validate command file

  - Write `cmd/goby-cli/cmd/topics_validate.go` with validation functionality
  - Implement validation logic and user-friendly output formatting
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 6. Update root command integration

  - Update root command help text to include topics functionality
  - Ensure consistent CLI experience across all commands
  - _Requirements: 1.2, 1.3, 1.4_

- [x] 6.1 Update root command documentation

  - Modify `cmd/goby-cli/cmd/root.go` to include topics in help text
  - Ensure topics commands are properly registered and discoverable
  - _Requirements: 1.2, 1.3, 1.4_

- [ ]\* 7. Add comprehensive testing

  - Write unit tests for topic utilities and command handlers
  - Create integration tests for full command execution scenarios
  - _Requirements: All requirements validation_

- [ ]\* 7.1 Write unit tests for utilities

  - Test topic initialization and formatting functions
  - Verify error handling and edge cases
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [ ]\* 7.2 Write integration tests for commands
  - Test all topics subcommands with various flag combinations
  - Verify output formats and error conditions
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 3.4, 4.1, 4.2, 4.3, 4.4_
