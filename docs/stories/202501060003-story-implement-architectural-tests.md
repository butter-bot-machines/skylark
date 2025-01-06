# Architectural Tests & Pre-Commit Hooks

## Context

The project has a robust test infrastructure in test/suite but needs architectural validation:

1. Current Test Infrastructure:
   - test/suite provides test organization with hooks
   - Support for categories and tags
   - Timeout handling and cleanup
   - Test data helpers

2. Areas Needing Implementation:
   - No architectural validation tests
   - No pre-commit hooks
   - pkg/processor needs cleanup
   - No structure validation

## Goals

1. Implement Architectural Tests:
   - Validate package dependencies
   - Check interface compliance
   - Enforce structure rules
   - Monitor complexity

2. Add Pre-Commit Hooks:
   - Code formatting
   - Lint validation
   - Test coverage
   - Commit message format

3. Clean Up Processor Package:
   - Improve separation of concerns
   - Enhance error handling
   - Add comprehensive tests
   - Reduce complexity

## Technical Details

### Architectural Tests
- Create new test suite for architectural validation:
  ```go
  func TestArchitecture(t *testing.T) {
      suite := suite.NewSuite()

      // Package dependency tests
      suite.Add("package dependencies", func(t *testing.T) {
          // Verify no circular dependencies
          // Check allowed imports
          // Validate layer boundaries
      }, suite.WithCategory("architecture"))

      // Interface compliance tests
      suite.Add("interface compliance", func(t *testing.T) {
          // Verify provider implementations
          // Check processor interfaces
          // Validate tool contracts
      }, suite.WithCategory("architecture"))

      // Structure tests
      suite.Add("project structure", func(t *testing.T) {
          // Check file organization
          // Validate naming conventions
          // Verify documentation
      }, suite.WithCategory("architecture"))

      // Complexity tests
      suite.Add("code complexity", func(t *testing.T) {
          // Check cyclomatic complexity
          // Monitor function size
          // Validate package cohesion
      }, suite.WithCategory("architecture"))

      suite.Run(t)
  }
  ```

### Pre-Commit Hooks
- Create .git/hooks/pre-commit:
  ```bash
  #!/bin/bash
  
  # Run formatters
  gofmt -w .
  
  # Run linters
  golangci-lint run
  
  # Run tests
  go test -v ./...
  
  # Check coverage
  go test -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out | grep total
  
  # Validate commit message
  commit_msg=$(cat $1)
  if ! echo "$commit_msg" | grep -qE "^(feat|fix|chore|docs|test|refactor): "; then
      echo "Invalid commit message format"
      exit 1
  fi
  ```

### Processor Cleanup
- Update pkg/processor/concrete/processor.go:
  ```go
  // Split into smaller interfaces
  type CommandProcessor interface {
      Process(cmd *parser.Command) (string, error)
  }

  type FileProcessor interface {
      ProcessFile(path string) error
      ProcessDirectory(dir string) error
  }

  type ResponseHandler interface {
      HandleResponse(cmd *parser.Command, response string) error
      UpdateFile(path string, responses []Response) error
  }

  // Improve error handling
  func (p *processorImpl) Process(cmd *parser.Command) (string, error) {
      if cmd == nil {
          return "", fmt.Errorf("%w: nil command", ErrInvalidInput)
      }

      assistant, err := p.assistants.Get(cmd.Assistant)
      if err != nil {
          return "", fmt.Errorf("%w: failed to get assistant: %v", ErrProcessing, err)
      }

      response, err := assistant.Process(cmd)
      if err != nil {
          return "", fmt.Errorf("%w: failed to process command: %v", ErrProcessing, err)
      }

      return response, nil
  }
  ```

## Implementation Plan

1. Architectural Tests
   - Create test/architecture package
   - Implement dependency validation
   - Add interface compliance checks
   - Create structure validation

2. Pre-Commit Hooks
   - Set up git hooks
   - Configure linters
   - Add coverage checks
   - Implement message validation

3. Processor Cleanup
   - Split into interfaces
   - Improve error handling
   - Add comprehensive tests
   - Document changes

## Acceptance Criteria

- [ ] Architectural tests verify:
  - No circular dependencies
  - Interface compliance
  - Project structure
  - Code complexity

- [ ] Pre-commit hooks enforce:
  - Code formatting
  - Lint rules
  - Test coverage
  - Commit messages

- [ ] Processor package has:
  - Clear interfaces
  - Proper error handling
  - Comprehensive tests
  - Documentation

## Dependencies

- test/suite package
- pkg/processor implementation
- Git hooks support

## Related

- test/suite/suite.go implementation
- pkg/processor/concrete/processor.go
- Current test infrastructure
