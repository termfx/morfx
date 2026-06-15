package engine

import (
	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/providers"
)

func New(cfg Config) (*Engine, error) {
	cfg = normalizeConfig(cfg)

	rt, err := BuildRuntime(cfg)
	if err != nil {
		return nil, err
	}

	return &Engine{cfg: cfg, runtime: rt}, nil
}

func normalizeConfig(cfg Config) Config {
	if cfg.WriteMode == "" {
		cfg.WriteMode = WriteModePreview
	}
	return cfg
}

func (e *Engine) Providers() *providers.Registry {
	if e == nil || e.runtime == nil {
		return nil
	}
	return e.runtime.Providers
}

func (e *Engine) FileProcessor() *core.FileProcessor {
	if e == nil || e.runtime == nil {
		return nil
	}
	return e.runtime.FileProcessor
}
