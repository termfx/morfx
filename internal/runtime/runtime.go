package runtime

import "github.com/oxhq/morfx/engine"

// Config controls shared runtime construction.
type Config struct {
	TransactionLogDir string
}

// Runtime contains the shared provider registry and file processor.
//
// Deprecated: use `engine.Runtime`.
type Runtime = engine.Runtime

// Build constructs the shared Morfx runtime used by MCP and standalone tools.
//
// Deprecated: use `engine.BuildRuntime`.
func Build(cfg Config) (*Runtime, error) {
	return engine.BuildRuntime(engine.Config{
		TransactionLogDir: cfg.TransactionLogDir,
	})
}
