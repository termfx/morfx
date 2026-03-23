package toolenv

import (
	"fmt"
	"os"

	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/providers"
	"github.com/oxhq/morfx/providers/golang"
	"github.com/oxhq/morfx/providers/javascript"
	"github.com/oxhq/morfx/providers/php"
	"github.com/oxhq/morfx/providers/python"
	"github.com/oxhq/morfx/providers/typescript"
)

// Environment wires together providers and the file processor for standalone tools.
type Environment struct {
	providers     *providers.Registry
	fileProcessor *core.FileProcessor
}

// NewEnvironment constructs an Environment with all built-in language providers registered.
func NewEnvironment() (*Environment, error) {
	registry := providers.NewRegistry()

	// Register built-in language providers.
	registry.Register(golang.New())
	registry.Register(javascript.New())
	registry.Register(typescript.New())
	registry.Register(php.New())
	registry.Register(python.New())

	adapter := &providerRegistryAdapter{registry: registry}
	fileProcessor := core.NewFileProcessor(adapter)

	// Ensure the transaction directory exists so file operations can log safely.
	if err := os.MkdirAll(".morfx/transactions", 0o755); err != nil {
		return nil, fmt.Errorf("create transaction log directory: %w", err)
	}

	return &Environment{
		providers:     registry,
		fileProcessor: fileProcessor,
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

// providerRegistryAdapter adapts providers.Registry to core.ProviderRegistry.
type providerRegistryAdapter struct {
	registry *providers.Registry
}

func (pra *providerRegistryAdapter) Get(language string) (core.Provider, bool) {
	provider, ok := pra.registry.Get(language)
	if !ok {
		return nil, false
	}
	return &providerAdapter{provider: provider}, true
}

// providerAdapter adapts providers.Provider to core.Provider.
type providerAdapter struct {
	provider providers.Provider
}

func (pa *providerAdapter) Language() string {
	return pa.provider.Language()
}

func (pa *providerAdapter) Query(source string, query core.AgentQuery) core.QueryResult {
	return pa.provider.Query(source, query)
}

func (pa *providerAdapter) Transform(source string, op core.TransformOp) core.TransformResult {
	return pa.provider.Transform(source, op)
}
