# TODO: Punchlist Items

## Primary Tasks

1. Config Management Normalization
   - Ensure consistency in assistant metadata handling (model fallback, defaults, config.yaml structure).
   - Validate existing ConfigManagement implementation and normalize behavior.
   - Introduce logging control in config.yaml as the primary method for setting log levels.
2. Reference Context Validation
   - Validate functionality to reference and extract contextual blocks from documentation (within the parser).
   - Investigate adding a mapping package for extracting and sending referenced blocks to the provider.
3. Architectural Tests & Pre-Commit Hooks
   - Implement architectural tests to ensure project structure cleanliness and maintainability.
   - Add pre-commit hooks to enforce standards and catch issues early.
   - Clean up /pkg/processor (high-priority chore).

## Secondary Tasks

4. Logging Enhancements
   - Start with logging controls in config.yaml.
   - Consider adding CLI overrides later for flexibility (e.g., bumping to debug during watch or run commands).
5. Documentation-Based Assistant
   - Build the framework for a !help command or assistant creation via skai init.
   - Optional flag: --llm for environment-specific configurations.
   - Leverage validated reference context mapping for this functionality.
6. Watcher Feedback Improvements
   - Confirm usability of watcher updates in VSCode.
   - Adjust default settings if necessary.
7. Dynamic Source Compilation Enhancements
   - Maintain current behavior of dynamically adding source files for compilation.
   - Integrate with MCP and OAS-provider for richer functionality.
