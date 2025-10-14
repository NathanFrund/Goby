# Go Style Guide for Goby

This document outlines the coding standards and conventions used in the Goby project. Following these guidelines helps maintain consistency and readability across the codebase.

## Table of Contents
- [General Principles](#general-principles)
- [Naming Conventions](#naming-conventions)
- [Code Organization](#code-organization)
- [Error Handling](#error-handling)
- [Documentation](#documentation)
- [Testing](#testing)
- [Dependencies](#dependencies)
- [Commit Messages](#commit-messages)

## General Principles

1. **Clarity over brevity**: Favor clear, descriptive names over short, ambiguous ones.
2. **Consistency**: Follow existing patterns in the codebase.
3. **Simplicity**: Keep functions and types focused on a single responsibility.
4. **Idiomatic Go**: Follow the principles outlined in [Effective Go](https://golang.org/doc/effective_go.html).

## Naming Conventions

### Acronyms and Initialisms

- Use all uppercase for acronyms and initialisms in names (e.g., `URL`, `HTTP`, `ID`, `DBURL`).
- Examples:
  - ✅ `userID` (not `userId`)
  - ✅ `apiURL` (not `apiUrl`)
  - ✅ `httpServer` (not `HTTPServer`)
  - ✅ `dbConn` (not `DBConn`)

### Packages

- Use lowercase, single-word names.
- Avoid underscores or mixedCaps.
- Examples:
  - ✅ `storage`
  - ✅ `auth`
  - ❌ `storageHandler`
  - ❌ `auth_handler`

### Files

- Use snake_case for file names.
- Examples:
  - ✅ `file_handler.go`
  - ✅ `user_repository.go`
  - ❌ `filehandler.go`
  - ❌ `UserRepository.go`

### Functions and Methods

- Use MixedCaps (camelCase for unexported, PascalCase for exported).
- Be consistent with verb-noun naming for methods that perform actions.
- Examples:
  - ✅ `GetUserByID` (exported)
  - ✅ `validateInput` (unexported)
  - ✅ `SaveToDatabase` (exported)
  - ❌ `get_user_by_id`
  - ❌ `Validate_Input`

### Variables

- Use MixedCaps (camelCase for unexported, PascalCase for exported).
- Keep names concise but meaningful.
- Examples:
  - ✅ `userCount`
  - ✅ `maxRetries`
  - ✅ `DefaultTimeout` (exported)
  - ❌ `usercount`
  - ❌ `MAX_RETRIES`

### Interfaces

- Single-method interfaces should be named with an `-er` suffix.
- Multi-method interfaces should describe their general behavior.
- Examples:
  - ✅ `Reader` (single method)
  - ✅ `FileRepository` (multiple methods)
  - ❌ `IReader` (no Hungarian notation)

## Code Organization

### Imports

- Group imports into standard library, third-party, and local imports, separated by a blank line.
- Use the `goimports` tool to automatically format imports.

```go
import (
    // Standard library
    "context"
    "fmt"
    "net/http"

    // Third-party
    "github.com/labstack/echo/v4"
    "github.com/sirupsen/logrus"

    // Local
    "github.com/nfrund/goby/internal/domain"
)
```

### File Structure

- Follow this general structure within each file:
  1. Package declaration
  2. Imports
  3. Constants
  4. Variables
  5. Types
  6. Functions and Methods
  7. Tests (in `_test.go` files)

## Error Handling

- Always handle errors explicitly; never discard them using `_` unless you have a good reason.
- Use `fmt.Errorf` with `%w` to wrap errors with additional context.
- Create custom error types for expected error conditions.
- Examples:
  ```go
  if err != nil {
      return fmt.Errorf("failed to process user %s: %w", userID, err)
  }
  ```

## Documentation

- Document all exported functions, types, and packages.
- Use complete sentences with proper punctuation.
- Follow the Go doc comment style (start with the name being documented).
- Examples:
  ```go
  // User represents a user in the system.
  // It contains the user's ID, name, and email address.
  type User struct {
      ID    string
      Name  string
      Email string
  }
  
  // GetUserByID retrieves a user by their unique identifier.
  // Returns an error if the user is not found or if there's a database error.
  func GetUserByID(id string) (*User, error) {
      // ...
  }
  ```

## Testing

- Table-driven tests are preferred for testing multiple cases.
- Test files should be named `*_test.go`.
- Test functions should be named `TestXxx` where `Xxx` describes what's being tested.
- Use the `testify/assert` package for assertions.
- Example:
  ```go
  func TestAdd(t *testing.T) {
      tests := []struct {
          name     string
          a, b     int
          expected int
      }{
          {"positive numbers", 2, 3, 5},
          {"negative numbers", -1, -1, -2},
          {"zero", 0, 0, 0},
      }

      for _, tt := range tests {
          t.Run(tt.name, func(t *testing.T) {
              result := Add(tt.a, tt.b)
              assert.Equal(t, tt.expected, result)
          })
      }
  }
  ```

## Dependencies

- Use Go modules for dependency management.
- Keep dependencies to a minimum.
- Regularly update dependencies to their latest compatible versions.
- Document any non-obvious dependencies in the relevant package's documentation.

## Commit Messages

Follow the [Conventional Commits](https://www.conventionalcommits.org/) specification:

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types:
- `feat`: A new feature
- `fix`: A bug fix
- `docs`: Documentation only changes
- `style`: Changes that do not affect the meaning of the code
- `refactor`: A code change that neither fixes a bug nor adds a feature
- `perf`: A code change that improves performance
- `test`: Adding missing tests or correcting existing tests
- `chore`: Changes to the build process or auxiliary tools

### Examples:
```
feat(auth): add password reset functionality

Adds a new password reset flow that allows users to reset their password via email.

Closes #123
```

```
fix(api): prevent race condition in user creation

Fixes a potential race condition that could occur when multiple users try to register with the same email address.

Closes #124
```

## Linting and Formatting

- Use `gofmt` or `goimports` to format your code.
- Run `golangci-lint` locally before pushing changes.
- Fix all linter warnings before submitting a pull request.

## Review Process

1. Make sure all tests pass.
2. Ensure new code has appropriate test coverage.
3. Update documentation as needed.
4. Create a pull request with a clear description of the changes.
5. Request reviews from at least one other developer.

## Additional Resources

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)

---
*Last updated: October 2023*
