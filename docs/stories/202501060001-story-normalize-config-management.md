# Config Management Normalization

## Context

The current configuration system has several areas that need normalization:

1. Config Structure:

   - config.Config uses a flat structure with ModelConfigSet mapping provider->model->settings
   - Recent bug fix (202501051057) addressed assistant-specific configurations
   - Proposed refactoring (202501051058) suggests hierarchical provider structure

2. Current Implementation:
   - pkg/config/config.go defines core configuration types
   - pkg/processor/concrete/processor.go handles provider initialization
   - Assistant configurations live in .skai/assistants/<name>/prompt.md
   - Logging configuration is currently hardcoded in processor initialization

## Goals

1. Normalize Configuration Structure:

   - Implement hierarchical provider configuration as proposed in 202501051058
   - Move API keys to provider level
   - Consolidate model settings under providers

2. Standardize Assistant Configuration:

   - Validate assistant metadata in prompt.md front-matter
   - Implement proper model fallback chain:
     a. Assistant-specific overrides
     b. Provider defaults
     c. Global defaults

3. Integrate Logging Control:
   - Move hardcoded logging.Options to config.yaml
   - Support component-specific log levels
   - Add log file configuration

## Technical Details

### Configuration Restructuring

- Update pkg/config/config.go:

  ```go
  type Config struct {
      Version     string                     `yaml:"version"`
      Environment EnvironmentConfig          `yaml:"environment"`
      Providers   map[string]ProviderConfig  `yaml:"providers"`  // New structure
      Tools       map[string]ToolConfig      `yaml:"tools"`
      Workers     WorkerConfig               `yaml:"workers"`
      FileWatch   FileWatchConfig            `yaml:"file_watch"`
      WatchPaths  []string                   `yaml:"watch_paths"`
      Security    types.SecurityConfig       `yaml:"security"`
      Logging     LoggingConfig             `yaml:"logging"`    // New section
  }

  type ProviderConfig struct {
      APIKey   string                    `yaml:"api_key"`
      BaseURL  string                    `yaml:"base_url"`
      Models   map[string]ModelConfig    `yaml:"models"`
  }

  type LoggingConfig struct {
      Level     string            `yaml:"level"`
      File      string            `yaml:"file"`
      AddSource bool              `yaml:"add_source"`
      Components map[string]string `yaml:"components"`
  }
  ```

### Assistant Configuration

- Update pkg/processor/concrete/processor.go:

  ```go
  // Update provider initialization
  reg.Register("openai", func(model string) (provider.Provider, error) {
      providerConfig, ok := cfg.Providers["openai"]
      if !ok {
          return nil, fmt.Errorf("OpenAI provider configuration not found")
      }

      modelConfig, ok := providerConfig.Models[model]
      if !ok {
          return nil, fmt.Errorf("Model configuration not found: %s", model)
      }

      return openai.New(model, modelConfig, openai.Options{
          APIKey:  providerConfig.APIKey,
          BaseURL: providerConfig.BaseURL,
      })
  })
  ```

### Logging Integration

- Update logging initialization in processor:
  ```go
  func init() {
      cfg := config.GetConfig()
      logOpts := &logging.Options{
          Level:     cfg.Logging.Level,
          File:      cfg.Logging.File,
          AddSource: cfg.Logging.AddSource,
      }
      logger = logging.NewLogger(logOpts)
  }
  ```

## Implementation Plan

1. Update Configuration Types

   - Implement new Config struct with providers section
   - Add logging configuration section
   - Update validation logic

2. Update Provider Initialization

   - Modify provider registry to use new structure
   - Update provider factory functions
   - Add provider-level configuration handling

3. Implement Logging Control

   - Add logging configuration parsing
   - Update logger initialization
   - Add component-specific logging

4. Migration Support
   - Add config version check
   - Implement migration for existing configs
   - Update documentation

## Acceptance Criteria

- [ ] New configuration structure implemented in pkg/config/config.go
- [ ] Provider initialization uses hierarchical config
- [ ] Assistant configurations properly override provider defaults
- [ ] Logging configuration works through config.yaml
- [ ] Component-specific log levels function correctly
- [ ] All tests pass with new structure
- [ ] Migration path exists for existing configs
- [ ] Documentation updated to reflect changes

## Dependencies

- pkg/config package
- pkg/processor/concrete/processor.go
- pkg/logging package
- pkg/provider implementations

## Related

- Bug fix: 202501051057-bug-issue-with-assistant-config.md
- Refactor: 202501051058-chore-refactor-provider-config.md
- pkg/config implementation
- pkg/processor implementation
- pkg/logging implementation
