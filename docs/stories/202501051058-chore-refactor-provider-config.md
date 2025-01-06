**Chore: Refactor Provider Configuration Structure**

**Summary**
The current configuration structure uses ModelConfigSet to group models by provider, but this creates a disconnect between provider-level settings (like API keys) and model-specific settings. We should refactor to a cleaner hierarchical structure.

---
**Current Structure**
```yaml
models:
  openai:
    gpt-4:
      api_key: "..."      # API key duplicated for each model
      temperature: 0.7
      max_tokens: 2000
    gpt-3.5-turbo:
      api_key: "..."      # Same API key duplicated
      temperature: 0.9
      max_tokens: 1000
```

**Proposed Structure**
```yaml
providers:
  openai:
    api_key: "..."        # Provider-level settings
    base_url: "https://api.openai.com/v1"
    models:
      gpt-4:              # Model-specific settings
        temperature: 0.7
        max_tokens: 2000
      gpt-3.5-turbo:
        temperature: 0.9
        max_tokens: 1000
```

---
**Changes Required**

1. Configuration Types:
   ```go
   // New types
   type ProviderConfig struct {
       APIKey   string                    `yaml:"api_key"`
       BaseURL  string                    `yaml:"base_url"`
       Models   map[string]ModelConfig    `yaml:"models"`
   }

   type Config struct {
       Providers map[string]ProviderConfig `yaml:"providers"`
       // ... other existing fields
   }

   // Update ModelConfig to remove provider-level settings
   type ModelConfig struct {
       Temperature float64 `yaml:"temperature"`
       MaxTokens   int     `yaml:"max_tokens"`
       TopP        float64 `yaml:"top_p"`
   }
   ```

2. Registry Changes:
   * Update model lookup to use new structure
   * Remove defaultProvider parameter
   * Add helper methods for provider discovery

3. Provider Factory Updates:
   * Modify to use provider-level settings
   * Pass only model-specific settings to model initialization

4. Migration Support:
   * Add config validation
   * Provide upgrade path for existing configs

---
**Benefits**

1. Better Organization:
   * Provider settings in one place
   * No duplication of API keys
   * Clear separation of concerns
   * More maintainable configuration

2. Improved Code:
   * More logical structure
   * Better encapsulation
   * Easier to add new provider settings
   * No need for default provider concept

---
**Testing Strategy**

1. Unit Tests:
   * Config parsing and validation
   * Provider/model lookup
   * Settings inheritance

2. Integration Tests:
   * Provider initialization
   * Model configuration
   * Error handling

3. Migration Tests:
   * Config upgrade path
   * Validation errors
   * Backward compatibility

---
This refactoring will improve the configuration structure by properly separating provider and model concerns, while eliminating the need for duplicated settings and default providers.
