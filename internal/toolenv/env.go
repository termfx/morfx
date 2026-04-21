package toolenv

import (
	"fmt"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/internal/runtime"
	"github.com/oxhq/morfx/providers"
)

// Environment wires together providers and the file processor for standalone tools.
type Environment struct {
	providers     *providers.Registry
	fileProcessor *core.FileProcessor
}

// NewEnvironment constructs an Environment with all built-in language providers registered.
func NewEnvironment() (*Environment, error) {
	rt, err := runtime.Build(runtime.Config{})
	if err != nil {
		return nil, fmt.Errorf("build runtime: %w", err)
	}

	return &Environment{
		providers:     rt.Providers,
		fileProcessor: rt.FileProcessor,
	}, nil
}

// Provider returns the provider implementation for the requested language.
func (env *Environment) Provider(language string) (providers.Provider, error) {
	provider, ok := env.providers.Get(language)
	if !ok {
		return nil, fmt.Errorf("language not supported: %s", language)
	}
	return provider, nil
}

// Providers exposes the underlying registry for callers that need direct access.
func (env *Environment) Providers() *providers.Registry {
	return env.providers
}

// FileProcessor returns the configured file processor capable of multi-file operations.
func (env *Environment) FileProcessor() *core.FileProcessor {
	return env.fileProcessor
}
