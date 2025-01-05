package registry

import (
	"fmt"
	"strings"
	"sync"

	"github.com/butter-bot-machines/skylark/pkg/provider"
)

// Factory creates provider instances
type Factory func(model string) (provider.Provider, error)

// Registry manages provider factories and instances
type Registry struct {
	mu        sync.RWMutex
	factories map[string]Factory
}

// New creates a new provider registry
func New() *Registry {
	return &Registry{
		factories: make(map[string]Factory),
	}
}

// Register adds a provider factory
func (r *Registry) Register(name string, factory Factory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[name] = factory
}

// CreateForModel creates a provider for a model specification
// Model spec can be either:
// - "model-name" (uses default provider)
// - "provider:model-name" (uses specified provider)
func (r *Registry) CreateForModel(modelSpec string, defaultProvider string) (provider.Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Parse model spec
	providerName, modelName := ParseModelSpec(modelSpec)
	if providerName == "" {
		providerName = defaultProvider
	}

	// Get factory
	factory, ok := r.factories[providerName]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	// Create provider
	return factory(modelName)
}

// ParseModelSpec parses a model specification into provider and model names
// Returns ("", model) if no provider is specified
func ParseModelSpec(spec string) (provider, model string) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", spec
}
