# Implementation Plan

- [ ] 1. Set up Cobra framework foundation

  - Create new `cmd/goby` directory structure
  - Add Cobra dependency to go.mod
  - Implement root command with basic structure
  - Set up command registration system
  - _Requirements: 1.1, 1.4_

- [ ] 2. Implement server command functionality

  - [ ] 2.1 Create serve command structure

    - Implement `cmd/goby/cmd/serve.go` with Cobra command definition
    - Set up flag definitions matching current server flags
    - _Requirements: 2.2, 2.3_

  - [ ] 2.2 Migrate server startup logic

    - Move server initialization code from `cmd/server/main.go`
    - Preserve all configuration loading and validation
    - Maintain signal handling and graceful shutdown
    - _Requirements: 2.1, 2.3_

  - [ ] 2.3 Configure default command behavior
    - Set server as default action when no subcommand specified
    - Implement backward compatibility for direct execution
    - _Requirements: 2.1_

- [ ] 3. Implement script management commands

  - [ ] 3.1 Create scripts parent command

    - Implement `cmd/goby/cmd/scripts/scripts.go` as parent command
    - Set up subcommand registration for script operations
    - _Requirements: 3.1_

  - [ ] 3.2 Migrate script extraction functionality

    - Move `handleScriptExtraction` logic to `cmd/goby/cmd/scripts/extract.go`
    - Preserve all current flags and behavior
    - Implement backward compatibility for `--extract-scripts` flag
    - _Requirements: 3.2, 6.1, 6.2_

  - [ ] 3.3 Implement script listing command

    - Create `cmd/goby/cmd/scripts/list.go` for displaying available scripts
    - Show script metadata including module, language, and source
    - Support both human-readable and structured output formats
    - _Requirements: 3.3_

  - [ ] 3.4 Implement script validation command
    - Create `cmd/goby/cmd/scripts/validate.go` for syntax checking
    - Validate script syntax using existing script engine
    - Report validation errors with helpful messages
    - _Requirements: 3.4_

- [ ] 4. Implement topic management commands

  - [ ] 4.1 Create topics parent command

    - Implement `cmd/goby/cmd/topics/topics.go` as parent command
    - Set up subcommand registration for topic operations
    - _Requirements: 4.1_

  - [ ] 4.2 Migrate topic listing functionality

    - Move topic list logic from `cmd/topics/main.go` to `cmd/goby/cmd/topics/list.go`
    - Preserve identical tabular output format
    - _Requirements: 4.2, 4.4_

  - [ ] 4.3 Migrate topic detail functionality
    - Move topic get logic from `cmd/topics/main.go` to `cmd/goby/cmd/topics/get.go`
    - Maintain identical functionality and output
    - _Requirements: 4.3_

- [ ] 5. Implement help and documentation system

  - [ ] 5.1 Configure comprehensive help system

    - Set up Cobra help templates with consistent formatting
    - Add usage examples for complex commands
    - Implement version command with build information
    - _Requirements: 5.1, 5.2, 5.3, 5.5_

  - [ ] 5.2 Add command completion support
    - Implement shell completion for bash, zsh, and fish
    - Add completion for command arguments where applicable
    - _Requirements: 5.4_

- [ ] 6. Implement backward compatibility layer

  - [ ] 6.1 Add legacy flag support

    - Map existing flags to new command structure
    - Implement deprecation warnings for legacy usage
    - Ensure identical behavior for existing flag combinations
    - _Requirements: 6.1, 6.2, 6.3_

  - [ ] 6.2 Preserve exit codes and error messages
    - Maintain identical exit codes for all existing functionality
    - Preserve error message formats for script compatibility
    - _Requirements: 6.4_

- [ ] 7. Create CLI utilities and helpers

  - [ ] 7.1 Implement output formatting utilities

    - Create `cmd/goby/internal/output.go` for consistent formatting
    - Support JSON/YAML output options for machine consumption
    - Implement progress indicators for long operations
    - _Requirements: 1.4_

  - [ ] 7.2 Create configuration helpers
    - Implement `cmd/goby/internal/config.go` for CLI-specific configuration
    - Handle configuration precedence (env vars, files, flags, defaults)
    - _Requirements: 1.4_

- [ ] 8. Update build and deployment configuration

  - [ ] 8.1 Update build scripts and Makefiles

    - Modify build configuration to produce `goby` binary
    - Update any deployment scripts or Docker configurations
    - Create symlinks or aliases for backward compatibility if needed
    - _Requirements: 2.4_

  - [ ] 8.2 Update documentation
    - Create migration guide for existing users
    - Update CLI reference documentation
    - Add usage examples for new command structure
    - _Requirements: 5.1, 5.3_

- [ ]\* 9. Comprehensive testing

  - [ ]\* 9.1 Unit tests for command implementations

    - Write tests for each command's logic and flag handling
    - Test error scenarios and edge cases
    - Verify output formatting consistency
    - _Requirements: All_

  - [ ]\* 9.2 Integration tests for backward compatibility

    - Test all existing command invocations work identically
    - Verify environment variable handling
    - Test configuration file compatibility
    - _Requirements: 6.1, 6.2, 6.4_

  - [ ]\* 9.3 End-to-end testing
    - Test complete workflows using new CLI structure
    - Verify cross-platform compatibility
    - Test shell completion functionality
    - _Requirements: 1.1, 1.5, 5.4_

- [ ] 10. Deprecation and cleanup preparation

  - [ ] 10.1 Add deprecation warnings

    - Implement warnings for legacy binary usage
    - Add migration hints in deprecation messages
    - Plan timeline for legacy binary removal
    - _Requirements: 6.3_

  - [ ] 10.2 Prepare migration documentation
    - Document breaking changes and migration steps
    - Create examples showing old vs new command syntax
    - Prepare release notes highlighting CLI improvements
    - _Requirements: 5.1_
