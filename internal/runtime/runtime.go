package runtime

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

// Config controls shared runtime construction.
type Config struct {
	TransactionLogDir string
}

// Runtime contains the shared provider registry and file processor.
type Runtime struct {
	Providers     *providers.Registry
	FileProcessor *core.FileProcessor
}

// Build constructs the shared Morfx runtime used by MCP and standalone tools.
func Build(cfg Config) (*Runtime, error) {
	registry := providers.NewRegistry()
	registerBuiltInProviders(registry)

	fileProcessor := core.NewFileProcessor(&providerRegistryAdapter{registry: registry})

	logDir := cfg.TransactionLogDir
	if logDir == "" {
		logDir = ".morfx/transactions"
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return nil, fmt.Errorf("create transaction log directory: %w", err)
	}
	fileProcessor.SetTransactionLogDir(logDir)

	return &Runtime{
		Providers:     registry,
		FileProcessor: fileProcessor,
	}, nil
}

func registerBuiltInProviders(registry *providers.Registry) {
	registry.Register(golang.New())
	registry.Register(javascript.New())
	registry.Register(typescript.New())
	registry.Register(php.New())
	registry.Register(python.New())
}

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
