package engine

import (
	"github.com/oxhq/morfx/core"
	"github.com/oxhq/morfx/providers"
)

type Engine struct {
	cfg     Config
	runtime *Runtime
}

type Runtime struct {
	Providers     *providers.Registry
	FileProcessor *core.FileProcessor
}
