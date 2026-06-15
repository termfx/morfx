package engine

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
