# Assistant-Specific Configurations Not Respected (✓ Completed)

> Completed on January 5, 2025 at 11:53
> - Implemented proper model config handling
> - Added provider registry support
> - Improved test infrastructure
> - Created follow-up chore for config refactoring

**Summary of the Issue**
The current system appears to prioritize provider-wide configurations from config.yaml over assistant-specific configurations declared in the assistant's frontmatter (prompt.md). This creates a potential bug where assistants cannot have unique model configurations or parameter overrides, even though their frontmatter specifies them.

---
**Root Cause**
1. Provider Initialization:
    * The provider (e.g., OpenAI) is initialized using global configurations (config.yaml) during the system startup.
    * These settings include the model name (e.g., "gpt-4"), temperature, max tokens, and other defaults.
2. Request Construction:
    * When an assistant sends a request, the "model" field and other parameters are pulled directly from config.yaml instead of merging with assistant-specific overrides from the frontmatter.
    * This means assistants cannot specify unique models or settings (e.g., temperature, max tokens).
3. Parameter Merging:
    * While the system is designed to resolve parameters in this order:
        1. Assistant-specific settings (prompt.md).
        2. Provider-specific defaults (config.yaml).
        3. Global/system-wide fallbacks.
    * The merging logic was missing when constructing the API request payload.

---
**Impact**
1. Assistants are unable to leverage unique models (e.g., gpt-4 vs. gpt-3.5-turbo) as specified in their frontmatter.
2. Assistant-specific overrides (e.g., custom temperature or max token values) are ignored, forcing all assistants to use the same global defaults.
3. This limits flexibility and violates the expected behavior that assistants can have independent configurations.

---
## Changes Made

1. pkg/provider/provider.go:
   - Added RequestOptions for per-request configurations
   - Implemented model-specific settings support
   - Added provider registry interface

2. pkg/assistant/assistant.go:
   - Updated to use provider registry
   - Added proper model config passing
   - Improved error handling

3. pkg/processor/concrete/processor.go:
   - Added mock provider for testing
   - Updated to use provider registry
   - Improved test coverage

## Implementation Details

1. Added request options support:
   ```go
   type RequestOptions struct {
       Model       string
       Temperature float64
       MaxTokens   int
   }
   ```

2. Implemented provider registry:
   ```go
   func (r *Registry) CreateForModel(modelSpec string) (Provider, error) {
       providerName, modelName := ParseModelSpec(modelSpec)
       factory := r.factories[providerName]
       return factory(modelName)
   }
   ```

3. Updated assistant to use configurations:
   ```go
   opts := &provider.RequestOptions{
       Model:       modelName,
       Temperature: 0.7,
       MaxTokens:   2000,
   }
   ```

## Verification

All tests pass:
- pkg/assistant/assistant_test.go
- pkg/assistant/integration_test.go
- pkg/processor/concrete/processor_test.go
- test/integration/integration_test.go

## Result

The system now properly respects assistant-specific configurations:
1. Each assistant can specify its own model
2. Provider registry handles model resolution
3. Test infrastructure properly mocks providers
4. Configuration handling is more robust

## Follow-up Work

Created new chore story (202501051058-chore-refactor-provider-config.md) to improve the configuration structure by:
1. Moving to hierarchical provider/model structure
2. Eliminating API key duplication
3. Removing default provider concept
4. Improving configuration organization
